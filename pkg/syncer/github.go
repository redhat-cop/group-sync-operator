package syncer

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gregjones/httpcache"
	"github.com/shurcooL/githubv4"

	"github.com/google/go-github/v39/github"
	userv1 "github.com/openshift/api/user/v1"
	"github.com/palantir/go-githubapp/githubapp"
	redhatcopv1alpha1 "github.com/redhat-cop/group-sync-operator/api/v1alpha1"
	"github.com/redhat-cop/group-sync-operator/pkg/constants"
	"github.com/redhat-cop/operator-utils/pkg/util"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	gitHubLogger   = logf.Log.WithName("syncer_github")
	defaultBaseURL = "https://api.github.com/"
	scimQuery      struct {
		Organization struct {
			SamlIdentityProvider struct {
				ExternalIdentities struct {
					PageInfo struct {
						HasNextPage githubv4.Boolean
						EndCursor   githubv4.String
					} `graphql:"pageInfo"`
					Edges []struct {
						Node struct {
							SamlIdentity struct {
								NameId   githubv4.String
								Username githubv4.String
							} `graphql:"samlIdentity"`
							User struct {
								Login githubv4.String
							} `graphql:"user"`
						} `graphql:"node"`
					} `graphql:"edges"`
				} `graphql:"externalIdentities(first: $first, after: $after)"`
			} `graphql:"samlIdentityProvider"`
		} `graphql:"organization(login: $organization)"`
	}
)

const (
	pageSize  = 100
	userAgent = "redhat-cop/group-sync-operator"
)

type GitHubSyncer struct {
	Name              string
	GroupSync         *redhatcopv1alpha1.GroupSync
	Provider          *redhatcopv1alpha1.GitHubProvider
	Client            *github.Client
	V4Client          *githubv4.Client
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
	err := g.ReconcilerBase.GetClient().Get(g.Context, types.NamespacedName{Name: g.Provider.CredentialsSecret.Name, Namespace: g.Provider.CredentialsSecret.Namespace}, credentialsSecret)

	if err != nil {
		validationErrors = append(validationErrors, err)
	} else {
		// Check that provided secret contains required keys
		_, tokenSecretFound := credentialsSecret.Data[secretTokenKey]
		_, privateKeyFound := credentialsSecret.Data[privateKey]
		_, integrationIdFound := credentialsSecret.Data[appId]

		if !tokenSecretFound && !(privateKeyFound && integrationIdFound) {
			validationErrors = append(validationErrors, fmt.Errorf("Could not find `token` or `privateKey` and `appId` key in secret '%s' in namespace '%s", g.Provider.CredentialsSecret.Name, g.Provider.CredentialsSecret.Namespace))
		}

		g.CredentialsSecret = credentialsSecret
	}

	if g.Provider.Organization == "" {
		validationErrors = append(validationErrors, fmt.Errorf("Organization name not provided"))
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
			validationErrors = append(validationErrors, fmt.Errorf("Could not find '%s' key in %s '%s' in namespace '%s", resourceCaKey, providerCaResource.Kind, providerCaResource.Name, providerCaResource.Namespace))
		}

		g.CaCertificate = caResource[resourceCaKey]
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
	privateKey, privateKeyFound := g.CredentialsSecret.Data[privateKey]
	appId, appIdFound := g.CredentialsSecret.Data[appId]

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

	config := githubapp.Config{
		V3APIURL: *g.Provider.URL,
		V4APIURL: *g.Provider.V4URL,
	}

	opts := []githubapp.ClientOption{
		githubapp.WithClientUserAgent(userAgent),
		githubapp.WithClientCaching(false, func() httpcache.Cache { return httpcache.NewMemoryCache() }),
	}
	if transport != nil {
		opts = append(opts, githubapp.WithTransport(transport))
	}

	if privateKeyFound && appIdFound {
		config.App.PrivateKey = string(privateKey)

		intId, err := strconv.ParseInt(string(appId), 10, 64)
		if err != nil {
			return err
		}
		config.App.IntegrationID = intId

		clientCreator, err := githubapp.NewDefaultCachingClientCreator(config, opts...)
		if err != nil {
			return err
		}

		appClient, err := clientCreator.NewAppClient()
		if err != nil {
			return err
		}

		installService := githubapp.NewInstallationsService(appClient)
		installation, err := installService.GetByOwner(g.Context, g.Provider.Organization)
		if err != nil {
			return err
		}

		ghClient, err = clientCreator.NewInstallationClient(installation.ID)
		if err != nil {
			return err
		}

		g.V4Client, err = clientCreator.NewInstallationV4Client(installation.ID)
		if err != nil {
			return err
		}

	} else if tokenSecretFound {
		clientCreator, err := githubapp.NewDefaultCachingClientCreator(config, opts...)
		if err != nil {
			return err
		}
		ghClient, err = clientCreator.NewTokenClient(string(tokenSecret))
		if err != nil {
			return err
		}

		g.V4Client, err = clientCreator.NewTokenV4Client(string(tokenSecret))
		if err != nil {
			return err
		}
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

	var scimUserIdMap map[string]string = nil
	if g.Provider.MapByScimId {
		scimUserIdMap, err = g.getScimIdentity()
		if err != nil {
			return nil, err
		}
	}

	for _, team := range teams {
		if !isGroupAllowed(*team.Name, g.Provider.Teams) {
			continue
		}

		ocpGroup := userv1.Group{
			TypeMeta: v1.TypeMeta{
				Kind:       "Group",
				APIVersion: userv1.GroupVersion.String(),
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
			var userId string
			if g.Provider.MapByScimId {
				userId = scimUserIdMap[*teamMember.Login]
			} else {
				userId = *teamMember.Login
			}

			ocpGroup.Users = append(ocpGroup.Users, userId)
		}

		ocpGroups = append(ocpGroups, ocpGroup)
	}

	return ocpGroups, nil
}

func (g *GitHubSyncer) getScimIdentity() (map[string]string, error) {
	const after = "after"
	// query vars for graphQl
	variables := map[string]interface{}{
		"organization": githubv4.String(g.Provider.Organization),
		"first":        githubv4.Int(pageSize),
		after:          (*githubv4.String)(nil),
	}

	userMap := make(map[string]string)
	for { // while
		err := g.V4Client.Query(g.Context, &scimQuery, variables)
		if err != nil {
			return nil, err
		}

		// map from loginId -> SCIM/SAML Id
		for _, v := range scimQuery.Organization.SamlIdentityProvider.ExternalIdentities.Edges {
			userMap[string(v.Node.User.Login)] = string(v.Node.SamlIdentity.Username)
		}
		if !scimQuery.Organization.SamlIdentityProvider.ExternalIdentities.PageInfo.HasNextPage {
			break
		}
		variables[after] = scimQuery.Organization.SamlIdentityProvider.ExternalIdentities.PageInfo.EndCursor
	}

	return userMap, nil
}

func (g *GitHubSyncer) getOrganizationTeams() ([]*github.Team, error) {
	opts := &github.ListOptions{PerPage: pageSize}
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
		ListOptions: github.ListOptions{PerPage: pageSize},
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
