package syncer

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/okta/okta-sdk-golang/v2/okta/query"

	"github.com/okta/okta-sdk-golang/v2/okta"
	userv1 "github.com/openshift/api/user/v1"
	"github.com/redhat-cop/group-sync-operator/api/v1alpha1"
	"github.com/redhat-cop/group-sync-operator/pkg/constants"
	"github.com/redhat-cop/operator-utils/pkg/util"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/validation"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	oktaLogger = logf.Log.WithName("syncer_okta")
)

const (
	// API token given by Okta application
	secretOktaTokenKey = "okta-api-token"
	activeStatus       = "ACTIVE"
)

type OktaSyncer struct {
	cachedGroups       map[string]*okta.Group
	cachedGroupMembers map[string][]*okta.User
	credentialsSecret  *corev1.Secret
	goOkta             *okta.Client
	GroupSync          *v1alpha1.GroupSync
	Name               string
	Provider           *v1alpha1.OktaProvider
	ReconcilerBase     util.ReconcilerBase
}

func (o *OktaSyncer) Init() bool {
	o.cachedGroupMembers = make(map[string][]*okta.User)
	o.cachedGroups = make(map[string]*okta.Group)

	if o.Provider.GroupLimit == 0 {
		o.Provider.GroupLimit = 1000
	}

	if o.Provider.ProfileKey == "" {
		o.Provider.ProfileKey = "login"
		return true
	}

	return false
}

func (o *OktaSyncer) Validate() error {
	const validations = 2
	validationErrors := make([]error, validations)
	credentialsSecret, err := o.getSecrets()

	if err != nil {
		validationErrors = append(validationErrors, err)
	} else {
		if _, found := credentialsSecret.Data[secretOktaTokenKey]; !found {
			validationErrors = append(validationErrors, fmt.Errorf("could not find api token '%s' in namespace '%s'", o.Provider.CredentialsSecret.Name, o.Provider.CredentialsSecret.Namespace))
		}

		o.credentialsSecret = credentialsSecret
	}

	if _, err := url.ParseRequestURI(o.Provider.URL); err != nil {
		validationErrors = append(validationErrors, err)
	}

	return utilerrors.NewAggregate(validationErrors)
}

func (o OktaSyncer) getSecrets() (*corev1.Secret, error) {
	credentialsSecret := &corev1.Secret{}
	nameSpacedName := types.NamespacedName{
		Name:      o.Provider.CredentialsSecret.Name,
		Namespace: o.Provider.CredentialsSecret.Namespace,
	}
	err := o.ReconcilerBase.GetClient().Get(context.TODO(), nameSpacedName, credentialsSecret)
	return credentialsSecret, err
}

func (o *OktaSyncer) Bind() error {
	var err error

	_, o.goOkta, err = okta.NewClient(context.TODO(),
		okta.WithOrgUrl(o.Provider.URL),
		okta.WithToken(string(o.credentialsSecret.Data[secretOktaTokenKey])))
	if err != nil {
		oktaLogger.Error(err, "establishing new okta client")
		return err
	}

	return nil
}

func (o *OktaSyncer) Sync() ([]userv1.Group, error) {

	groups, err := o.getGroups()
	if err != nil {
		oktaLogger.Error(err, "failed to get Groups", "Provider", o.Name)
		return nil, err
	}

	for _, group := range groups {
		if _, groupFound := o.cachedGroups[group.Id]; !groupFound {
			if err := o.processGroupsAndMembers(group); err != nil {
				oktaLogger.Error(err, "processing groups and members")
			}
		}
	}

	providerUrl, err := url.Parse(o.Provider.URL)
	if err != nil {
		return nil, err
	}

	var ocpGroups []userv1.Group
	for _, cachedGroup := range o.cachedGroups {
		validatedGroupAttr := map[string]string{}
		groupAttributes := o.mapAttributes(cachedGroup)
		for key, value := range groupAttributes {
			if errs := validation.IsQualifiedName(key); len(errs) != 0 {
				oktaLogger.Info("unable to add annotation to", "group", cachedGroup.Profile.Name, "key", key, "value", value)
			} else {
				validatedGroupAttr[key] = value
			}
		}

		ocpGroup := userv1.Group{
			TypeMeta: v1.TypeMeta{
				Kind:       "Group",
				APIVersion: userv1.GroupVersion.String(),
			},
			ObjectMeta: v1.ObjectMeta{
				Name:        cachedGroup.Profile.Name,
				Annotations: validatedGroupAttr,
				Labels:      map[string]string{},
			},
			Users: []string{},
		}

		ocpGroup.GetAnnotations()[constants.SyncSourceHost] = providerUrl.Host
		ocpGroup.GetAnnotations()[constants.SyncSourceUID] = cachedGroup.Id

		users := o.cachedGroupMembers[cachedGroup.Id]
		for _, user := range users {
			profile := *user.Profile
			if user.Status == activeStatus {
				if userName, ok := profile[o.Provider.ProfileKey].(string); !ok {
					oktaLogger.Info("attribute unavailable on okta user profile " + o.Provider.ProfileKey)
				} else if o.Provider.ExtractLoginUsername {
					userName = strings.Split(userName, "@")[0]
					ocpGroup.Users = append(ocpGroup.Users, userName)
				} else {
					ocpGroup.Users = append(ocpGroup.Users, userName)
				}
			}
		}
		ocpGroups = append(ocpGroups, ocpGroup)
	}

	return ocpGroups, nil
}

func (o OktaSyncer) getGroups() ([]*okta.Group, error) {
	var (
		groups []*okta.Group
	)

	appGroups, resp, err := o.goOkta.Application.ListApplicationGroupAssignments(context.TODO(), o.Provider.AppId, query.NewQueryParams(query.WithLimit(int64(o.Provider.GroupLimit))))

	if err != nil {
		oktaLogger.Error(err, "getting groups for specified application")
		return nil, err
	}

	for resp.HasNextPage() {
		var nextAppGroups []*okta.ApplicationGroupAssignment
		resp, err = resp.Next(context.TODO(), &nextAppGroups)

		if err != nil {
			oktaLogger.Error(err, "getting groups for specified application")
			return nil, err
		}

		appGroups = append(appGroups, nextAppGroups...)
	}

	groups, err = o.fetchGroupsAsync(appGroups)
	return groups, err
}

func (o OktaSyncer) fetchGroupsAsync(appGroups []*okta.ApplicationGroupAssignment) ([]*okta.Group, error) {

	wg := &sync.WaitGroup{}
	groupCh := make(chan *okta.Group, len(appGroups))
	wg.Add(len(appGroups))
	for _, appGroup := range appGroups {
		go getGroup(appGroup, groupCh, o.goOkta.Group, wg)
	}

	wg.Wait()
	close(groupCh)

	var groups []*okta.Group
	for v := range groupCh {
		groups = append(groups, v)
	}

	if len(groups) != len(appGroups) {
		return groups, errors.New("failed to retrieve all groups")
	}

	return groups, nil
}

func getGroup(app *okta.ApplicationGroupAssignment, groupChan chan *okta.Group, resource *okta.GroupResource, wg *sync.WaitGroup) {
	defer wg.Done()
	group, _, err := resource.GetGroup(context.TODO(), app.Id)
	if err != nil {
		oktaLogger.Error(err, "fetching group id "+app.Id)
	} else {
		groupChan <- group
	}
}

func (o OktaSyncer) mapAttributes(group *okta.Group) map[string]string {
	attr := make(map[string]string)

	attr["created"] = group.Created.String()
	attr["description"] = group.Profile.Description
	attr["id"] = group.Id
	attr["lastUpdated"] = group.LastUpdated.String()
	attr["lastMembershipUpdated"] = group.LastMembershipUpdated.String()
	attr["name"] = group.Profile.Name
	attr["objectClass"] = strings.Join(group.ObjectClass, ",")
	attr["type"] = group.Type

	return attr
}

func (o *OktaSyncer) processGroupsAndMembers(group *okta.Group) error {

	if !isGroupAllowed(group.Profile.Name, o.Provider.Groups) {
		return nil
	}

	o.cachedGroups[group.Id] = group
	users, _, err := o.goOkta.Group.ListGroupUsers(context.TODO(), group.Id, nil)
	if err != nil {
		oktaLogger.Error(err, "failed to get users", "Provider", o.Name)
		return err
	}

	o.cachedGroupMembers[group.Id] = users
	return nil
}

func (o *OktaSyncer) GetProviderName() string {
	return o.Name
}

func (o *OktaSyncer) GetPrune() bool {
	return o.Provider.Prune
}
