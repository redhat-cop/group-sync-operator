package syncer

import (
	"context"
	"fmt"
	"net/url"

	userv1 "github.com/openshift/api/user/v1"
	redhatcopv1alpha1 "github.com/redhat-cop/group-sync-operator/api/v1alpha1"
	"github.com/redhat-cop/group-sync-operator/pkg/constants"
	"github.com/redhat-cop/operator-utils/pkg/util"
	msgraph "github.com/yaegashi/msgraph.go/beta"
	"github.com/yaegashi/msgraph.go/msauth"
	"golang.org/x/oauth2"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	azureLogger = logf.Log.WithName("syncer_azure")
	Scopes      = []string{msauth.DefaultMSGraphScope}
)

const (
	TenantID               = "AZURE_TENANT_ID"
	ClientID               = "AZURE_CLIENT_ID"
	ClientSecret           = "AZURE_CLIENT_SECRET"
	GraphGroupType         = "#microsoft.graph.group"
	GraphUserType          = "#microsoft.graph.user"
	GraphOdataType         = "@odata.type"
	GraphID                = "id"
	GraphDisplayName       = "displayName"
	GraphUserNameAttribute = "userPrincipalName"
)

type AzureSyncer struct {
	Name              string
	GroupSync         *redhatcopv1alpha1.GroupSync
	Provider          *redhatcopv1alpha1.AzureProvider
	Client            *msgraph.GraphServiceRequestBuilder
	ReconcilerBase    util.ReconcilerBase
	CredentialsSecret *corev1.Secret
	CachedGroups      map[string]*msgraph.Group
	CachedGroupUsers  map[string][]*msgraph.User
	Context           context.Context
}

func (a *AzureSyncer) Init() bool {

	a.CachedGroups = make(map[string]*msgraph.Group)
	a.CachedGroupUsers = make(map[string][]*msgraph.User)
	a.Context = context.Background()

	return false
}

func (a *AzureSyncer) Validate() error {

	validationErrors := []error{}

	credentialsSecret := &corev1.Secret{}
	err := a.ReconcilerBase.GetClient().Get(a.Context, types.NamespacedName{Name: a.Provider.CredentialsSecret.Name, Namespace: a.Provider.CredentialsSecret.Namespace}, credentialsSecret)

	if err != nil {
		validationErrors = append(validationErrors, err)
	} else {

		// Check that provided secret contains required keys
		_, tenantIDSecretFound := credentialsSecret.Data[TenantID]
		_, clientIDSecretFound := credentialsSecret.Data[ClientID]
		_, clientSecretSecretFound := credentialsSecret.Data[ClientSecret]

		if !tenantIDSecretFound || !clientIDSecretFound || !clientSecretSecretFound {
			validationErrors = append(validationErrors, fmt.Errorf("Could not find `AZURE_TENANT_ID` or `AZURE_CLIENT_ID` or `AZURE_CLIENT_SECRET` key in secret '%s' in namespace '%s", a.Provider.CredentialsSecret.Name, a.Provider.CredentialsSecret.Namespace))
		}

		a.CredentialsSecret = credentialsSecret

	}

	return utilerrors.NewAggregate(validationErrors)

}

func (a *AzureSyncer) Bind() error {

	m := msauth.NewManager()

	ts, err := m.ClientCredentialsGrant(a.Context, string(a.CredentialsSecret.Data[TenantID]), string(a.CredentialsSecret.Data[ClientID]), string(a.CredentialsSecret.Data[ClientSecret]), Scopes)
	if err != nil {
		return err
	}

	httpClient := oauth2.NewClient(a.Context, ts)
	graphClient := msgraph.NewClient(httpClient)

	a.Client = graphClient

	return nil

}

func (a *AzureSyncer) Sync() ([]userv1.Group, error) {

	ocpGroups := []userv1.Group{}
	aadGroups := []msgraph.Group{}

	if a.Provider.BaseGroups != nil && len(a.Provider.BaseGroups) > 0 {

		for _, baseGroup := range a.Provider.BaseGroups {

			baseGroupRequest := a.Client.Groups().Request()
			baseGroupRequest.Filter(fmt.Sprintf("displayName eq '%s'", baseGroup))
			baseGroupResult, err := baseGroupRequest.Get(a.Context)

			if err != nil {
				azureLogger.Error(err, "Failed to get base group", "Provider", a.Name, "Base Group", baseGroup)
				return nil, err
			}

			// Check that only 1 group was found
			if len(baseGroupResult) != 1 {
				azureLogger.Info("Failed to find a single base group to search from", "Provider", a.Name, "Base Group", baseGroup)
				continue
			}

			// Add Base Group
			aadGroups = append(aadGroups, baseGroupResult[0])

			baseGroupMembersRequest := a.Client.Groups().ID(*baseGroupResult[0].ID).Members().Request()

			if a.Provider.Filter != "" {
				baseGroupMembersRequest.Filter(a.Provider.Filter)
			}

			baseGroupMembersResult, err := baseGroupMembersRequest.Get(a.Context)

			if err != nil {
				azureLogger.Error(err, "Failed to get base group members", "Provider", a.Name, "Base Group", baseGroup)
				return nil, err
			}

			for _, baseGroupMember := range baseGroupMembersResult {

				baseGroupMemberODataType, _ := baseGroupMember.GetAdditionalData(GraphOdataType)

				// Add base groups
				if GraphGroupType == baseGroupMemberODataType {

					baseGroupDisplayNameRaw, _ := baseGroupMember.GetAdditionalData(GraphDisplayName)
					baseGroupDisplayName := baseGroupDisplayNameRaw.(string)

					aadGroups = append(aadGroups, msgraph.Group{
						DirectoryObject: baseGroupMember,
						DisplayName:     &baseGroupDisplayName,
					})
				}
			}

		}

	} else {

		groupRequest := a.Client.Groups().Request()

		if a.Provider.Filter != "" {
			groupRequest.Filter(a.Provider.Filter)
		}

		groupResult, err := groupRequest.Get(a.Context)

		if err != nil {
			azureLogger.Error(err, "Failed to get base group", "Provider", a.Name)
			return nil, err
		}

		aadGroups = append(aadGroups, groupResult...)

	}

	azureURL, err := url.Parse(a.Client.URL())
	if err != nil {
		azureLogger.Error(err, "Failed to parse Azure URL", "URL", a.Client.URL())
		return nil, err
	}

	for _, group := range aadGroups {

		groupName := group.DisplayName

		if groupName == nil {
			azureLogger.Info(fmt.Sprintf("Warning: Skipping Group record with empty displayName"))
			continue
		}

		if !isGroupAllowed(*groupName, a.Provider.Groups) {
			continue
		}

		ocpGroup := userv1.Group{
			TypeMeta: v1.TypeMeta{
				Kind:       "Group",
				APIVersion: userv1.GroupVersion.String(),
			},
			ObjectMeta: v1.ObjectMeta{
				Name:        *groupName,
				Annotations: map[string]string{},
				Labels:      map[string]string{},
			},
			Users: []string{},
		}

		// Set Host Specific Details
		ocpGroup.GetAnnotations()[constants.SyncSourceHost] = azureURL.Host
		ocpGroup.GetAnnotations()[constants.SyncSourceUID] = *group.DirectoryObject.ID

		groupMembers, err := a.listGroupMembers(group.DirectoryObject.ID)

		if err != nil {
			azureLogger.Error(err, "Failed to get Group members for Group", "Group", group.DisplayName, "Provider", a.Name)
			return nil, err
		}

		for _, groupMember := range groupMembers {
			ocpGroup.Users = append(ocpGroup.Users, groupMember)
		}

		ocpGroups = append(ocpGroups, ocpGroup)

	}

	return ocpGroups, nil

}

func (a *AzureSyncer) GetProviderName() string {
	return a.Name
}

func (a *AzureSyncer) listGroupMembers(groupID *string) ([]string, error) {
	groupMembers := []string{}
	memberRequest := a.Client.Groups().ID(*groupID).TransitiveMembers().Request()

	members, err := memberRequest.Get(a.Context)

	if err != nil {
		return nil, err
	}
	for _, member := range members {

		memberODataType, _ := member.GetAdditionalData(GraphOdataType)

		if memberODataType == GraphUserType {
			if username, found := a.getUsernameForUser(member); found {
				groupMembers = append(groupMembers, fmt.Sprintf("%v", username))
			} else {
				azureLogger.Info(fmt.Sprintf("Warning: Username for user cannot be found in Group ID '%v'", *groupID))
			}
		}

	}

	return groupMembers, nil

}

func (a *AzureSyncer) getUsernameForUser(user msgraph.DirectoryObject) (string, bool) {

	if a.Provider.UserNameAttributes == nil {
		return a.isUsernamePresent(user, GraphUserNameAttribute)
	}

	for _, usernameAttribute := range *a.Provider.UserNameAttributes {

		username, found := a.isUsernamePresent(user, usernameAttribute)

		if found {
			return username, true
		}
	}

	return "", false

}

func (a *AzureSyncer) isUsernamePresent(user msgraph.DirectoryObject, field string) (string, bool) {

	value, ok := user.GetAdditionalData(field)

	return fmt.Sprintf("%v", value), ok
}
