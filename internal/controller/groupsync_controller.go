/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	userv1 "github.com/openshift/api/user/v1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/redhat-cop/group-sync-operator/pkg/constants"
	"github.com/redhat-cop/group-sync-operator/pkg/syncer"
	"github.com/redhat-cop/operator-utils/pkg/util"
	"github.com/robfig/cron"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	apimachineryvalidation "k8s.io/apimachinery/pkg/util/validation"
	kubeclock "k8s.io/utils/clock"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	redhatcopv1alpha1 "github.com/redhat-cop/group-sync-operator/api/v1alpha1"
)

var clock kubeclock.Clock = &kubeclock.RealClock{}

// GroupSyncReconciler reconciles a GroupSync object
type GroupSyncReconciler struct {
	Log logr.Logger
	util.ReconcilerBase
}

// +kubebuilder:rbac:groups=redhatcop.redhat.io,resources=groupsyncs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=redhatcop.redhat.io,resources=groupsyncs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=user.openshift.io,resources=groups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch

func (r *GroupSyncReconciler) Reconcile(context context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("groupsync", req.NamespacedName)

	// Fetch the GroupSync instance
	instance := &redhatcopv1alpha1.GroupSync{}
	err := r.GetClient().Get(context, req.NamespacedName, instance)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	// Get Group Sync Manager
	groupSyncMgr, err := syncer.GetGroupSyncMgr(instance, r.ReconcilerBase)

	if err != nil {
		return r.ManageError(context, instance, err)
	}

	// Set Defaults
	if changed := groupSyncMgr.SetDefaults(); changed {
		err := r.GetClient().Update(context, instance)
		if err != nil {
			r.Log.Error(err, "unable to update instance", "instance", instance)
			return r.ManageError(context, instance, err)
		}
		return reconcile.Result{}, nil
	}

	// Validate Providers
	if err := groupSyncMgr.Validate(); err != nil {
		return r.ManageError(context, instance, err)
	}

	// Collect the names of all live GroupSync resources once per reconcile so
	// that a Group whose recorded owner no longer exists (typically after a
	// GroupSync was renamed) can be adopted instead of skipped forever
	liveGroupSyncNames, err := r.liveGroupSyncNames(context)
	if err != nil {
		return r.ManageError(context, instance, err)
	}

	syncErrors := []error{}

	// Execute Each Provider Syncer
	for _, groupSyncer := range groupSyncMgr.GroupSyncers {

		logger.Info("Beginning Sync", "Provider", groupSyncer.GetProviderName())

		prometheusLabels := prometheus.Labels{METRICS_CR_NAMESPACE_LABEL: instance.GetNamespace(), METRICS_CR_NAME_LABEL: instance.GetName(), METRICS_PROVIDER_LABEL: groupSyncer.GetProviderName()}

		// Provider Label
		providerLabel := fmt.Sprintf("%s_%s", instance.Name, groupSyncer.GetProviderName())

		// Initialize Connection
		if err := groupSyncer.Bind(); err != nil {
			r.manageSyncError(prometheusLabels, &syncErrors, err)
			continue
		}

		// Perform Sync
		groups, err := groupSyncer.Sync()

		if err != nil {
			logger.Error(err, "Failed to Complete Sync", "Provider", groupSyncer.GetProviderName())
			r.manageSyncError(prometheusLabels, &syncErrors, err)
			continue
		}

		updatedGroups := 0
		prunedGroups := 0

		for i, group := range groups {

			// Verify valid Group Names
			if instance.Spec.ExcludeInvalidGroupNames {
				msgs := apimachineryvalidation.IsDNS1035Label(group.Name)
				if len(msgs) > 0 {
					r.Log.Info(fmt.Sprintf("Group '%s' contains invalid name: %s", group.Name, strings.Join(msgs, ",")))
					continue
				}
			}

			ocpGroup := &userv1.Group{}
			err := r.GetClient().Get(context, types.NamespacedName{Name: group.Name, Namespace: ""}, ocpGroup)

			if apierrors.IsNotFound(err) {

				ocpGroup = &userv1.Group{
					TypeMeta: metav1.TypeMeta{
						Kind:       "Group",
						APIVersion: userv1.GroupVersion.String(),
					},
				}
				ocpGroup.Name = group.Name

			} else if err != nil {
				r.manageSyncError(prometheusLabels, &syncErrors, err)
				continue
			} else {
				// Verify this group is not managed by another provider
				groupProviderLabel, exists := ocpGroup.Labels[constants.SyncProvider]
				if !shouldSyncExistingGroup(groupProviderLabel, exists, providerLabel, liveGroupSyncNames) {
					r.Log.Info("Group Provider Label Did Not Match Expected Provider Label", "Provider", groupSyncer.GetProviderName(), "Group Name", ocpGroup.Name, "Expected Label", providerLabel, "Found Label", groupProviderLabel)
					continue
				}
				if groupProviderLabel != providerLabel {
					r.Log.Info("Adopting Group Whose GroupSync No Longer Exists", "Provider", groupSyncer.GetProviderName(), "Group Name", ocpGroup.Name, "New Label", providerLabel, "Previous Label", groupProviderLabel)
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
			ocpGroup.SetLabels(mergeMap(ocpGroup.GetLabels(), ocpGroupLabels))
			ocpGroup.SetAnnotations(mergeMap(ocpGroup.GetAnnotations(), ocpGroupAnnotations))

			// Add Label for new resource
			ocpGroup.Labels[constants.SyncProvider] = providerLabel

			// Add Gloabl Annotations/Labels
			now := time.Now().UTC().Format(time.RFC3339)
			ocpGroup.Annotations[constants.SyncTimestamp] = now

			ocpGroup.Users = group.Users

			err = r.CreateOrUpdateResource(context, nil, "", ocpGroup)

			group.UID = ocpGroup.UID
			groups[i] = group

			if err != nil {
				r.Log.Error(err, "Failed to Create or Update OpenShift Group", "Provider", groupSyncer.GetProviderName())
				r.manageSyncError(prometheusLabels, &syncErrors, err)
				continue
			}

			updatedGroups++
		}

		if groupSyncer.GetPrune() {
			logger.Info("Start Pruning Groups", "Provider", groupSyncer.GetProviderName())
			prunedGroups, err = r.pruneGroups(context, instance, groups, groupSyncer.GetProviderName(), providerLabel, logger)
			if err != nil {
				r.Log.Error(err, "Failed to Prune Group", "Provider", groupSyncer.GetProviderName())
				r.manageSyncError(prometheusLabels, &syncErrors, err)
			}
			logger.Info("Pruning Completed", "Provider", groupSyncer.GetProviderName())
		}

		logger.Info("Sync Completed Successfully", "Provider", groupSyncer.GetProviderName(), "Groups Created or Updated", updatedGroups, "Groups Pruned", prunedGroups)

		// Add Metrics
		successfulGroupSyncs.With(prometheusLabels).Inc()
		groupsSynchronized.With(prometheusLabels).Set(float64(updatedGroups))
		groupSyncError.With(prometheusLabels).Set(0)
		if groupSyncer.GetPrune() {
			groupsPruned.With(prometheusLabels).Set(float64(prunedGroups))
		}
	}

	// Throw error if error occurred during sync
	if len(syncErrors) > 0 {
		return r.ManageError(context, instance, utilerrors.NewAggregate(syncErrors))
	}

	instance.Status.LastSyncSuccessTime = &metav1.Time{Time: clock.Now()}

	successResult, err := r.ManageSuccess(context, instance)

	if err == nil && instance.Spec.Schedule != "" {
		sched, _ := cron.ParseStandard(instance.Spec.Schedule)

		currentTime := time.Now().UTC()
		nextScheduledTime := sched.Next(currentTime)
		nextScheduledSynchronization.With(prometheus.Labels{METRICS_CR_NAMESPACE_LABEL: instance.GetNamespace(), METRICS_CR_NAME_LABEL: instance.GetName()}).Set(float64(nextScheduledTime.UTC().Unix()))
		successResult.RequeueAfter = nextScheduledTime.Sub(currentTime)
	}

	return successResult, err
}

func (r *GroupSyncReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&redhatcopv1alpha1.GroupSync{}).
		WithEventFilter(util.ResourceGenerationOrFinalizerChangedPredicate{}).
		Complete(r)
}

func (r *GroupSyncReconciler) manageSyncError(prometheusLabels prometheus.Labels, syncErrors *[]error, err error) {

	unsuccessfulGroupSyncs.With(prometheusLabels).Inc()
	groupSyncError.With(prometheusLabels).Set(1)

	*syncErrors = append(*syncErrors, err)

}

// liveGroupSyncNames returns the names of all GroupSync resources currently
// present in the cluster. Groups record only the GroupSync name in their
// sync-provider label (not the namespace), so names are collected across all
// namespaces: as long as any live GroupSync claims a name, Groups labeled with
// that name are treated as owned and are never adopted.
func (r *GroupSyncReconciler) liveGroupSyncNames(context context.Context) (map[string]bool, error) {

	groupSyncList := &redhatcopv1alpha1.GroupSyncList{}
	if err := r.GetClient().List(context, groupSyncList); err != nil {
		return nil, err
	}

	names := make(map[string]bool, len(groupSyncList.Items))
	for _, groupSync := range groupSyncList.Items {
		names[groupSync.Name] = true
	}

	return names, nil
}

// shouldSyncExistingGroup decides whether the current reconcile may write to an
// existing Group, based on the Group's sync-provider label. It answers true
// when the label matches the provider label of the current reconcile, or when
// the label names a GroupSync that no longer exists — the Group is orphaned
// (typically by a GroupSync rename) and is adopted. A Group without the label
// was never synced by this operator and is left alone, as is a Group whose
// label names a live GroupSync (genuine contention).
func shouldSyncExistingGroup(foundLabel string, labelExists bool, providerLabel string, liveGroupSyncNames map[string]bool) bool {

	if !labelExists {
		return false
	}

	if foundLabel == providerLabel {
		return true
	}

	// The label format is "<groupsync-name>_<provider-name>". Kubernetes object
	// names cannot contain an underscore, so the first underscore unambiguously
	// terminates the GroupSync name even when the provider name contains one.
	groupSyncName, _, found := strings.Cut(foundLabel, "_")
	if !found || groupSyncName == "" {
		return false
	}

	return !liveGroupSyncNames[groupSyncName]
}

func (r *GroupSyncReconciler) pruneGroups(context context.Context, instance *redhatcopv1alpha1.GroupSync, syncedGroups []userv1.Group, providerName, providerLabel string, logger logr.Logger) (int, error) {
	prunedGroups := 0

	ocpGroups := &userv1.GroupList{}
	opts := []client.ListOption{
		client.InNamespace(""),
		client.MatchingLabels{constants.SyncProvider: providerLabel},
	}
	err := r.GetClient().List(context, ocpGroups, opts...)
	if err != nil {
		return prunedGroups, err
	}

	for _, group := range ocpGroups.Items {

		// Remove group if not found in the list of synchronized groups
		groupFound := isGroupFound(group, syncedGroups)

		if !groupFound {
			logger.Info("Pruning Group", "Provider", providerName, "Group", group.Name)
			err = r.GetClient().Delete(context, &group)
			prunedGroups++
			if err != nil {
				return prunedGroups, err
			}
		}
	}
	return prunedGroups, nil
}

func isGroupFound(canidateGroup userv1.Group, baseGroups []userv1.Group) bool {

	for _, baseGroup := range baseGroups {
		if baseGroup.UID == canidateGroup.UID {
			return true
		}
	}

	return false
}

func mergeMap(m1, m2 map[string]string) map[string]string {

	if m1 != nil {
		for mKey, mValue := range m2 {
			m1[mKey] = mValue
		}

		return m1

	} else {
		return m2
	}

}
