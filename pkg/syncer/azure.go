package syncer

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"time"

	nethttp "net/http"

	abstractions "github.com/microsoft/kiota-abstractions-go"
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

	"github.com/google/cel-go/cel"
)

var (
	azureLogger   = logf.Log.WithName("syncer_azure")
	caser         = cases.Title(language.Und, cases.NoLower)
	azurePageSize = int32(999)
	headers       = abstractions.NewRequestHeaders()
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
	compiledFilter    cel.Program
}

func (a *AzureSyncer) Init() bool {

	a.CachedGroups = make(map[string]*graph.Group)
	a.CachedGroupUsers = make(map[string][]*graph.User)
	a.Context = context.Background()
	headers.Add("ConsistencyLevel", "eventual")

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

	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	defaultTransport := &nethttp.Transport{
		Proxy:                 nethttp.ProxyFromEnvironment,
		DialContext:           dialer.DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	if a.Provider.Insecure || len(a.CaCertificate) > 0 {

		httpClient = kiota.GetDefaultClient()

		var tlsConfig *tls.Config

		if a.Provider.Insecure {
			tlsConfig = &tls.Config{InsecureSkipVerify: true}
		} else {
			if len(a.CaCertificate) > 0 {

				tlsConfig = &tls.Config{}
				if tlsConfig.RootCAs == nil {
					tlsConfig.RootCAs = x509.NewCertPool()
				}

				tlsConfig.RootCAs.AppendCertsFromPEM(a.CaCertificate)

			}
		}

		defaultTransport.TLSClientConfig = tlsConfig

		httpClient.Transport = kiota.NewCustomTransportWithParentTransport(defaultTransport)

	}

	opts := &azidentity.ClientSecretCredentialOptions{}
	opts.Cloud.ActiveDirectoryAuthorityHost = getAuthorityHost(a.Provider.AuthorityHost)
	opts.Transport = &nethttp.Client{
		Transport: defaultTransport,
	}
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

	// Compile CEL client filter if provided
	if a.Provider.ClientFilter != "" {
		err = a.compileClientFilter()
		if err != nil {
			return fmt.Errorf("failed to compile client filter: %w", err)
		}
	}

	return nil

}

func (a *AzureSyncer) Sync() ([]userv1.Group, error) {

	if err := a.probeClientFilter(); err != nil {
		return nil, err
	}

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

			var baseGroupMembersRequestConfiguration *msgroups.ItemMembersGraphGroupRequestBuilderGetRequestConfiguration

			if a.Provider.Filter != "" {
				requestParameters := &msgroups.ItemMembersGraphGroupRequestBuilderGetQueryParameters{
					Filter: &a.Provider.Filter,
					Top:    &azurePageSize,
				}
				baseGroupMembersRequestConfiguration = &msgroups.ItemMembersGraphGroupRequestBuilderGetRequestConfiguration{
					QueryParameters: requestParameters,
				}

			}

			baseGroupMembersRequest, err := a.Client.GroupsById(*baseGroupResult[0].GetId()).Members().GraphGroup().Get(a.Context, baseGroupMembersRequestConfiguration)

			if err != nil {
				azureLogger.Error(err, "Failed to get base group members", "Provider", a.Name, "Base Group", baseGroup)
				return nil, err
			}

			pageIterator, err := msgraphcore.NewPageIterator[*graph.Group](baseGroupMembersRequest, &a.Adapter.GraphRequestAdapterBase, graph.CreateGroupCollectionResponseFromDiscriminatorValue)

			if err != nil {
				return nil, err
			}

			err = pageIterator.Iterate(a.Context, func(group *graph.Group) bool {

				aadGroups = append(aadGroups, *group)
				return true
			})

			if err != nil {
				azureLogger.Error(err, "Failed to get iterate over group members", "Provider", a.Name, "Group ID", *baseGroupResult[0].GetId())
				return nil, err
			}

		}

	} else {

		var groupConfiguration = msgroups.GroupsRequestBuilderGetRequestConfiguration{
			QueryParameters: &msgroups.GroupsRequestBuilderGetQueryParameters{
				Top: &azurePageSize,
			},
		}

		if a.Provider.Filter != "" {
			groupRequestParameters := &msgroups.GroupsRequestBuilderGetQueryParameters{
				Filter: &a.Provider.Filter,
				Top:    &azurePageSize,
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

		// Apply client-side CEL filter if configured
		if shouldInclude, err := a.evaluateClientFilter(group); err != nil {
			azureLogger.Error(err, "Failed to evaluate client filter, skipping group", "Group", *groupName, "Provider", a.Name)
			continue
		} else if !shouldInclude {
			azureLogger.V(1).Info("Group filtered by clientFilter", "Group", *groupName, "Provider", a.Name)
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

	queryParameters := msgroups.ItemTransitiveMembersGraphUserRequestBuilderGetQueryParameters{
		Select: selectParameter,
		Top:    &azurePageSize,
		Count:  &truthy,
	}

	transitiveMembersConfiguration := msgroups.ItemTransitiveMembersGraphUserRequestBuilderGetRequestConfiguration{
		QueryParameters: &queryParameters,
		Headers:         headers,
	}

	memberRequest, err := a.Client.GroupsById(*groupID).TransitiveMembers().GraphUser().Get(a.Context, &transitiveMembersConfiguration)

	if err != nil {
		return nil, err
	}

	for {

		for _, member := range memberRequest.GetValue() {
			if username, found := a.getUsernameForUser(member); found {
				groupMembers = append(groupMembers, fmt.Sprintf("%v", username))
			}
		}

		nextPageUrl := memberRequest.GetOdataNextLink()
		if nextPageUrl != nil {
			transitiveMembersConfiguration := msgroups.ItemTransitiveMembersGraphUserRequestBuilderGetRequestConfiguration{
				Headers: headers,
			}

			memberRequest, err = msgroups.NewItemTransitiveMembersGraphUserRequestBuilder(*nextPageUrl, a.Client.GetAdapter()).Get(context.Background(), &transitiveMembersConfiguration)

			if err != nil {
				azureLogger.Error(err, "Failed to get iterate over group members", "Provider", a.Name, "Group ID", groupID)
				return nil, err
			}
		} else {
			break
		}

	}

	return groupMembers, nil

}

func (a *AzureSyncer) getUsernameForUser(user graph.Userable) (string, bool) {

	userValue := reflect.ValueOf(user)

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

		attr := method.Call(nil)[0]

		if !attr.IsNil() {
			return fmt.Sprintf("%s", attr.Elem().Interface()), true
		} else {
			azureLogger.Info(fmt.Sprintf("Warning: Skipping User record with empty %s.", field))
		}
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

	pageIterator, err := msgraphcore.NewPageIterator[*graph.Group](result, &a.Adapter.GraphRequestAdapterBase, graph.CreateGroupCollectionResponseFromDiscriminatorValue)

	if err != nil {
		return nil, err
	}

	iterateErr := pageIterator.Iterate(a.Context, func(group *graph.Group) bool {
		groups = append(groups, *group)
		return true
	})

	if iterateErr != nil {
		return nil, iterateErr
	}

	return groups, nil
}

// celCompatibleKinds lists the reflect.Kind values that are safe to pass to CEL.
// Complex SDK types (interfaces, structs, func maps) are excluded intentionally.
var celCompatibleKinds = map[reflect.Kind]bool{
	reflect.String: true,
	reflect.Bool:   true,
	reflect.Int32:  true,
}

// zeroForType returns the zero value used when a getter returns nil,
// ensuring CEL always receives a concrete typed value instead of nil.
func zeroForType(t reflect.Type) interface{} {
	switch t.Kind() {
	case reflect.String:
		return ""
	case reflect.Bool:
		return false
	case reflect.Int32:
		return int32(0)
	}
	if t == reflect.TypeOf(time.Time{}) {
		return time.Time{}
	}
	// []string
	if t.Kind() == reflect.Slice && t.Elem().Kind() == reflect.String {
		return []string{}
	}
	return nil
}

// extractGroupFields dynamically extracts all CEL-compatible scalar fields from
// a Graph Group by reflecting over its Get* methods. Only methods that:
//   - take no arguments
//   - return exactly one value
//   - return *string, *bool, *int32, *time.Time, or []string
//
// are included. Complex SDK types (collections of objects, interfaces, func maps)
// are silently skipped. Absent (nil pointer) fields are replaced with their zero
// value so CEL expressions never receive nil.
func extractGroupFields(group graph.Group) map[string]interface{} {
	result := make(map[string]interface{})

	// All Get* methods on graph.Group use pointer receivers, so we must reflect
	// on the pointer to see the full method set.
	val := reflect.ValueOf(&group)
	typ := val.Type()

	for i := 0; i < val.NumMethod(); i++ {
		methodType := typ.Method(i)
		if !strings.HasPrefix(methodType.Name, "Get") {
			continue
		}

		// Only zero-argument methods: on a concrete type, Type.Method includes
		// the receiver as in[0], so NumIn()==1 means no extra arguments.
		mt := methodType.Type
		if mt.NumIn() != 1 || mt.NumOut() != 1 {
			continue
		}

		retType := mt.Out(0)

		// Determine if the return type is CEL-compatible and extract value
		var fieldValue interface{}
		ret := val.Method(i).Call(nil)[0]

		switch {
		case retType.Kind() == reflect.Ptr && celCompatibleKinds[retType.Elem().Kind()]:
			// *string, *bool, *int32 — dereference or use zero value
			if ret.IsNil() {
				fieldValue = zeroForType(retType.Elem())
			} else {
				fieldValue = ret.Elem().Interface()
			}
		case retType.Kind() == reflect.Ptr && retType.Elem() == reflect.TypeOf(time.Time{}):
			// *time.Time
			if ret.IsNil() {
				fieldValue = time.Time{}
			} else {
				fieldValue = ret.Elem().Interface()
			}
		case retType.Kind() == reflect.Slice && retType.Elem().Kind() == reflect.String:
			// []string
			if ret.IsNil() {
				fieldValue = []string{}
			} else {
				fieldValue = ret.Interface()
			}
		default:
			// Complex SDK type — skip entirely
			continue
		}

		// Convert "GetDisplayName" → "displayName"
		name := methodType.Name[3:]
		if len(name) == 0 {
			continue
		}
		fieldName := strings.ToLower(name[:1]) + name[1:]
		result[fieldName] = fieldValue
	}

	return result
}

// probeClientFilter evaluates the compiled filter against a zero-value group map
// to catch field name and type errors before processing any real groups.
// The available field list is included in any error message.
func (a *AzureSyncer) probeClientFilter() error {
	if a.compiledFilter == nil {
		return nil
	}
	probe := extractGroupFields(*graph.NewGroup())
	_, _, err := a.compiledFilter.Eval(map[string]interface{}{"group": probe})
	if err != nil {
		keys := make([]string, 0, len(probe))
		for k := range probe {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		return fmt.Errorf("clientFilter expression error: %w\navailable fields: %v", err, keys)
	}
	return nil
}

// compileClientFilter compiles the CEL expression for client-side filtering.
func (a *AzureSyncer) compileClientFilter() error {
	env, err := cel.NewEnv(
		cel.Variable("group", cel.MapType(cel.StringType, cel.AnyType)),
	)
	if err != nil {
		return err
	}

	ast, issues := env.Compile(a.Provider.ClientFilter)
	if issues != nil && issues.Err() != nil {
		return issues.Err()
	}

	a.compiledFilter, err = env.Program(ast)
	if err != nil {
		return err
	}

	// Log the available fields so operators can see them in pod logs
	fields := extractGroupFields(*graph.NewGroup())
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	azureLogger.Info("CEL clientFilter compiled", "availableFields", keys)

	return nil
}

// evaluateClientFilter evaluates the CEL filter against a group.
func (a *AzureSyncer) evaluateClientFilter(group graph.Group) (bool, error) {
	if a.compiledFilter == nil {
		return true, nil // No filter means include all
	}

	groupData := extractGroupFields(group)

	out, _, err := a.compiledFilter.Eval(map[string]interface{}{
		"group": groupData,
	})
	if err != nil {
		keys := make([]string, 0, len(groupData))
		for k := range groupData {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		return false, fmt.Errorf("clientFilter evaluation error: %w (available fields: %v)", err, keys)
	}

	result, ok := out.Value().(bool)
	if !ok {
		return false, fmt.Errorf("client filter did not return a boolean value")
	}

	return result, nil
}

