package groupsync

import (
	"context"
	"fmt"
	"time"

	userv1 "github.com/openshift/api/user/v1"
	redhatcopv1alpha1 "github.com/redhat-cop/group-sync-operator/pkg/apis/redhatcop/v1alpha1"
	"github.com/redhat-cop/group-sync-operator/pkg/controller/constants"
	"github.com/redhat-cop/group-sync-operator/pkg/controller/syncer"
	"github.com/redhat-cop/operator-utils/pkg/util"
	"k8s.io/apimachinery/pkg/api/errors"
	kapierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	controllerName = "groupsync-controller"
)

var log = logf.Log.WithName("controller_groupsync")

// Add creates a new GroupSync Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileGroupSync{
		ReconcilerBase: util.NewReconcilerBase(mgr.GetClient(), mgr.GetScheme(), mgr.GetConfig(), mgr.GetEventRecorderFor(controllerName)),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("groupsync-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource GroupSync
	err = c.Watch(&source.Kind{Type: &redhatcopv1alpha1.GroupSync{}}, &handler.EnqueueRequestForObject{}, util.ResourceGenerationOrFinalizerChangedPredicate{})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileGroupSync implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileGroupSync{}

// ReconcileGroupSync reconciles a GroupSync object
type ReconcileGroupSync struct {
	util.ReconcilerBase
}

// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileGroupSync) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling GroupSync")

	// Fetch the GroupSync instance
	instance := &redhatcopv1alpha1.GroupSync{}
	err := r.GetClient().Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Get Group Sync Manager
	groupSyncMgr, err := syncer.GetGroupSyncMgr(instance, r.ReconcilerBase)

	if err != nil {
		return r.ManageError(instance, err)
	}

	// Set Defaults
	if changed := groupSyncMgr.SetDefaults(); changed {
		err := r.GetClient().Update(context.TODO(), instance)
		if err != nil {
			log.Error(err, "unable to update instance", "instance", instance)
			return r.ManageError(instance, err)
		}
		return reconcile.Result{}, nil
	}

	// Validate Providers
	if err := groupSyncMgr.Validate(); err != nil {
		return r.ManageError(instance, err)
	}

	// Execute Each Provider Syncer
	for _, groupSyncer := range groupSyncMgr.GroupSyncers {

		reqLogger.Info("Beginning Sync", "Provider", groupSyncer.GetProviderName())

		// Provider Label
		providerLabel := fmt.Sprintf("%s_%s_%s", instance.Namespace, instance.Name, groupSyncer.GetProviderName())

		// Initialize Connection
		if err := groupSyncer.Bind(); err != nil {
			return r.ManageError(instance, err)
		}

		// Perform Sync
		groups, err := groupSyncer.Sync()

		if err != nil {
			reqLogger.Error(err, "Failed to Complete Sync", "Provider", groupSyncer.GetProviderName())
			return r.ManageError(instance, err)
		}

		updatedGroups := 0

		for _, group := range groups {

			ocpGroup := &userv1.Group{}
			err := r.GetClient().Get(context.TODO(), types.NamespacedName{Name: group.Name, Namespace: ""}, ocpGroup)

			if kapierrors.IsNotFound(err) {

				ocpGroup = &userv1.Group{}
				ocpGroup.Name = group.Name

			} else if err != nil {
				return r.ManageError(instance, err)
			} else {
				// Verify this group is not managed by another provider
				if groupProviderLabel, exists := ocpGroup.Labels[constants.SyncProvider]; !exists || (groupProviderLabel != providerLabel) {
					log.Info("Group Provider Label Did Not Match Expected Provider Label", "Group Name", ocpGroup.Name, "Expected Label", providerLabel, "Found Label", groupProviderLabel)
					continue
				}
			}

			// Copy Annotations/Labels
			ocpGroupLabels := map[string]string{}
			ocpGroupAnnotations := map[string]string{}

			if group.GetAnnotations() != nil {
				ocpGroupAnnotations = group.GetAnnotations()
			}

			if group.GetLabels() != nil {
				ocpGroupLabels = group.GetLabels()
			}
			ocpGroup.SetLabels(ocpGroupLabels)
			ocpGroup.SetAnnotations(ocpGroupAnnotations)

			// Add Label for new resource
			if ocpGroup.GetCreationTimestamp().Local().IsZero() {
				ocpGroup.Labels[constants.SyncProvider] = providerLabel
			}

			// Add Gloabl Annotations/Labels
			ocpGroup.Annotations[constants.SyncTimestamp] = ISO8601(time.Now())

			ocpGroup.Users = group.Users

			err = r.CreateOrUpdateResource(instance, "", ocpGroup)

			if err != nil {
				log.Error(err, "Failed to Create or Update OpenShift Group")
				return r.ManageError(instance, err)
			}

			updatedGroups++

		}

		reqLogger.Info("Sync Completed Successfully", "Provider", groupSyncer.GetProviderName(), "Groups Created or Updated", updatedGroups)

	}

	successResult, err := r.ManageSuccess(instance)

	if err == nil && instance.Spec.ResyncPeriodMinutes != nil {
		successResult.RequeueAfter = time.Duration((*instance.Spec.ResyncPeriodMinutes * 60) * (1000 * 1000 * 1000))
	}

	return successResult, err
}

func ISO8601(t time.Time) string {
	var tz string
	if zone, offset := t.Zone(); zone == "UTC" {
		tz = "Z"
	} else {
		tz = fmt.Sprintf("%03d00", offset/3600)
	}
	return fmt.Sprintf("%04d-%02d-%02dT%02d:%02d:%02d%s",
		t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), tz)
}
