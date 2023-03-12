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
	redhatcopv1alpha1 "github.com/redhat-cop/group-sync-operator/api/v1alpha1"
	"github.com/redhat-cop/group-sync-operator/pkg/constants"
	"github.com/xanzy/go-gitlab"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/redhat-cop/operator-utils/pkg/util"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
)

type GitLabTokenType string

var (
	gitlabLogger = logf.Log.WithName("syncer_gitlab")
)

const (
	JobGitLabTokenType      GitLabTokenType = "job"
	PersonalGitLabTokenType GitLabTokenType = "personal"
	OAuthGitLabTokenType    GitLabTokenType = "oauth"
)

type GitLabSyncer struct {
	Name              string
	GroupSync         *redhatcopv1alpha1.GroupSync
	Provider          *redhatcopv1alpha1.GitLabProvider
	Client            *gitlab.Client
	Context           context.Context
	ReconcilerBase    util.ReconcilerBase
	CredentialsSecret *corev1.Secret
	URL               *url.URL
	CaCertificate     []byte
}

func (g *GitLabSyncer) Init() bool {

	g.Context = context.Background()

	if g.Provider.Scope == "" {
		g.Provider.Scope = redhatcopv1alpha1.SubSyncScope
		return true
	}

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
			validationErrors = append(validationErrors, fmt.Errorf("Could not find 'username' and `password` or `token` key in secret '%s' in namespace '%s'", g.Provider.CredentialsSecret.Name, g.Provider.CredentialsSecret.Namespace))
		}

		g.CredentialsSecret = credentialsSecret

	}

	providerCaResource := determineFromDeprecatedObjectRef(g.Provider.Ca, g.Provider.CaSecret)
	if providerCaResource != nil {

		caResource, err := getObjectRefData(g.Context, g.ReconcilerBase.GetClient(), providerCaResource)

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

		g.CaCertificate = caResource[resourceCaKey]
	}

	if g.Provider.URL != nil {

		g.URL, err = url.Parse(*g.Provider.URL)

		if err != nil {
			validationErrors = append(validationErrors, fmt.Errorf("Invalid GitLab URL: '%s'", *g.Provider.URL))
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
	tokenTypeSecret := g.CredentialsSecret.Data[secretTokenTypeKey]

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

		if string(PersonalGitLabTokenType) == string(tokenTypeSecret) {
			gitlabClient, err = gitlab.NewClient(
				string(tokenSecret),
				clientFns...,
			)
		} else if string(JobGitLabTokenType) == string(tokenTypeSecret) {
			gitlabClient, err = gitlab.NewJobClient(
				string(tokenSecret),
				clientFns...,
			)
		} else {
			gitlabClient, err = gitlab.NewOAuthClient(
				string(tokenSecret),
				clientFns...,
			)
		}

		if err != nil {
			return err
		}
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

		groupMembers, err := g.getGroupMembers(group.ID, g.Provider.Scope)

		if err != nil {
			return nil, err
		}

		ocpGroup := userv1.Group{
			TypeMeta: v1.TypeMeta{
				Kind:       "Group",
				APIVersion: userv1.GroupVersion.String(),
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

			// Get Decendent Groups
			descendantGroups, err := g.getDescendantGroups(t.ID)

			if err != nil {
				return nil, err
			}

			allGroups = append(allGroups, descendantGroups...)

		}

		if resp.CurrentPage >= resp.TotalPages {
			break
		}

		opt.Page = resp.NextPage

	}

	return allGroups, nil

}

func (g *GitLabSyncer) getDescendantGroups(groupId int) ([]*gitlab.Group, error) {

	descendantGroups := []*gitlab.Group{}

	opt := &gitlab.ListDescendantGroupsOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 50,
			Page:    1,
		},
	}

	for {
		groups, resp, err := g.Client.Groups.ListDescendantGroups(groupId, opt)

		if err != nil {
			return nil, err
		}

		descendantGroups = append(descendantGroups, groups...)

		if resp.CurrentPage >= resp.TotalPages {
			break
		}

		opt.Page = resp.NextPage
	}

	return descendantGroups, nil

}

func (g *GitLabSyncer) getGroupMembers(groupId int, scope redhatcopv1alpha1.SyncScope) ([]*gitlab.GroupMember, error) {

	groupMembers := []*gitlab.GroupMember{}

	opt := &gitlab.ListGroupMembersOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 50,
			Page:    1,
		},
	}

	for {

		var members []*gitlab.GroupMember
		var resp *gitlab.Response
		var err error

		if redhatcopv1alpha1.SubSyncScope == scope {
			members, resp, err = g.Client.Groups.ListAllGroupMembers(groupId, opt)
		} else {
			members, resp, err = g.Client.Groups.ListGroupMembers(groupId, opt)
		}

		if err != nil {
			return nil, err
		}

		groupMembers = append(groupMembers, members...)

		if resp.CurrentPage >= resp.TotalPages {
			break
		}

		opt.Page = resp.NextPage
	}

	return groupMembers, nil

}

func (g *GitLabSyncer) GetProviderName() string {
	return g.Name
}

func (g *GitLabSyncer) GetPrune() bool {
	return g.Provider.Prune
}
