package syncer

import (
	"context"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/adal"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	userv1 "github.com/openshift/api/user/v1"
	redhatcopv1alpha1 "github.com/redhat-cop/group-sync-operator/pkg/apis/redhatcop/v1alpha1"
	"github.com/redhat-cop/group-sync-operator/pkg/controller/constants"
	"github.com/redhat-cop/operator-utils/pkg/util"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	azureLogger = logf.Log.WithName("syncer_azure")
)

const (
	SubscriptionID          = "AZURE_SUBSCRIPTION_ID"
	TenantID                = "AZURE_TENANT_ID"
	AuxiliaryTenantIDs      = "AZURE_AUXILIARY_TENANT_IDS"
	ClientID                = "AZURE_CLIENT_ID"
	ClientSecret            = "AZURE_CLIENT_SECRET"
	CertificatePath         = "AZURE_CERTIFICATE_PATH"
	CertificatePassword     = "AZURE_CERTIFICATE_PASSWORD"
	Username                = "AZURE_USERNAME"
	Password                = "AZURE_PASSWORD"
	EnvironmentName         = "AZURE_ENVIRONMENT"
	Resource                = "AZURE_AD_RESOURCE"
	ActiveDirectoryEndpoint = "ActiveDirectoryEndpoint"
	ResourceManagerEndpoint = "ResourceManagerEndpoint"
	GraphResourceID         = "GraphResourceID"
	SQLManagementEndpoint   = "SQLManagementEndpoint"
	GalleryEndpoint         = "GalleryEndpoint"
	ManagementEndpoint      = "ManagementEndpoint"
)

type AzureSyncer struct {
	Name              string
	GroupSync         *redhatcopv1alpha1.GroupSync
	Provider          *redhatcopv1alpha1.AzureProvider
	Client            graphrbac.GroupsClient
	ReconcilerBase    util.ReconcilerBase
	CredentialsSecret *corev1.Secret
	CachedGroups      map[string]*graphrbac.ADGroup
	CachedGroupUsers  map[string][]*graphrbac.User
}

func (a *AzureSyncer) Init() bool {

	a.CachedGroups = make(map[string]*graphrbac.ADGroup)
	a.CachedGroupUsers = make(map[string][]*graphrbac.User)

	return false
}

func (a *AzureSyncer) Validate() error {

	validationErrors := []error{}

	credentialsSecret := &corev1.Secret{}
	err := a.ReconcilerBase.GetClient().Get(context.TODO(), types.NamespacedName{Name: a.Provider.CredentialsSecretName, Namespace: a.GroupSync.Namespace}, credentialsSecret)

	if err != nil {
		validationErrors = append(validationErrors, err)
	} else {

		// Check that provided secret contains required keys
		_, subscriptionIDSecretFound := credentialsSecret.Data[SubscriptionID]
		_, tenantIDSecretFound := credentialsSecret.Data[TenantID]
		_, clientIDSecretFound := credentialsSecret.Data[ClientID]
		_, clientSecretSecretFound := credentialsSecret.Data[ClientSecret]

		if !subscriptionIDSecretFound || !tenantIDSecretFound || !clientIDSecretFound || !clientSecretSecretFound {
			validationErrors = append(validationErrors, fmt.Errorf("Could not find 'AZURE_SUBSCRIPTION_ID' or `AZURE_TENANT_ID` or `AZURE_CLIENT_ID` or `AZURE_CLIENT_SECRET` key in secret '%s' in namespace '%s", a.Provider.CredentialsSecretName, a.GroupSync.Namespace))
		}

		a.CredentialsSecret = credentialsSecret

	}

	return utilerrors.NewAggregate(validationErrors)

}

func (a *AzureSyncer) Bind() error {

	envSettings := auth.EnvironmentSettings{
		Values: map[string]string{},
	}

	// Map all settings
	a.setEnvironmentSettingsValue(&envSettings, a.CredentialsSecret.Data, SubscriptionID)
	a.setEnvironmentSettingsValue(&envSettings, a.CredentialsSecret.Data, TenantID)
	a.setEnvironmentSettingsValue(&envSettings, a.CredentialsSecret.Data, AuxiliaryTenantIDs)
	a.setEnvironmentSettingsValue(&envSettings, a.CredentialsSecret.Data, ClientID)
	a.setEnvironmentSettingsValue(&envSettings, a.CredentialsSecret.Data, ClientSecret)
	a.setEnvironmentSettingsValue(&envSettings, a.CredentialsSecret.Data, CertificatePath)
	a.setEnvironmentSettingsValue(&envSettings, a.CredentialsSecret.Data, CertificatePassword)
	a.setEnvironmentSettingsValue(&envSettings, a.CredentialsSecret.Data, Username)
	a.setEnvironmentSettingsValue(&envSettings, a.CredentialsSecret.Data, Password)
	a.setEnvironmentSettingsValue(&envSettings, a.CredentialsSecret.Data, CertificatePassword)
	a.setEnvironmentSettingsValue(&envSettings, a.CredentialsSecret.Data, EnvironmentName)
	a.setEnvironmentSettingsValue(&envSettings, a.CredentialsSecret.Data, Resource)
	if v := envSettings.Values[EnvironmentName]; v == "" {
		envSettings.Environment = azure.PublicCloud
	} else {
		envSettings.Environment, _ = azure.EnvironmentFromName(v)

	}
	if envSettings.Values[Resource] == "" {
		envSettings.Values[Resource] = envSettings.Environment.ResourceManagerEndpoint
	}

	// authorizer, err := envSettings.GetAuthorizer()
	oauthConfig, err := adal.NewOAuthConfig(
		envSettings.Environment.ActiveDirectoryEndpoint, envSettings.Values[TenantID])
	if err != nil {
		return err
	}

	token, err := adal.NewServicePrincipalToken(
		*oauthConfig, envSettings.Values[ClientID], envSettings.Values[ClientSecret], envSettings.Environment.GraphEndpoint)
	if err != nil {
		return err
	}

	authorizer := autorest.NewBearerAuthorizer(token)
	groupsClient := graphrbac.NewGroupsClient(envSettings.Values[TenantID])
	groupsClient.Authorizer = authorizer
	groupsClient.UserAgent = "group-sync-operator"

	a.Client = groupsClient

	return nil
}

func (a *AzureSyncer) Sync() ([]userv1.Group, error) {

	for group, err := a.Client.ListComplete(context.Background(), ""); group.NotDone(); err = group.Next() {

		if err != nil {
			return nil, err
		}

		if _, groupFound := a.CachedGroups[*group.Value().ObjectID]; !groupFound {

			groupValue, _ := group.Value().AsADGroup()

			a.processGroupsAndMembers(a.Client, groupValue, nil)
		}

	}

	ocpGroups := []userv1.Group{}

	for _, cachedGroup := range a.CachedGroups {

		ocpGroup := userv1.Group{
			TypeMeta: v1.TypeMeta{
				Kind:       "Group",
				APIVersion: userv1.SchemeGroupVersion.String(),
			},
			ObjectMeta: v1.ObjectMeta{
				Name:        *cachedGroup.DisplayName,
				Annotations: map[string]string{},
				Labels:      map[string]string{},
			},
			Users: []string{},
		}

		// Set Host Specific Details
		ocpGroup.GetAnnotations()[constants.SyncSourceHost] = a.Client.BaseURI
		ocpGroup.GetAnnotations()[constants.SyncSourceUID] = *cachedGroup.ObjectID

		for _, user := range a.CachedGroupUsers[*cachedGroup.ObjectID] {
			ocpGroup.Users = append(ocpGroup.Users, *user.DisplayName)
		}

		ocpGroups = append(ocpGroups, ocpGroup)

	}

	return ocpGroups, nil

}

func (a *AzureSyncer) GetProviderName() string {
	return a.Name
}

func (a *AzureSyncer) processGroupsAndMembers(groupsClient graphrbac.GroupsClient, group, parentGroup *graphrbac.ADGroup) error {

	if parentGroup == nil && !isGroupAllowed(*group.DisplayName, []string{}) {
		return nil
	}

	// Check to see if we have seen this group already
	if _, groupFound := a.CachedGroups[*group.ObjectID]; groupFound {
		if parentGroup != nil {
			usersToAdd, _ := a.diff(a.CachedGroupUsers[*group.ObjectID], a.CachedGroupUsers[*parentGroup.ObjectID])
			a.CachedGroupUsers[*parentGroup.ObjectID] = append(a.CachedGroupUsers[*parentGroup.ObjectID], usersToAdd...)
		}
		return nil
	}

	a.CachedGroups[*group.ObjectID] = group

	groupUsers := []*graphrbac.User{}
	subGroups := []*graphrbac.ADGroup{}

	// Query Group Membership (Separate out groups and users)
	for member, err := groupsClient.GetGroupMembersComplete(context.Background(), *group.ObjectID); member.NotDone(); err = member.Next() {
		if err != nil {
			return err
		}

		if user, ok := member.Value().AsUser(); ok {
			groupUsers = append(groupUsers, user)
		}

		if subgroup, ok := member.Value().AsADGroup(); ok {
			subGroups = append(subGroups, subgroup)
		}
	}

	usersToAdd, _ := a.diff(groupUsers, a.CachedGroupUsers[*group.ObjectID])
	a.CachedGroupUsers[*group.ObjectID] = append(a.CachedGroupUsers[*group.ObjectID], usersToAdd...)

	if parentGroup != nil {
		usersToAdd, _ := a.diff(groupUsers, a.CachedGroupUsers[*parentGroup.ObjectID])
		a.CachedGroupUsers[*parentGroup.ObjectID] = append(a.CachedGroupUsers[*parentGroup.ObjectID], usersToAdd...)
	}

	for _, subgroup := range subGroups {
		a.processGroupsAndMembers(groupsClient, subgroup, group)
	}

	return nil

}

func (a *AzureSyncer) diff(lhsSlice, rhsSlice []*graphrbac.User) (lhsOnly []*graphrbac.User, rhsOnly []*graphrbac.User) {
	return a.singleDiff(lhsSlice, rhsSlice), a.singleDiff(rhsSlice, lhsSlice)
}

func (a *AzureSyncer) singleDiff(lhsSlice, rhsSlice []*graphrbac.User) (lhsOnly []*graphrbac.User) {
	for _, lhs := range lhsSlice {
		found := false
		for _, rhs := range rhsSlice {
			if *lhs.ObjectID == *rhs.ObjectID {
				found = true
				break
			}
		}

		if !found {
			lhsOnly = append(lhsOnly, lhs)
		}
	}

	return lhsOnly
}

func (a *AzureSyncer) setEnvironmentSettingsValue(environmentSettings *auth.EnvironmentSettings, data map[string][]byte, key string) {
	if _, found := data[key]; found {
		environmentSettings.Values[key] = string(data[key])
	}

}
