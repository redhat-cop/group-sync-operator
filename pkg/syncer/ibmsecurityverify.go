package syncer

import (
	"io"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/gregjones/httpcache"

	userv1 "github.com/openshift/api/user/v1"
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
	isvLogger   = logf.Log.WithName("syncer_ibmsecurityverify")
)

type IbmSecurityVerifySyncer struct {
	Name              string
	GroupSync         *redhatcopv1alpha1.GroupSync
	Provider          *redhatcopv1alpha1.IbmSecurityVerifyProvider
	Client            *http.Client
	Context           context.Context
	ReconcilerBase    util.ReconcilerBase
	CredentialsSecret *corev1.Secret
	URL               *url.URL
	CaCertificate     []byte
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

		g.CredentialsSecret = credentialsSecret
	}

	if g.Provider.TenantUrl == "" {
		validationErrors = append(validationErrors, fmt.Errorf("tenant URL not provided"))
	}

	return utilerrors.NewAggregate(validationErrors)
}

func (g *IbmSecurityVerifySyncer) Sync() ([]userv1.Group, error) {
	ocpGroups = append(ocpGroups, ocpGroup)

	for isvGroup := range g.Provider.Groups {
		url := fmt.Sprint("%s/v2.0/Groups/%s?membershipType=firstLevelUsersAndGroups", g.Provider.TenantUrl, isvGroup)
	}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("accept", "application/scim+json")
	res, _ := http.DefaultClient.Do(req)
	defer res.Body.Close()
	isvGroups, _ := io.ReadAll(res.Body)

	ocpGroups := []userv1.Group{}

	for isvGroup := range isvGroups {
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
	}
	

	return ocpGroups, nil
}

func (g *IbmSecurityVerifySyncer) GetProviderName() string {
	return g.Name
}

func (g *IbmSecurityVerifySyncer) GetPrune() bool {
	
}
