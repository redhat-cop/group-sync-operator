package builders

import (
	legacyconfigv1 "github.com/openshift/api/legacyconfig/v1"
	"github.com/openshift/library-go/pkg/security/ldapclient"
	ldapquery "github.com/openshift/library-go/pkg/security/ldapquery"
	syncgroups "github.com/redhat-cop/group-sync-operator/pkg/provider/ldap/helpers"
	"github.com/redhat-cop/group-sync-operator/pkg/provider/ldap/helpers/ad"
	"github.com/redhat-cop/group-sync-operator/pkg/provider/ldap/helpers/syncerror"
)

var _ SyncBuilder = &ADBuilder{}
var _ PruneBuilder = &ADBuilder{}

type ADBuilder struct {
	ClientConfig ldapclient.Config
	Config       *legacyconfigv1.ActiveDirectoryConfig

	adLDAPInterface *ad.ADLDAPInterface
}

func (b *ADBuilder) GetGroupLister() (syncerror.LDAPGroupLister, error) {
	return b.getADLDAPInterface()
}

func (b *ADBuilder) GetGroupNameMapper() (syncerror.LDAPGroupNameMapper, error) {
	return &syncgroups.DNLDAPGroupNameMapper{}, nil
}

func (b *ADBuilder) GetUserNameMapper() (syncerror.LDAPUserNameMapper, error) {
	return syncgroups.NewUserNameMapper(b.Config.UserNameAttributes), nil
}

func (b *ADBuilder) GetGroupMemberExtractor() (syncerror.LDAPMemberExtractor, error) {
	return b.getADLDAPInterface()
}

func (b *ADBuilder) getADLDAPInterface() (*ad.ADLDAPInterface, error) {
	if b.adLDAPInterface != nil {
		return b.adLDAPInterface, nil
	}

	userQuery, err := ldapquery.NewLDAPQuery(ToLDAPQuery(b.Config.AllUsersQuery))
	if err != nil {
		return nil, err
	}
	b.adLDAPInterface = ad.NewADLDAPInterface(b.ClientConfig,
		userQuery, b.Config.GroupMembershipAttributes, b.Config.UserNameAttributes)
	return b.adLDAPInterface, nil
}

func (b *ADBuilder) GetGroupDetector() (syncerror.LDAPGroupDetector, error) {
	return b.getADLDAPInterface()
}
