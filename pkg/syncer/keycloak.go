package syncer

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/url"

	"crypto/x509"

	"github.com/Nerzal/gocloak/v5"
	userv1 "github.com/openshift/api/user/v1"
	redhatcopv1alpha1 "github.com/redhat-cop/group-sync-operator/api/v1alpha1"
	"github.com/redhat-cop/group-sync-operator/pkg/constants"
	"github.com/redhat-cop/operator-utils/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	masterRealm = "master"
)

var (
	keycloakLogger = logf.Log.WithName("syncer_keycloak")
	truthy         = true
	iterationMax   = 100

	errGroupNameContainsSeparator = errors.New("group name contains separator")
)

type KeycloakSyncer struct {
	Name               string
	GroupSync          *redhatcopv1alpha1.GroupSync
	Provider           *redhatcopv1alpha1.KeycloakProvider
	GoCloak            gocloak.GoCloak
	Token              *gocloak.JWT
	CachedGroups       map[string]*gocloak.Group
	CachedGroupMembers map[string][]*gocloak.User
	ReconcilerBase     util.ReconcilerBase
	CredentialsSecret  *corev1.Secret
	CaCertificate      []byte
}

func (k *KeycloakSyncer) Init() bool {

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

	if k.Provider.SubGroupProcessing == "" {
		k.Provider.SubGroupProcessing = redhatcopv1alpha1.FlatSubGroupProcessing
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

	if k.Provider.CaSecret != nil {
		caSecret := &corev1.Secret{}
		err := k.ReconcilerBase.GetClient().Get(context.TODO(), types.NamespacedName{Name: k.Provider.CaSecret.Name, Namespace: k.Provider.CaSecret.Namespace}, caSecret)

		if err != nil {
			validationErrors = append(validationErrors, err)
		}

		var secretCaKey string
		if k.Provider.CaSecret.Key != "" {
			secretCaKey = k.Provider.CaSecret.Key
		} else {
			secretCaKey = defaultSecretCaKey
		}

		// Password key validation
		if _, found := caSecret.Data[secretCaKey]; !found {
			validationErrors = append(validationErrors, fmt.Errorf("Could not find '%s' key in secret '%s' in namespace '%s", secretCaKey, k.Provider.CaSecret.Name, k.Provider.CaSecret.Namespace))
		}

		k.CaCertificate = caSecret.Data[secretCaKey]

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

	mapper := KeycloakGroupMapper{
		GetGroupMembers: k.getGroupMembers,

		AllowedGroups:         k.Provider.Groups,
		Scope:                 k.Provider.Scope,
		SubGroupProcessing:    k.Provider.SubGroupProcessing,
		SubGroupJoinSeparator: k.Provider.SubGroupJoinSeparator,
	}

	ocpGroups, err := mapper.Map(groups)
	if err != nil {
		return nil, fmt.Errorf("error mapping keycloak groups: %w", err)
	}

	url, err := url.Parse(k.Provider.URL)
	if err != nil {
		return nil, err
	}

	for _, g := range ocpGroups {
		g.GetAnnotations()[constants.SyncSourceHost] = url.Host
	}

	return ocpGroups, nil
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
