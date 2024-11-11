package syncer

import (
	"context"
	"fmt"
	"net/url"
	userv1 "github.com/openshift/api/user/v1"
	redhatcopv1alpha1 "github.com/redhat-cop/group-sync-operator/api/v1alpha1"
	"github.com/redhat-cop/group-sync-operator/pkg/provider/ibmsecurityverify"
	"github.com/redhat-cop/group-sync-operator/pkg/constants"
	"github.com/redhat-cop/operator-utils/pkg/util"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	retryablehttp "github.com/hashicorp/go-retryablehttp"
)

var (
	isvLogger   = logf.Log.WithName("syncer_ibmsecurityverify")
)

type IbmSecurityVerifySyncer struct {
	Name              string
	GroupSync         *redhatcopv1alpha1.GroupSync
	Provider          *redhatcopv1alpha1.IbmSecurityVerifyProvider
	Context           context.Context
	ReconcilerBase    util.ReconcilerBase
	ApiClient		  ibmsecurityverify.IbmSecurityVerifyClient
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
		validationErrors = append(validationErrors, fmt.Errorf("ISV group IDs not provided"))
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
	for _, groupId := range g.Provider.Groups {
		isvGroup := g.ApiClient.GetGroup(g.Provider.TenantURL, groupId)
		// TODO validate
		ocpGroup := userv1.Group{
			TypeMeta: v1.TypeMeta{
				Kind:       "Group",
				APIVersion: userv1.GroupVersion.String(),
			},
			ObjectMeta: v1.ObjectMeta{
				Name:        isvGroup.DisplayName,
				Annotations: map[string]string{},
				Labels:      map[string]string{},
			},
			Users: []string{},
		}

		sourceUrl, _ := url.Parse(g.Provider.TenantURL)
		ocpGroup.GetAnnotations()[constants.SyncSourceHost] = sourceUrl.Host
		ocpGroup.GetAnnotations()[constants.SyncSourceUID] = groupId

		for _, member := range isvGroup.Members {
			ocpGroup.Users = append(ocpGroup.Users, member.Id)
		}
		ocpGroups = append(ocpGroups, ocpGroup)
	}
	return ocpGroups, nil
}

func (g *IbmSecurityVerifySyncer) GetProviderName() string {
	return g.Name
}

func (g *IbmSecurityVerifySyncer) GetPrune() bool {
	return false
}
