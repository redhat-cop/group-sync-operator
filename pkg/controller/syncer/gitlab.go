package syncer

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/hashicorp/go-cleanhttp"
	userv1 "github.com/openshift/api/user/v1"
	redhatcopv1alpha1 "github.com/redhat-cop/group-sync-operator/pkg/apis/redhatcop/v1alpha1"
	"github.com/redhat-cop/group-sync-operator/pkg/controller/constants"
	"github.com/xanzy/go-gitlab"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/redhat-cop/operator-utils/pkg/util"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
)

var (
	gitlabLogger = logf.Log.WithName("syncer_gitlab")
)

type GitLabSyncer struct {
	Name              string
	GroupSync         *redhatcopv1alpha1.GroupSync
	Provider          *redhatcopv1alpha1.GitLabProvider
	Client            *gitlab.Client
	ReconcilerBase    util.ReconcilerBase
	CredentialsSecret *corev1.Secret
	URL               *url.URL
	CaCertificate     []byte
}

func (g *GitLabSyncer) Init() bool {

	return false
}

func (g *GitLabSyncer) Validate() error {

	validationErrors := []error{}

	credentialsSecret := &corev1.Secret{}
	err := g.ReconcilerBase.GetClient().Get(context.TODO(), types.NamespacedName{Name: g.Provider.CredentialsSecret.Name, Namespace: g.Provider.CredentialsSecret.Namespace}, credentialsSecret)

	if err != nil {
		validationErrors = append(validationErrors, err)
	} else {

		// Check that provided secret contains required keys
		_, usernameSecretFound := credentialsSecret.Data[secretUsernameKey]
		_, passwordSecretFound := credentialsSecret.Data[secretPasswordKey]
		_, tokenSecretFound := credentialsSecret.Data[secretTokenKey]

		if !(usernameSecretFound && passwordSecretFound) && !tokenSecretFound {
			validationErrors = append(validationErrors, fmt.Errorf("Could not find 'username' and `password` or `token` key in secret '%s' in namespace '%s", g.Provider.CredentialsSecret.Name, g.Provider.CredentialsSecret.Namespace))
		}

		g.CredentialsSecret = credentialsSecret

	}

	if g.Provider.CaSecret != nil {
		caSecret := &corev1.Secret{}
		err := g.ReconcilerBase.GetClient().Get(context.TODO(), types.NamespacedName{Name: g.Provider.CaSecret.Name, Namespace: g.Provider.CaSecret.Namespace}, caSecret)

		if err != nil {
			validationErrors = append(validationErrors, err)
		}

		var secretCaKey string
		if g.Provider.CaSecret.Key != "" {
			secretCaKey = g.Provider.CaSecret.Key
		} else {
			secretCaKey = defaultSecretCaKey
		}

		// Certificate key validation
		if _, found := caSecret.Data[secretCaKey]; !found {
			validationErrors = append(validationErrors, fmt.Errorf("Could not find '%s' key in secret '%s' in namespace '%s", secretCaKey, g.Provider.CaSecret.Name, g.Provider.CaSecret.Namespace))
		}

		g.CaCertificate = caSecret.Data[secretCaKey]

	}

	if g.Provider.URL != nil {

		g.URL, err = url.Parse(*g.Provider.URL)

		if err != nil {
			validationErrors = append(validationErrors, fmt.Errorf("Invalid GitLab URL: '%s", *g.Provider.URL))
		}

	}

	return utilerrors.NewAggregate(validationErrors)
}

func (g *GitLabSyncer) Bind() error {

	var gitlabClient *gitlab.Client
	var err error

	usernameSecret, usernameSecretFound := g.CredentialsSecret.Data[secretUsernameKey]
	passwordSecret, passwordSecretFound := g.CredentialsSecret.Data[secretPasswordKey]
	tokenSecret, tokenSecretFound := g.CredentialsSecret.Data[secretTokenKey]

	clientFns := []gitlab.ClientOptionFunc{}

	if g.URL != nil {
		clientFns = append(clientFns, gitlab.WithBaseURL(g.URL.String()))
	}

	if g.Provider.Insecure == true || len(g.CaCertificate) > 0 {

		transport := cleanhttp.DefaultPooledTransport()

		if g.Provider.Insecure == true {
			transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
		} else if g.CaCertificate != nil {
			tlsConfig := &tls.Config{}
			if tlsConfig.RootCAs == nil {
				tlsConfig.RootCAs = x509.NewCertPool()
			}

			tlsConfig.RootCAs.AppendCertsFromPEM(g.CaCertificate)

			transport.TLSClientConfig = tlsConfig

		}

		clientFns = append(clientFns, gitlab.WithHTTPClient(&http.Client{Transport: transport}))
	}

	if tokenSecretFound {
		gitlabClient, err = gitlab.NewOAuthClient(
			string(tokenSecret),
			clientFns...,
		)
	} else if usernameSecretFound && passwordSecretFound {
		gitlabClient, err = gitlab.NewBasicAuthClient(
			string(usernameSecret),
			string(passwordSecret),
			clientFns...,
		)

		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("Could not locate credentials in secret '%s' in namespace '%s'", g.Provider.CredentialsSecret.Name, g.Provider.CredentialsSecret.Namespace)
	}

	g.Client = gitlabClient

	return nil

}

func (g *GitLabSyncer) Sync() ([]userv1.Group, error) {

	ocpGroups := []userv1.Group{}

	groups, err := g.getGroups()

	if err != nil {
		return nil, err
	}

	for _, group := range groups {

		if !isGroupAllowed(group.Name, g.Provider.Groups) {
			continue
		}

		groupMembers, err := g.getGroupMembers(group.ID)

		if err != nil {
			return nil, err
		}

		ocpGroup := userv1.Group{
			TypeMeta: v1.TypeMeta{
				Kind:       "Group",
				APIVersion: userv1.SchemeGroupVersion.String(),
			},
			ObjectMeta: v1.ObjectMeta{
				Name:        group.Name,
				Annotations: map[string]string{},
				Labels:      map[string]string{},
			},
			Users: []string{},
		}

		// Set Host Specific Details
		ocpGroup.GetAnnotations()[constants.SyncSourceHost] = g.URL.Host
		ocpGroup.GetAnnotations()[constants.SyncSourceUID] = strconv.Itoa(group.ID)

		for _, groupMember := range groupMembers {
			ocpGroup.Users = append(ocpGroup.Users, groupMember.Username)
		}

		ocpGroups = append(ocpGroups, ocpGroup)

	}

	return ocpGroups, nil

}

func (g *GitLabSyncer) getGroups() ([]*gitlab.Group, error) {

	var allGroups []*gitlab.Group

	opt := &gitlab.ListGroupsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 50,
			Page:    1,
		},
	}

	for {

		groups, resp, err := g.Client.Groups.ListGroups(opt)

		if err != nil {
			return nil, err
		}

		for _, t := range groups {
			allGroups = append(allGroups, t)
		}

		if resp.CurrentPage >= resp.TotalPages {
			break
		}

		opt.Page = resp.NextPage

	}

	return allGroups, nil

}

func (g *GitLabSyncer) getGroupMembers(groupId int) ([]*gitlab.GroupMember, error) {

	groupMembers := []*gitlab.GroupMember{}

	opt := &gitlab.ListGroupMembersOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 50,
			Page:    1,
		},
	}

	for {
		members, resp, err := g.Client.Groups.ListAllGroupMembers(groupId, opt)

		if err != nil {
			return nil, err
		}

		for _, u := range members {
			groupMembers = append(groupMembers, u)
		}

		if resp.CurrentPage >= resp.TotalPages {
			break
		}
	}

	return groupMembers, nil

}

func (g *GitLabSyncer) GetProviderName() string {
	return g.Name
}
