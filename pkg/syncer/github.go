package syncer

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/google/go-github/v32/github"
	userv1 "github.com/openshift/api/user/v1"
	redhatcopv1alpha1 "github.com/redhat-cop/group-sync-operator/api/v1alpha1"
	"github.com/redhat-cop/group-sync-operator/pkg/constants"
	"github.com/redhat-cop/operator-utils/pkg/util"
	"golang.org/x/oauth2"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	gitHubLogger   = logf.Log.WithName("syncer_github")
	defaultBaseURL = "https://api.github.com/"
)

const (
	pageSize = 100
)

type GitHubSyncer struct {
	Name              string
	GroupSync         *redhatcopv1alpha1.GroupSync
	Provider          *redhatcopv1alpha1.GitHubProvider
	Client            *github.Client
	Context           context.Context
	ReconcilerBase    util.ReconcilerBase
	CredentialsSecret *corev1.Secret
	URL               *url.URL
	CaCertificate     []byte
}

func (g *GitHubSyncer) Init() bool {

	g.Context = context.Background()
	g.URL, _ = url.Parse(defaultBaseURL)

	return false
}

func (g *GitHubSyncer) Validate() error {

	validationErrors := []error{}

	credentialsSecret := &corev1.Secret{}
	err := g.ReconcilerBase.GetClient().Get(context.TODO(), types.NamespacedName{Name: g.Provider.CredentialsSecret.Name, Namespace: g.Provider.CredentialsSecret.Namespace}, credentialsSecret)

	if err != nil {
		validationErrors = append(validationErrors, err)
	} else {

		// Check that provided secret contains required keys
		_, tokenSecretFound := credentialsSecret.Data[secretTokenKey]

		if !tokenSecretFound {
			validationErrors = append(validationErrors, fmt.Errorf("Could not find `token` key in secret '%s' in namespace '%s", g.Provider.CredentialsSecret.Name, g.Provider.CredentialsSecret.Namespace))
		}

		g.CredentialsSecret = credentialsSecret
	}

	if g.Provider.Organization == "" {
		validationErrors = append(validationErrors, fmt.Errorf("Organization name not provided"))
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

		if (*g.Provider.URL)[len(*g.Provider.URL)-1] != '/' {
			validationErrors = append(validationErrors, fmt.Errorf("GitHub URL Must end with a slash ('/')"))
		}

		g.URL, err = url.Parse(*g.Provider.URL)

		if err != nil {
			validationErrors = append(validationErrors, fmt.Errorf("Invalid GitHub URL: '%s", *g.Provider.URL))
		}

	}

	return utilerrors.NewAggregate(validationErrors)
}

func (g *GitHubSyncer) Bind() error {

	tokenSecret, tokenSecretFound := g.CredentialsSecret.Data[secretTokenKey]

	var ghClient *github.Client

	var transport *http.Transport

	if g.Provider.Insecure == true {
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	} else {
		if len(g.CaCertificate) > 0 {

			tlsConfig := &tls.Config{}
			if tlsConfig.RootCAs == nil {
				tlsConfig.RootCAs = x509.NewCertPool()
			}

			tlsConfig.RootCAs.AppendCertsFromPEM(g.CaCertificate)

			transport = &http.Transport{
				TLSClientConfig: tlsConfig,
			}
		}
	}

	if tokenSecretFound {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: string(tokenSecret)})

		if transport != nil {
			g.Context = context.WithValue(g.Context, oauth2.HTTPClient, &http.Client{Transport: transport})
		}

		httpClient := oauth2.NewClient(g.Context, ts)
		ghClient = github.NewClient(httpClient)
	} else {
		return fmt.Errorf("Could not locate credentials in secret '%s' in namespace '%s'", g.Provider.CredentialsSecret.Name, g.Provider.CredentialsSecret.Namespace)
	}

	if g.URL != nil {
		ghClient.BaseURL = g.URL
	}

	g.Client = ghClient

	return nil

}

func (g *GitHubSyncer) Sync() ([]userv1.Group, error) {

	ocpGroups := []userv1.Group{}

	organization, _, err := g.Client.Organizations.Get(g.Context, g.Provider.Organization)

	if err != nil {
		gitHubLogger.Error(err, "Failed to get Organization", "Organization", g.Provider.Organization, "Provider", g.Name)
		return nil, err
	}

	// Get List of Teams in Organization
	teams, err := g.getOrganizationTeams()

	if err != nil {
		gitHubLogger.Error(err, "Failed to get Teams", "Provider", g.Name)
		return nil, err
	}

	for _, team := range teams {
		if !isGroupAllowed(*team.Name, g.Provider.Teams) {
			continue
		}

		ocpGroup := userv1.Group{
			TypeMeta: v1.TypeMeta{
				Kind:       "Group",
				APIVersion: userv1.SchemeGroupVersion.String(),
			},
			ObjectMeta: v1.ObjectMeta{
				Name:        *team.Name,
				Annotations: map[string]string{},
				Labels:      map[string]string{},
			},
			Users: []string{},
		}

		// Set Host Specific Details
		ocpGroup.GetAnnotations()[constants.SyncSourceHost] = g.URL.Host
		ocpGroup.GetAnnotations()[constants.SyncSourceUID] = strconv.FormatInt(*team.ID, 10)

		teamMembers, err := g.listTeamMembers(team.ID, organization.ID)

		if err != nil {
			gitHubLogger.Error(err, "Failed to get Team Member for Team", "Team", team.Name, "Provider", g.Name)
			return nil, err
		}

		for _, teamMember := range teamMembers {
			ocpGroup.Users = append(ocpGroup.Users, *teamMember.Login)
		}

		ocpGroups = append(ocpGroups, ocpGroup)

	}

	return ocpGroups, nil
}

func (g *GitHubSyncer) getOrganizationTeams() ([]*github.Team, error) {
	opts := &github.ListOptions{PerPage: 100}
	var allTeams []*github.Team

	for {
		teams, r, err := g.Client.Teams.ListTeams(g.Context, g.Provider.Organization, opts)

		if err != nil {
			return nil, err
		}

		for _, t := range teams {
			allTeams = append(allTeams, t)
		}

		if r.NextPage == 0 {
			break
		}

		opts.Page = r.NextPage
	}

	return allTeams, nil
}

func (g *GitHubSyncer) listTeamMembers(teamID *int64, organizationID *int64) ([]*github.User, error) {

	teamUsers := []*github.User{}

	opts := github.TeamListTeamMembersOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	for {
		users, resp, err := g.Client.Teams.ListTeamMembersByID(g.Context, *organizationID, *teamID, &opts)
		if err != nil {
			return nil, err
		}
		teamUsers = append(teamUsers, users...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return teamUsers, nil

}

func (g *GitHubSyncer) GetProviderName() string {
	return g.Name
}
