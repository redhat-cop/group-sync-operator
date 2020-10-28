package builders

import (
	legacyconfigv1 "github.com/openshift/api/legacyconfig/v1"
	"github.com/openshift/library-go/pkg/security/ldapquery"
	"github.com/redhat-cop/group-sync-operator/pkg/provider/ldap/helpers/syncerror"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SyncBuilder describes an object that can build all the schema-specific parts of an LDAPGroupSyncer
type SyncBuilder interface {
	GetGroupLister() (syncerror.LDAPGroupLister, error)
	GetGroupNameMapper() (syncerror.LDAPGroupNameMapper, error)
	GetUserNameMapper() (syncerror.LDAPUserNameMapper, error)
	GetGroupMemberExtractor() (syncerror.LDAPMemberExtractor, error)
}

// PruneBuilder describes an object that can build all the schema-specific parts of an LDAPGroupPruner
type PruneBuilder interface {
	GetGroupLister() (syncerror.LDAPGroupLister, error)
	GetGroupNameMapper() (syncerror.LDAPGroupNameMapper, error)
	GetGroupDetector() (syncerror.LDAPGroupDetector, error)
}

// GroupNameRestrictions desribes an object that holds blacklists and whitelists
type GroupNameRestrictions interface {
	GetWhitelist() []string
	GetBlacklist() []string
}

// OpenShiftGroupNameRestrictions describes an object that holds blacklists and whitelists as well as
// a client that can retrieve OpenShift groups to satisfy those lists
type OpenShiftGroupNameRestrictions interface {
	GroupNameRestrictions
	GetClient() client.Client
}

// MappedNameRestrictions describes an object that holds user name mappings for a group sync job
type MappedNameRestrictions interface {
	GetGroupNameMappings() map[string]string
}

func ToLDAPQuery(in legacyconfigv1.LDAPQuery) ldapquery.SerializeableLDAPQuery {
	return ldapquery.SerializeableLDAPQuery{
		BaseDN:       in.BaseDN,
		Scope:        in.Scope,
		DerefAliases: in.DerefAliases,
		TimeLimit:    in.TimeLimit,
		Filter:       in.Filter,
		PageSize:     in.PageSize,
	}
}
