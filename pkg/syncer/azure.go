package syncer

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/url"
	"reflect"

	nethttp "net/http"

	userv1 "github.com/openshift/api/user/v1"
	redhatcopv1alpha1 "github.com/redhat-cop/group-sync-operator/api/v1alpha1"
	"github.com/redhat-cop/group-sync-operator/pkg/constants"
	"github.com/redhat-cop/operator-utils/pkg/util"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/cloud"
	azidentity "github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	az "github.com/microsoft/kiota-authentication-azure-go"
	kiota "github.com/microsoft/kiota-http-go"
	msgraphsdk "github.com/microsoftgraph/msgraph-sdk-go"
	msgraphcore "github.com/microsoftgraph/msgraph-sdk-go-core"
	msgroups "github.com/microsoftgraph/msgraph-sdk-go/groups"
	graph "github.com/microsoftgraph/msgraph-sdk-go/models"
)

var (
	azureLogger = logf.Log.WithName("syncer_azure")
	caser       = cases.Title(language.Und, cases.NoLower)
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
	Client            *msgraphsdk.GraphServiceClient
	ReconcilerBase    util.ReconcilerBase
	CredentialsSecret *corev1.Secret
	CachedGroups      map[string]*graph.Group
	CachedGroupUsers  map[string][]*graph.User
	Context           context.Context
	Adapter           *msgraphsdk.GraphRequestAdapter
	CaCertificate     []byte
}

func (a *AzureSyncer) Init() bool {

	a.CachedGroups = make(map[string]*graph.Group)
	a.CachedGroupUsers = make(map[string][]*graph.User)
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
			validationErrors = append(validationErrors, fmt.Errorf("Could not find `AZURE_TENANT_ID` or `AZURE_CLIENT_ID` or `AZURE_CLIENT_SECRET` key in secret '%s' in namespace '%s'", a.Provider.CredentialsSecret.Name, a.Provider.CredentialsSecret.Namespace))
		}

		a.CredentialsSecret = credentialsSecret

	}

	providerCaResource := determineFromDeprecatedObjectRef(a.Provider.Ca, a.Provider.CaSecret)
	if providerCaResource != nil {

		caResource, err := getObjectRefData(a.Context, a.ReconcilerBase.GetClient(), providerCaResource)

		if err != nil {
			validationErrors = append(validationErrors, err)
		}

		var resourceCaKey string
		if providerCaResource.Key != "" {
			resourceCaKey = providerCaResource.Key
		} else {
			resourceCaKey = defaultResourceCaKey
		}

		// Certificate key validation
		if _, found := caResource[resourceCaKey]; !found {
			validationErrors = append(validationErrors, fmt.Errorf("Could not find '%s' key in %s '%s' in namespace '%s'", resourceCaKey, providerCaResource.Kind, providerCaResource.Name, providerCaResource.Namespace))
		}

		a.CaCertificate = caResource[resourceCaKey]
	}

	return utilerrors.NewAggregate(validationErrors)

}

func (a *AzureSyncer) Bind() error {

	var httpClient *nethttp.Client

	if a.Provider.Insecure || len(a.CaCertificate) > 0 {

		httpClient = kiota.GetDefaultClient()

		defaultTransport := nethttp.DefaultTransport.(*nethttp.Transport).Clone()
		defaultTransport.ForceAttemptHTTP2 = true
		defaultTransport.DisableCompression = false

		if a.Provider.Insecure {
			defaultTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		} else {
			if len(a.CaCertificate) > 0 {

				tlsConfig := &tls.Config{}
				if tlsConfig.RootCAs == nil {
					tlsConfig.RootCAs = x509.NewCertPool()
				}

				tlsConfig.RootCAs.AppendCertsFromPEM(a.CaCertificate)

				defaultTransport.TLSClientConfig = tlsConfig

			}
		}

		httpClient.Transport = kiota.NewCustomTransportWithParentTransport(defaultTransport)

	}

	opts := &azidentity.ClientSecretCredentialOptions{}
	opts.Cloud.ActiveDirectoryAuthorityHost = getAuthorityHost(a.Provider.AuthorityHost)
	cred, err := azidentity.NewClientSecretCredential(
		string(a.CredentialsSecret.Data[TenantID]), string(a.CredentialsSecret.Data[ClientID]), string(a.CredentialsSecret.Data[ClientSecret]),
		opts)

	if err != nil {
		return err
	}

	auth, err := az.NewAzureIdentityAuthenticationProvider(cred)

	if err != nil {
		return err
	}

	a.Adapter, err = msgraphsdk.NewGraphRequestAdapterWithParseNodeFactoryAndSerializationWriterFactoryAndHttpClient(auth, nil, nil, httpClient)
	if err != nil {
		return err

	}

	a.Client = msgraphsdk.NewGraphServiceClient(a.Adapter)

	return nil

}

func (a *AzureSyncer) Sync() ([]userv1.Group, error) {

	ocpGroups := []userv1.Group{}
	aadGroups := []graph.Group{}

	if a.Provider.BaseGroups != nil && len(a.Provider.BaseGroups) > 0 {

		for _, baseGroup := range a.Provider.BaseGroups {

			filter := fmt.Sprintf("displayName eq '%s'", baseGroup)
			groupRequestParameters := &msgroups.GroupsRequestBuilderGetQueryParameters{
				Filter: &filter,
			}

			groupRequestConfiguration := &msgroups.GroupsRequestBuilderGetRequestConfiguration{
				QueryParameters: groupRequestParameters,
			}

			baseGroupRequest, err := a.Client.Groups().Get(a.Context, groupRequestConfiguration)

			if err != nil {
				azureLogger.Error(err, "Failed to get base group request", "Provider", a.Name, "Base Group", baseGroup)
				return nil, err
			}

			baseGroupResult, err := a.getGroupsFromResults(baseGroupRequest)

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

			var baseGroupMembersRequestConfiguration *msgroups.GroupsItemMembersRequestBuilderGetRequestConfiguration

			if a.Provider.Filter != "" {
				requestParameters := &msgroups.GroupsItemMembersRequestBuilderGetQueryParameters{
					Filter: &a.Provider.Filter,
				}
				baseGroupMembersRequestConfiguration = &msgroups.GroupsItemMembersRequestBuilderGetRequestConfiguration{
					QueryParameters: requestParameters,
				}

			}

			baseGroupMembersRequest, err := a.Client.GroupsById(*baseGroupResult[0].GetId()).Members().Get(a.Context, baseGroupMembersRequestConfiguration)

			if err != nil {
				azureLogger.Error(err, "Failed to get base group members", "Provider", a.Name, "Base Group", baseGroup)
				return nil, err
			}

			pageIterator, err := msgraphcore.NewPageIterator(baseGroupMembersRequest, &a.Adapter.GraphRequestAdapterBase, graph.CreateGroupCollectionResponseFromDiscriminatorValue)

			if err != nil {
				return nil, err
			}

			err = pageIterator.Iterate(a.Context, func(pageItem interface{}) bool {

				if member, ok := pageItem.(*graph.Group); ok {
					aadGroups = append(aadGroups, *member)
				}
				return true
			})

			if err != nil {
				azureLogger.Error(err, "Failed to get iterate over group members", "Provider", a.Name, "Group ID", *baseGroupResult[0].GetId())
				return nil, err
			}

		}

	} else {

		var groupConfiguration = msgroups.GroupsRequestBuilderGetRequestConfiguration{}

		if a.Provider.Filter != "" {
			groupRequestParameters := &msgroups.GroupsRequestBuilderGetQueryParameters{
				Filter: &a.Provider.Filter,
			}
			groupConfiguration.QueryParameters = groupRequestParameters

		}

		groupRequest, err := a.Client.Groups().Get(a.Context, &groupConfiguration)

		if err != nil {
			azureLogger.Error(err, "Failed to get groups request", "Provider", a.Name)
			return nil, err
		}

		groupResult, err := a.getGroupsFromResults(groupRequest)

		if err != nil {
			azureLogger.Error(err, "Failed to get groups", "Provider", a.Name)
			return nil, err
		}

		aadGroups = append(aadGroups, groupResult...)

	}

	authorityHost := string(getAuthorityHost(a.Provider.AuthorityHost))
	azureURL, err := url.Parse(authorityHost)
	if err != nil {
		azureLogger.Error(err, "Failed to parse Azure URL", "URL", authorityHost)
		return nil, err
	}

	for _, group := range aadGroups {

		groupName := group.GetDisplayName()

		if groupName == nil {
			azureLogger.Info(fmt.Sprintf("Warning: Skipping Group record with empty displayName. Group ID: %s", *group.GetId()))
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
		ocpGroup.GetAnnotations()[constants.SyncSourceUID] = *group.DirectoryObject.GetId()

		groupMembers, err := a.listGroupMembers(group.DirectoryObject.GetId())

		if err != nil {
			azureLogger.Error(err, "Failed to get Group members for Group", "Group", group.GetDisplayName(), "Provider", a.Name)
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
	var groupMembers []string
	var selectParameter []string

	if a.Provider.UserNameAttributes != nil {
		selectParameter = *a.Provider.UserNameAttributes
	} else {
		selectParameter = []string{GraphUserNameAttribute}
	}

	pageSize := int32(999)
	queryParameters := msgroups.GroupsItemTransitiveMembersRequestBuilderGetQueryParameters{
		Select: selectParameter,
		Top:    &pageSize,
	}

	transitiveMembersGetConfiguration := msgroups.GroupsItemTransitiveMembersRequestBuilderGetRequestConfiguration{
		QueryParameters: &queryParameters,
	}

	memberRequest, err := a.Client.GroupsById(*groupID).TransitiveMembers().Get(a.Context, &transitiveMembersGetConfiguration)

	if err != nil {
		return nil, err
	}

	pageIterator, err := msgraphcore.NewPageIterator(memberRequest, &a.Adapter.GraphRequestAdapterBase, graph.CreateGroupCollectionResponseFromDiscriminatorValue)

	if err != nil {
		return nil, err
	}

	err = pageIterator.Iterate(a.Context, func(pageItem interface{}) bool {

		if member, ok := pageItem.(*graph.User); ok {
			if username, found := a.getUsernameForUser(*member); found {
				groupMembers = append(groupMembers, fmt.Sprintf("%v", username))
			}
		}
		return true
	})

	if err != nil {
		azureLogger.Error(err, "Failed to get iterate over group members", "Provider", a.Name, "Group ID", groupID)
		return nil, err
	}

	return groupMembers, nil

}

func (a *AzureSyncer) getUsernameForUser(user graph.User) (string, bool) {

	userValue := reflect.ValueOf(&user)

	if a.Provider.UserNameAttributes == nil {
		return a.isUsernamePresent(userValue, GraphUserNameAttribute)
	}

	for _, usernameAttribute := range *a.Provider.UserNameAttributes {

		username, found := a.isUsernamePresent(userValue, usernameAttribute)

		if found {
			return username, true
		}
	}

	return "", false

}

func (a *AzureSyncer) isUsernamePresent(value reflect.Value, field string) (string, bool) {

	method := value.MethodByName(fmt.Sprintf("Get%s", caser.String(field)))

	if method.IsValid() {
		return fmt.Sprintf("%s", method.Call(nil)[0].Elem().Interface()), true
	}

	return "", false

}

func (a *AzureSyncer) GetPrune() bool {
	return a.Provider.Prune
}

func getAuthorityHost(authorityHost *string) string {

	if authorityHost == nil {
		return cloud.AzurePublic.ActiveDirectoryAuthorityHost

	} else {
		return *authorityHost
	}

}

func (a *AzureSyncer) getGroupsFromResults(result graph.GroupCollectionResponseable) ([]graph.Group, error) {
	groups := []graph.Group{}

	pageIterator, err := msgraphcore.NewPageIterator(result, &a.Adapter.GraphRequestAdapterBase, graph.CreateGroupCollectionResponseFromDiscriminatorValue)

	if err != nil {
		return nil, err
	}

	iterateErr := pageIterator.Iterate(a.Context, func(pageItem interface{}) bool {
		group := pageItem.(*graph.Group)
		groups = append(groups, *group)
		return true
	})

	if iterateErr != nil {
		return nil, iterateErr
	}

	return groups, nil
}
