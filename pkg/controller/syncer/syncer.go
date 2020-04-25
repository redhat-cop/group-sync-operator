package syncer

import (
	"fmt"

	userv1 "github.com/openshift/api/user/v1"
	redhatcopv1alpha1 "github.com/redhat-cop/group-sync-operator/pkg/apis/redhatcop/v1alpha1"
	"github.com/redhat-cop/operator-utils/pkg/util"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
)

type GroupSyncer interface {
	GetProviderName() string
	Init() bool
	Bind() error
	Sync() ([]userv1.Group, error)
	Validate() error
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
	case provider.Keycloak != nil:
		{
			return &KeycloakSyncer{GroupSync: groupSync, Provider: provider.Keycloak, Name: provider.Name, ReconcilerBase: reconcilerBase}, nil
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

	for _, syncer := range m.GroupSyncers {
		err := syncer.Validate()

		if err != nil {
			syncersError = append(syncersError, err)
		}

	}

	return utilerrors.NewAggregate(syncersError)

}
