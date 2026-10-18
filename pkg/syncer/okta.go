package syncer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"

	"github.com/okta/okta-sdk-golang/v7/okta"
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
	cachedGroupMembers map[string][]okta.User
	credentialsSecret  *corev1.Secret
	goOkta             *okta.APIClient
	GroupSync          *v1alpha1.GroupSync
	Name               string
	Provider           *v1alpha1.OktaProvider
	ReconcilerBase     util.ReconcilerBase
}

func (o *OktaSyncer) Init() bool {
	o.cachedGroupMembers = make(map[string][]okta.User)
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

	if o.Provider.CredentialsSecret != nil {
		credentialsSecret, err := o.getSecrets()

		if err != nil {
			validationErrors = append(validationErrors, err)
		} else {
			if _, found := credentialsSecret.Data[secretOktaTokenKey]; !found {
				validationErrors = append(validationErrors, fmt.Errorf("could not find api token '%s' in namespace '%s'", o.Provider.CredentialsSecret.Name, o.Provider.CredentialsSecret.Namespace))
			}

			o.credentialsSecret = credentialsSecret
		}
	} else {
		validationErrors = append(validationErrors, fmt.Errorf("credentialsSecret must be provided for Okta provider"))
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
	config, err := okta.NewConfiguration(
		okta.WithOrgUrl(o.Provider.URL),
		okta.WithToken(string(o.credentialsSecret.Data[secretOktaTokenKey])))
	if err != nil {
		oktaLogger.Error(err, "establishing new okta client")
		return err
	}

	o.goOkta = okta.NewAPIClient(config)

	return nil
}

func (o *OktaSyncer) Sync() ([]userv1.Group, error) {

	groups, err := o.getGroups()
	if err != nil {
		oktaLogger.Error(err, "failed to get Groups", "Provider", o.Name)
		return nil, err
	}

	for _, group := range groups {
		if _, groupFound := o.cachedGroups[group.GetId()]; !groupFound {
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
		groupName, _ := groupProfileFields(cachedGroup)
		validatedGroupAttr := map[string]string{}
		groupAttributes := o.mapAttributes(cachedGroup)
		for key, value := range groupAttributes {
			if errs := validation.IsQualifiedName(key); len(errs) != 0 {
				oktaLogger.Info("unable to add annotation to", "group", groupName, "key", key, "value", value)
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
				Name:        groupName,
				Annotations: validatedGroupAttr,
				Labels:      map[string]string{},
			},
			Users: []string{},
		}

		ocpGroup.GetAnnotations()[constants.SyncSourceHost] = providerUrl.Host
		ocpGroup.GetAnnotations()[constants.SyncSourceUID] = cachedGroup.GetId()

		users := o.cachedGroupMembers[cachedGroup.GetId()]
		for _, user := range users {
			if user.GetStatus() == activeStatus {
				if userName, ok := getProfileAttribute(user.GetProfile(), o.Provider.ProfileKey); !ok {
					oktaLogger.Info("attribute unavailable on okta user profile " + o.Provider.ProfileKey)
				} else if o.Provider.ExtractLoginUsername {
					userName, _, _ = strings.Cut(userName, "@")
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

	appGroups, resp, err := o.goOkta.ApplicationGroupsAPI.ListApplicationGroupAssignments(context.TODO(), o.Provider.AppId).Limit(int32(o.Provider.GroupLimit)).Execute()

	if err != nil {
		oktaLogger.Error(err, "getting groups for specified application")
		return nil, err
	}

	for resp.HasNextPage() {
		var nextAppGroups []okta.ApplicationGroupAssignment
		resp, err = resp.Next(&nextAppGroups)

		if err != nil {
			oktaLogger.Error(err, "getting groups for specified application")
			return nil, err
		}

		appGroups = append(appGroups, nextAppGroups...)
	}

	groups, err = o.fetchGroupsAsync(appGroups)
	return groups, err
}

func (o OktaSyncer) fetchGroupsAsync(appGroups []okta.ApplicationGroupAssignment) ([]*okta.Group, error) {

	wg := &sync.WaitGroup{}
	groupCh := make(chan *okta.Group, len(appGroups))
	wg.Add(len(appGroups))
	for _, appGroup := range appGroups {
		go getGroup(appGroup, groupCh, o.goOkta.GroupAPI, wg)
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

func getGroup(app okta.ApplicationGroupAssignment, groupChan chan *okta.Group, groupAPI okta.GroupAPI, wg *sync.WaitGroup) {
	defer wg.Done()
	group, _, err := groupAPI.GetGroupExecute(groupAPI.GetGroup(context.TODO(), app.GetId()))
	if err != nil {
		oktaLogger.Error(err, "fetching group id "+app.GetId())
	} else {
		groupChan <- group
	}
}

// groupProfileFields returns the name and description of an Okta group. Group
// profiles are polymorphic: standard groups use OktaUserGroupProfile while
// groups imported from Active Directory use OktaActiveDirectoryGroupProfile.
func groupProfileFields(group *okta.Group) (string, string) {
	if group.Profile == nil {
		return "", ""
	}

	if profile := group.Profile.OktaUserGroupProfile; profile != nil {
		return profile.GetName(), profile.GetDescription()
	}

	if profile := group.Profile.OktaActiveDirectoryGroupProfile; profile != nil {
		return profile.GetName(), profile.GetDescription()
	}

	return "", ""
}

// getProfileAttribute looks up an attribute of an Okta user profile by key.
// Custom attributes are exposed through AdditionalProperties, while standard
// attributes are typed fields, so the profile is serialized to a map to allow
// generic lookups.
func getProfileAttribute(profile okta.UserProfile, key string) (string, bool) {
	profileBytes, err := json.Marshal(profile)
	if err != nil {
		return "", false
	}

	profileMap := map[string]interface{}{}
	if err := json.Unmarshal(profileBytes, &profileMap); err != nil {
		return "", false
	}

	value, found := profileMap[key]
	if !found {
		return "", false
	}

	userName, ok := value.(string)
	return userName, ok
}

func (o OktaSyncer) mapAttributes(group *okta.Group) map[string]string {
	attr := make(map[string]string)

	name, description := groupProfileFields(group)

	attr["created"] = group.GetCreated().String()
	attr["description"] = description
	attr["id"] = group.GetId()
	attr["lastUpdated"] = group.GetLastUpdated().String()
	attr["lastMembershipUpdated"] = group.GetLastMembershipUpdated().String()
	attr["name"] = name
	attr["objectClass"] = strings.Join(group.ObjectClass, ",")
	attr["type"] = group.GetType()

	return attr
}

func (o *OktaSyncer) processGroupsAndMembers(group *okta.Group) error {

	groupName, _ := groupProfileFields(group)
	if !isGroupAllowed(groupName, o.Provider.Groups) {
		return nil
	}

	o.cachedGroups[group.GetId()] = group
	users, _, err := o.goOkta.GroupAPI.ListGroupUsers(context.TODO(), group.GetId()).Execute()
	if err != nil {
		oktaLogger.Error(err, "failed to get users", "Provider", o.Name)
		return err
	}

	o.cachedGroupMembers[group.GetId()] = users
	return nil
}

func (o *OktaSyncer) GetProviderName() string {
	return o.Name
}

func (o *OktaSyncer) GetPrune() bool {
	return o.Provider.Prune
}
