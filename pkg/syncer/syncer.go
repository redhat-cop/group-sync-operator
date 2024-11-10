package syncer

import (
	"context"
	"fmt"

	userv1 "github.com/openshift/api/user/v1"
	redhatcopv1alpha1 "github.com/redhat-cop/group-sync-operator/api/v1alpha1"
	"github.com/redhat-cop/operator-utils/pkg/util"
	"github.com/robfig/cron/v3"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	secretUsernameKey    = "username"
	secretPasswordKey    = "password"
	secretTokenKey       = "token"
	secretTokenTypeKey   = "tokenType"
	privateKey           = "privateKey"
	appId                = "appId"
	defaultResourceCaKey = "ca.crt"
)

type GroupSyncer interface {
	GetProviderName() string
	Init() bool
	Bind() error
	Sync() ([]userv1.Group, error)
	Validate() error
	GetPrune() bool
}

type GroupSyncMgr struct {
	GroupSyncers []GroupSyncer
	GroupSync    *redhatcopv1alpha1.GroupSync
}

func GetGroupSyncMgr(groupSync *redhatcopv1alpha1.GroupSync, reconcilerBase util.ReconcilerBase) (GroupSyncMgr, error) {

	syncers := []GroupSyncer{}
	syncersError := []error{}

	for _, provider := range groupSync.Spec.Providers {

		syncer, err := getGroupSyncerForProvider(groupSync, &provider, reconcilerBase)

		if err != nil {
			syncersError = append(syncersError, err)
		}

		syncers = append(syncers, syncer)

	}

	return GroupSyncMgr{GroupSync: groupSync, GroupSyncers: syncers}, utilerrors.NewAggregate(syncersError)
}

func getGroupSyncerForProvider(groupSync *redhatcopv1alpha1.GroupSync, provider *redhatcopv1alpha1.Provider, reconcilerBase util.ReconcilerBase) (GroupSyncer, error) {

	switch {
	case provider.Okta != nil:
		{
			return &OktaSyncer{GroupSync: groupSync, Provider: provider.Okta, Name: provider.Name, ReconcilerBase: reconcilerBase}, nil
		}
	case provider.Keycloak != nil:
		{
			return &KeycloakSyncer{GroupSync: groupSync, Provider: provider.Keycloak, Name: provider.Name, ReconcilerBase: reconcilerBase}, nil
		}
	case provider.GitHub != nil:
		{
			return &GitHubSyncer{GroupSync: groupSync, Provider: provider.GitHub, Name: provider.Name, ReconcilerBase: reconcilerBase}, nil
		}
	case provider.GitLab != nil:
		{
			return &GitLabSyncer{GroupSync: groupSync, Provider: provider.GitLab, Name: provider.Name, ReconcilerBase: reconcilerBase}, nil
		}
	case provider.Azure != nil:
		{
			return &AzureSyncer{GroupSync: groupSync, Provider: provider.Azure, Name: provider.Name, ReconcilerBase: reconcilerBase}, nil
		}
	case provider.Ldap != nil:
		{
			return &LdapSyncer{GroupSync: groupSync, Provider: provider.Ldap, Name: provider.Name, ReconcilerBase: reconcilerBase}, nil
		}
	}
	return nil, fmt.Errorf("Could not find syncer for provider '%s'", provider.Name)
}

func (m *GroupSyncMgr) SetDefaults() bool {
	changed := false

	for _, syncer := range m.GroupSyncers {
		syncerChanged := syncer.Init()

		if syncerChanged == true {
			changed = true
		}

	}

	return changed

}

func (m *GroupSyncMgr) Validate() error {
	syncersError := []error{}

	// Validate Cron Schedule
	if m.GroupSync.Spec.Schedule != "" {
		if _, err := cron.ParseStandard(m.GroupSync.Spec.Schedule); err != nil {
			syncersError = append(syncersError, fmt.Errorf(fmt.Sprintf("Failed to validate cron schedule: %s", m.GroupSync.Spec.Schedule)))
		}
	}

	for _, syncer := range m.GroupSyncers {
		err := syncer.Validate()

		if err != nil {
			syncersError = append(syncersError, err)
		}

	}

	return utilerrors.NewAggregate(syncersError)

}

func isGroupAllowed(groupName string, allowedGroups []string) bool {
	if allowedGroups == nil || len(allowedGroups) == 0 {
		return true
	}

	for _, allowedGroup := range allowedGroups {
		if allowedGroup == groupName {
			return true
		}
	}

	return false
}

func getObjectRefData(context context.Context, client client.Client, resource *redhatcopv1alpha1.ObjectRef) (map[string][]byte, error) {

	if resource.Kind != "" && resource.Kind == redhatcopv1alpha1.ConfigMapObjectRefKind {
		caConfigMap := &corev1.ConfigMap{}
		err := client.Get(context, types.NamespacedName{Name: resource.Name, Namespace: resource.Namespace}, caConfigMap)

		if err != nil {
			return nil, err
		}

		configMapMap := map[string][]byte{}

		if caConfigMap.Data == nil {
			return nil, nil
		}

		for k, v := range caConfigMap.Data {
			configMapMap[k] = []byte(v)
		}

		return configMapMap, nil

	} else if resource.Kind != "" {
		caSecret := &corev1.Secret{}
		err := client.Get(context, types.NamespacedName{Name: resource.Name, Namespace: resource.Namespace}, caSecret)

		if err != nil {
			return nil, err
		}

		return caSecret.Data, nil

	}

	return nil, nil
}

func determineFromDeprecatedObjectRef(objectRef *redhatcopv1alpha1.ObjectRef, deprecatedObjectRef *redhatcopv1alpha1.ObjectRef) *redhatcopv1alpha1.ObjectRef {

	if objectRef != nil {
		return objectRef
	}

	return deprecatedObjectRef

}
