package syncer

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/url"
	"strings"

	"crypto/x509"

	"github.com/Nerzal/gocloak/v5"
	userv1 "github.com/openshift/api/user/v1"
	redhatcopv1alpha1 "github.com/redhat-cop/group-sync-operator/api/v1alpha1"
	"github.com/redhat-cop/group-sync-operator/pkg/constants"
	"github.com/redhat-cop/operator-utils/pkg/util"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/validation"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	masterRealm = "master"
)

var (
	keycloakLogger = logf.Log.WithName("syncer_keycloak")
	truthy         = true
	iterationMax   = 100
)

type KeycloakSyncer struct {
	Name               string
	GroupSync          *redhatcopv1alpha1.GroupSync
	Provider           *redhatcopv1alpha1.KeycloakProvider
	GoCloak            gocloak.GoCloak
	Context            context.Context
	Token              *gocloak.JWT
	CachedGroups       map[string]*gocloak.Group
	CachedGroupMembers map[string][]*gocloak.User
	ReconcilerBase     util.ReconcilerBase
	CredentialsSecret  *corev1.Secret
	CaCertificate      []byte
}

func (k *KeycloakSyncer) Init() bool {

	k.Context = context.Background()

	changed := false

	k.CachedGroupMembers = make(map[string][]*gocloak.User)
	k.CachedGroups = make(map[string]*gocloak.Group)
	k.GoCloak = gocloak.NewClient(k.Provider.URL)

	if k.Provider.LoginRealm == "" {
		k.Provider.LoginRealm = masterRealm
		changed = true
	}

	if k.Provider.Scope == "" {
		k.Provider.Scope = redhatcopv1alpha1.SubSyncScope
		changed = true
	}

	return changed

}

func (k *KeycloakSyncer) Validate() error {

	validationErrors := []error{}

	// Verify Secret Containing Username and Password Exists with Valid Keys
	credentialsSecret := &corev1.Secret{}
	err := k.ReconcilerBase.GetClient().Get(context.TODO(), types.NamespacedName{Name: k.Provider.CredentialsSecret.Name, Namespace: k.Provider.CredentialsSecret.Namespace}, credentialsSecret)

	if err != nil {
		validationErrors = append(validationErrors, err)
	} else {

		// Username key validation
		if _, found := credentialsSecret.Data[secretUsernameKey]; !found {
			validationErrors = append(validationErrors, fmt.Errorf("Could not find 'username' key in secret '%s' in namespace '%s", k.Provider.CredentialsSecret.Name, k.Provider.CredentialsSecret.Namespace))
		}

		// Password key validation
		if _, found := credentialsSecret.Data[secretPasswordKey]; !found {
			validationErrors = append(validationErrors, fmt.Errorf("Could not find 'password' key in secret '%s' in namespace '%s", k.Provider.CredentialsSecret.Name, k.Provider.CredentialsSecret.Namespace))
		}

		k.CredentialsSecret = credentialsSecret

	}

	if _, err := url.ParseRequestURI(k.Provider.URL); err != nil {
		validationErrors = append(validationErrors, err)
	}

	providerCaResource := determineFromDeprecatedObjectRef(k.Provider.Ca, k.Provider.CaSecret)
	if providerCaResource != nil {

		caResource, err := getObjectRefData(k.Context, k.ReconcilerBase.GetClient(), providerCaResource)

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
			validationErrors = append(validationErrors, fmt.Errorf("Could not find '%s' key in %s '%s' in namespace '%s", resourceCaKey, providerCaResource.Kind, providerCaResource.Name, providerCaResource.Namespace))
		}

		k.CaCertificate = caResource[resourceCaKey]
	}

	return utilerrors.NewAggregate(validationErrors)

}

func (k *KeycloakSyncer) Bind() error {

	restyClient := k.GoCloak.RestyClient()

	if k.Provider.Insecure == true {
		restyClient.SetTLSClientConfig(&tls.Config{InsecureSkipVerify: true})
	}

	// Add trusted certificate if provided
	if len(k.CaCertificate) > 0 {

		tlsConfig := &tls.Config{}
		if tlsConfig.RootCAs == nil {
			tlsConfig.RootCAs = x509.NewCertPool()
		}

		tlsConfig.RootCAs.AppendCertsFromPEM(k.CaCertificate)

		restyClient.SetTLSClientConfig(tlsConfig)
	}

	k.GoCloak.SetRestyClient(restyClient)

	token, err := k.GoCloak.LoginAdmin(string(k.CredentialsSecret.Data[secretUsernameKey]), string(k.CredentialsSecret.Data[secretPasswordKey]), k.Provider.LoginRealm)

	k.Token = token

	if err != nil {
		return err
	}

	keycloakLogger.Info("Successfully Authenticated with Keycloak Provider")

	return nil
}

func (k *KeycloakSyncer) Sync() ([]userv1.Group, error) {

	// Get Groups
	groups, err := k.getGroups()

	if err != nil {
		keycloakLogger.Error(err, "Failed to get Groups", "Provider", k.Name)
		return nil, err
	}

	for _, group := range groups {

		if _, groupFound := k.CachedGroups[*group.ID]; !groupFound {
			k.processGroupsAndMembers(group, nil, k.Provider.Scope)
		}
	}

	ocpGroups := []userv1.Group{}

	for _, cachedGroup := range k.CachedGroups {

		groupAttributes := map[string]string{}

		for key, value := range cachedGroup.Attributes {
			// we add the annotation that qualify for OCP annotations and log for the ones that don't
			if errs := validation.IsQualifiedName(key); len(errs) == 0 {
				groupAttributes[key] = strings.Join(value, "'")
			} else {
				keycloakLogger.Info("unable to add annotation to", "group", cachedGroup.Name, "key", key, "value", value)
			}
		}

		ocpGroup := userv1.Group{
			TypeMeta: v1.TypeMeta{
				Kind:       "Group",
				APIVersion: userv1.GroupVersion.String(),
			},
			ObjectMeta: v1.ObjectMeta{
				Name:        *cachedGroup.Name,
				Annotations: groupAttributes,
				Labels:      map[string]string{},
			},
			Users: []string{},
		}

		url, err := url.Parse(k.Provider.URL)

		if err != nil {
			return nil, err
		}

		childrenGroups := []string{}

		for _, subgroup := range cachedGroup.SubGroups {
			childrenGroups = append(childrenGroups, *subgroup.Name)
		}

		parentGroups := []string{}

		for _, group := range k.CachedGroups {
			for _, subgroup := range group.SubGroups {
				if *subgroup.Name == *cachedGroup.Name {
					parentGroups = append(parentGroups, *group.Name)
				}
			}
		}

		// Set Host Specific Details
		ocpGroup.GetAnnotations()[constants.SyncSourceHost] = url.Host
		ocpGroup.GetAnnotations()[constants.SyncSourceUID] = *cachedGroup.ID
		if len(childrenGroups) > 0 {
			ocpGroup.GetAnnotations()[constants.HierarchyChildren] = strings.Join(childrenGroups, ",")
		}
		if len(parentGroups) == 1 {
			ocpGroup.GetAnnotations()[constants.HierarchyParent] = parentGroups[0]
		}
		if len(parentGroups) > 1 {
			ocpGroup.GetAnnotations()[constants.HierarchyParents] = strings.Join(parentGroups, ",")
		}

		for _, user := range k.CachedGroupMembers[*cachedGroup.ID] {
			ocpGroup.Users = append(ocpGroup.Users, *user.Username)
		}

		ocpGroups = append(ocpGroups, ocpGroup)

	}

	return ocpGroups, nil
}

func (k *KeycloakSyncer) processGroupsAndMembers(group, parentGroup *gocloak.Group, scope redhatcopv1alpha1.SyncScope) error {

	if parentGroup == nil && !isGroupAllowed(*group.Name, k.Provider.Groups) {
		return nil
	}

	k.CachedGroups[*group.ID] = group

	groupMembers, err := k.getGroupMembers(*group.ID)

	if err != nil {
		return err
	}

	k.CachedGroupMembers[*group.ID] = groupMembers

	// Add Group Members to Primary Group
	if parentGroup != nil {
		usersToAdd, _ := k.diff(groupMembers, k.CachedGroupMembers[*parentGroup.ID])
		k.CachedGroupMembers[*parentGroup.ID] = append(k.CachedGroupMembers[*parentGroup.ID], usersToAdd...)
	}

	// Process Subgroups
	if redhatcopv1alpha1.SubSyncScope == scope {
		for _, subGroup := range group.SubGroups {
			if _, subGroupFound := k.CachedGroups[*subGroup.ID]; !subGroupFound {
				k.processGroupsAndMembers(subGroup, group, scope)
			}
		}
	}

	return nil
}

func (k *KeycloakSyncer) diff(lhsSlice, rhsSlice []*gocloak.User) (lhsOnly []*gocloak.User, rhsOnly []*gocloak.User) {
	return k.singleDiff(lhsSlice, rhsSlice), k.singleDiff(rhsSlice, lhsSlice)
}

func (k *KeycloakSyncer) singleDiff(lhsSlice, rhsSlice []*gocloak.User) (lhsOnly []*gocloak.User) {
	for _, lhs := range lhsSlice {
		found := false
		for _, rhs := range rhsSlice {
			if *lhs.ID == *rhs.ID {
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

func (k *KeycloakSyncer) setGroupsAttributes(groups []*gocloak.Group) {
	for _, group := range groups {

		g, err := k.GoCloak.GetGroup(k.Token.AccessToken, k.Provider.Realm, *group.ID)

		if err != nil {
			continue
		}

		if g.Attributes != nil && len(g.Attributes) > 0 {
			group.Attributes = g.Attributes
		}

		if group.SubGroups != nil && len(group.SubGroups) > 0 {
			k.setGroupsAttributes(group.SubGroups)
		}
	}
}

func (k *KeycloakSyncer) getGroups() ([]*gocloak.Group, error) {
	groups := []*gocloak.Group{}

	iteration := 0

	for {
		gIteration := iteration * iterationMax
		groupsParams := gocloak.GetGroupsParams{First: &gIteration, Max: &iterationMax, BriefRepresentation: &truthy}
		groupsResponse, err := k.GoCloak.GetGroups(k.Token.AccessToken, k.Provider.Realm, groupsParams)

		if err != nil {
			return nil, err
		}

		if len(groupsResponse) == 0 {
			break
		}

		k.setGroupsAttributes(groupsResponse)

		groups = append(groups, groupsResponse...)
		iteration = iteration + 1

	}

	return groups, nil
}

func (k *KeycloakSyncer) getGroupMembers(groupId string) ([]*gocloak.User, error) {
	members := []*gocloak.User{}

	iteration := 0

	for {

		uIteration := iteration * iterationMax
		groupMemberParams := gocloak.GetGroupsParams{First: &uIteration, Max: &iterationMax, BriefRepresentation: &truthy}
		groupMembers, err := k.GoCloak.GetGroupMembers(k.Token.AccessToken, k.Provider.Realm, groupId, groupMemberParams)

		if err != nil {
			return nil, err
		}

		if len(groupMembers) == 0 {
			break
		}

		members = append(members, groupMembers...)
		iteration = iteration + 1

	}

	return members, nil
}

func (k *KeycloakSyncer) GetProviderName() string {
	return k.Name
}
