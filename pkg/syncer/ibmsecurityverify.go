package syncer

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	retryablehttp "github.com/hashicorp/go-retryablehttp"
	userv1 "github.com/openshift/api/user/v1"
	redhatcopv1alpha1 "github.com/redhat-cop/group-sync-operator/api/v1alpha1"
	"github.com/redhat-cop/group-sync-operator/pkg/constants"
	"github.com/redhat-cop/group-sync-operator/pkg/provider/ibmsecurityverify"
	"github.com/redhat-cop/operator-utils/pkg/util"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	isvLogger = logf.Log.WithName("syncer_ibmsecurityverify")
)

type IbmSecurityVerifySyncer struct {
	Name           string
	GroupSync      *redhatcopv1alpha1.GroupSync
	Provider       *redhatcopv1alpha1.IbmSecurityVerifyProvider
	Context        context.Context
	ReconcilerBase util.ReconcilerBase
	ApiClient      ibmsecurityverify.IbmSecurityVerifyClient
}

func (g *IbmSecurityVerifySyncer) Init() bool {
	g.Context = context.Background()
	return false
}

func (g *IbmSecurityVerifySyncer) Validate() error {
	validationErrors := []error{}
	credentialsSecret := &corev1.Secret{}
	err := g.ReconcilerBase.GetClient().Get(g.Context, types.NamespacedName{Name: g.Provider.CredentialsSecret.Name, Namespace: g.Provider.CredentialsSecret.Namespace}, credentialsSecret)
	if err != nil {
		validationErrors = append(validationErrors, err)
	} else {
		// Check that provided secret contains required keys
		_, clientIdFound := credentialsSecret.Data[secretClientIdKey]
		_, clientSecretFound := credentialsSecret.Data[secretClientSecretKey]

		if !clientIdFound && !clientSecretFound {
			validationErrors = append(validationErrors, fmt.Errorf("Could not find `clientId` and `clientSecret` secret '%s' in namespace '%s'", g.Provider.CredentialsSecret.Name, g.Provider.CredentialsSecret.Namespace))
		}

		g.ApiClient.SetCredentialsSecret(credentialsSecret)
	}

	if g.Provider.TenantURL == "" {
		validationErrors = append(validationErrors, fmt.Errorf("tenant URL not provided"))
	}

	if len(g.Provider.Groups) == 0 {
		validationErrors = append(validationErrors, fmt.Errorf("ISV groups not provided"))
	}

	return utilerrors.NewAggregate(validationErrors)
}

func (g *IbmSecurityVerifySyncer) Bind() error {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 10
	g.ApiClient.SetHttpClient(retryClient.StandardClient())
	return nil
}

func (g *IbmSecurityVerifySyncer) Sync() ([]userv1.Group, error) {
	ocpGroups := []userv1.Group{}
	for _, group := range g.Provider.Groups {
		isvGroup := g.ApiClient.GetGroup(g.Provider.TenantURL, group.Id)
		g.validateGroupName(isvGroup, group.Name)
		if isvGroup.Id != "" {
			ocpGroup := userv1.Group{
				TypeMeta: v1.TypeMeta{
					Kind:       "Group",
					APIVersion: userv1.GroupVersion.String(),
				},
				ObjectMeta: v1.ObjectMeta{
					Name:        g.normalizeName(isvGroup.DisplayName),
					Annotations: map[string]string{},
					Labels:      map[string]string{},
				},
				Users: []string{},
			}
			sourceUrl, _ := url.Parse(g.Provider.TenantURL)
			ocpGroup.GetAnnotations()[constants.SyncSourceHost] = sourceUrl.Host
			ocpGroup.GetAnnotations()[constants.SyncSourceUID] = isvGroup.Id
			for _, member := range isvGroup.Members {
				ocpGroup.Users = append(ocpGroup.Users, member.UserName)
			}
			ocpGroups = append(ocpGroups, ocpGroup)
		}
	}
	return ocpGroups, nil
}

func (g *IbmSecurityVerifySyncer) GetProviderName() string {
	return g.Name
}

func (g *IbmSecurityVerifySyncer) GetPrune() bool {
	return false
}

func (g *IbmSecurityVerifySyncer) normalizeName(name string) string {
	return strings.ReplaceAll(name, " ", "-")
}

func (g *IbmSecurityVerifySyncer) validateGroupName(group ibmsecurityverify.IsvGroup, expectedName string) {
	if expectedName != "" && (group.DisplayName != expectedName) {
		isvLogger.Error(nil, fmt.Sprintf("Retrieved group name '%s' does not match name '%s' in config", group.DisplayName, expectedName))
	}
}
