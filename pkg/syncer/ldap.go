package syncer

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"strings"

	legacyconfigv1 "github.com/openshift/api/legacyconfig/v1"
	userv1 "github.com/openshift/api/user/v1"
	"github.com/openshift/library-go/pkg/security/ldapclient"
	"github.com/openshift/library-go/pkg/security/ldaputil"
	redhatcopv1alpha1 "github.com/redhat-cop/group-sync-operator/api/v1alpha1"
	"github.com/redhat-cop/group-sync-operator/pkg/provider/ldap/builders"
	ldapbuilders "github.com/redhat-cop/group-sync-operator/pkg/provider/ldap/builders"
	syncgroups "github.com/redhat-cop/group-sync-operator/pkg/provider/ldap/helpers"
	"github.com/redhat-cop/group-sync-operator/pkg/provider/ldap/helpers/interfaces"
	syncerror "github.com/redhat-cop/group-sync-operator/pkg/provider/ldap/helpers/syncerror"
	"github.com/redhat-cop/operator-utils/pkg/util"
	"gopkg.in/ldap.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/validation/field"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	ldapLogger = logf.Log.WithName("syncer_ldap")
)

type LdapSyncer struct {
	Name              string
	GroupSync         *redhatcopv1alpha1.GroupSync
	Provider          *redhatcopv1alpha1.LdapProvider
	ReconcilerBase    util.ReconcilerBase
	CredentialsSecret *corev1.Secret
	URL               *url.URL
	CaCertificate     []byte
	CaCertificateFile string
	Whitelist         []string
	Blacklist         []string
	Syncer            *syncgroups.LDAPGroupSyncer
}

func (l *LdapSyncer) Init() bool {

	if l.Provider.Whitelist == nil {
		l.Whitelist = []string{}
	} else {
		l.Whitelist = *l.Provider.Whitelist
	}

	if l.Provider.Blacklist == nil {
		l.Blacklist = []string{}
	} else {
		l.Blacklist = *l.Provider.Blacklist
	}

	return false
}

func (l *LdapSyncer) Validate() error {
	validationErrors := []error{}

	if l.Provider.CredentialsSecret != nil {
		credentialsSecret := &corev1.Secret{}
		err := l.ReconcilerBase.GetClient().Get(context.TODO(), types.NamespacedName{Name: l.Provider.CredentialsSecret.Name, Namespace: l.Provider.CredentialsSecret.Namespace}, credentialsSecret)

		if err != nil {
			validationErrors = append(validationErrors, err)
		} else {
			l.CredentialsSecret = credentialsSecret
		}

	}

	if l.Provider.CaSecret != nil {
		caSecret := &corev1.Secret{}
		err := l.ReconcilerBase.GetClient().Get(context.TODO(), types.NamespacedName{Name: l.Provider.CaSecret.Name, Namespace: l.Provider.CaSecret.Namespace}, caSecret)

		if err != nil {
			validationErrors = append(validationErrors, err)
		}

		var secretCaKey string
		if l.Provider.CaSecret.Key != "" {
			secretCaKey = l.Provider.CaSecret.Key
		} else {
			secretCaKey = defaultResourceCaKey
		}

		// Certificate key validation
		if _, found := caSecret.Data[secretCaKey]; !found {
			validationErrors = append(validationErrors, fmt.Errorf("Could not find '%s' key in secret '%s' in namespace '%s", secretCaKey, l.Provider.CaSecret.Name, l.Provider.CaSecret.Namespace))
		}

		l.CaCertificate = caSecret.Data[secretCaKey]

	}

	if l.Provider.URL == nil {
		validationErrors = append(validationErrors, fmt.Errorf("LDAP URL must be provided"))
	} else {

		var err error

		l.URL, err = url.Parse(*l.Provider.URL)

		if err != nil {
			validationErrors = append(validationErrors, fmt.Errorf("Invalid LDAP URL: '%s", *l.Provider.URL))
		}

		if l.Provider.Insecure {

			if l.URL.Scheme == string(ldaputil.SchemeLDAPS) {
				validationErrors = append(validationErrors, fmt.Errorf("Cannot use %s scheme with insecure=true", l.URL.Scheme))
			}

			if l.CaCertificate != nil {
				validationErrors = append(validationErrors, fmt.Errorf("Cannot specify a caSecret with insecure=true"))
			}

		} else {
			if l.CaCertificate == nil {
				validationErrors = append(validationErrors, fmt.Errorf("caSecret must be specified when insecure=false"))
			}
		}

	}

	for ldapGroupUID, openShiftGroupName := range l.Provider.LDAPGroupUIDToOpenShiftGroupNameMapping {
		if len(ldapGroupUID) == 0 || len(openShiftGroupName) == 0 {
			validationErrors = append(validationErrors, field.Invalid(field.NewPath("groupUIDNameMapping").Key(ldapGroupUID), openShiftGroupName, "has empty key or value"))
		}
	}

	schemaConfigsFound := []string{}

	if l.Provider.RFC2307Config != nil {
		validationErrors = append(validationErrors, ValidateRFC2307Config(l.Provider.RFC2307Config)...)
		schemaConfigsFound = append(schemaConfigsFound, "rfc2307")
	}
	if l.Provider.ActiveDirectoryConfig != nil {
		validationErrors = append(validationErrors, ValidateActiveDirectoryConfig(l.Provider.ActiveDirectoryConfig)...)
		schemaConfigsFound = append(schemaConfigsFound, "activeDirectory")
	}
	if l.Provider.AugmentedActiveDirectoryConfig != nil {
		validationErrors = append(validationErrors, ValidateAugmentedActiveDirectoryConfig(l.Provider.AugmentedActiveDirectoryConfig)...)
		schemaConfigsFound = append(schemaConfigsFound, "augmentedActiveDirectory")
	}

	if len(schemaConfigsFound) > 1 {
		validationErrors = append(validationErrors, fmt.Errorf("only one schema-specific config is allowed; found %v", schemaConfigsFound))
	}
	if len(schemaConfigsFound) == 0 {
		validationErrors = append(validationErrors, field.Required(field.NewPath("schema"), fmt.Sprintf("exactly one schema-specific config is required;  one of %v", []string{"rfc2307", "activeDirectory", "augmentedActiveDirectory"})))
	}

	return utilerrors.NewAggregate(validationErrors)
}

func (l *LdapSyncer) Bind() error {

	// Create a temporary file
	if len(l.CaCertificate) > 0 {

		file, err := ioutil.TempFile(os.TempDir(), fmt.Sprintf("group-sync-operator_ldap_%s.crt", l.Name))
		if err != nil {
			return err
		}

		if err := ioutil.WriteFile(file.Name(), l.CaCertificate, 0600); err != nil {
			return err
		}

		l.CaCertificateFile = file.Name()

		defer os.Remove(file.Name())

	}

	userName := l.getLdapCredentialValue(secretUsernameKey)
	password := l.getLdapCredentialValue(secretPasswordKey)

	clientConfig, err := ldapclient.NewLDAPClientConfig(l.URL.String(), userName, password, l.CaCertificateFile, l.Provider.Insecure)
	if err != nil {
		return fmt.Errorf("could not determine LDAP client configuration: %v", err)
	}

	errorHandler := l.CreateErrorHandler()

	syncBuilder, err := buildSyncBuilder(clientConfig, l.Provider, errorHandler)
	if err != nil {
		return err
	}

	// populate schema-independent syncer fields
	syncer := &syncgroups.LDAPGroupSyncer{
		Host:   clientConfig.Host(),
		Client: l.ReconcilerBase.GetClient(),
		DryRun: true,
		Log:    ldapLogger,
	}

	syncer.GroupLister, err = getLDAPGroupLister(syncBuilder, l)
	if err != nil {
		return err
	}
	syncer.GroupNameMapper, err = getGroupNameMapper(syncBuilder, l)
	if err != nil {
		return err
	}

	syncer.GroupMemberExtractor, err = syncBuilder.GetGroupMemberExtractor()
	if err != nil {
		return err
	}

	syncer.UserNameMapper, err = syncBuilder.GetUserNameMapper()
	if err != nil {
		return err
	}

	l.Syncer = syncer

	return nil
}

func (l *LdapSyncer) Sync() ([]userv1.Group, error) {

	// Now we run the Syncer and report any errors
	ocpGroups := []userv1.Group{}

	openshiftGroups, syncErrors := l.Syncer.Sync()

	if len(syncErrors) == 0 {
		for _, group := range openshiftGroups {
			ocpGroups = append(ocpGroups, *group)
		}
	}

	return ocpGroups, utilerrors.NewAggregate(syncErrors)
}

func (l *LdapSyncer) GetProviderName() string {
	return l.Name
}

func ValidateRFC2307Config(config *legacyconfigv1.RFC2307Config) []error {
	validationErrors := []error{}

	validationErrors = append(validationErrors, ValidateLDAPQuery(config.AllGroupsQuery, field.NewPath("groupsQuery"))...)
	if len(config.GroupUIDAttribute) == 0 {
		validationErrors = append(validationErrors, field.Required(field.NewPath("groupUIDAttribute"), ""))
	}
	if len(config.GroupNameAttributes) == 0 {
		validationErrors = append(validationErrors, field.Required(field.NewPath("groupNameAttributes"), ""))
	}
	if len(config.GroupMembershipAttributes) == 0 {
		validationErrors = append(validationErrors, field.Required(field.NewPath("groupMembershipAttributes"), ""))
	}

	isUserDNQuery := strings.TrimSpace(strings.ToLower(config.UserUIDAttribute)) == "dn"
	validationErrors = append(validationErrors, validateLDAPQuery(config.AllUsersQuery, field.NewPath("usersQuery"), isUserDNQuery)...)
	if len(config.UserUIDAttribute) == 0 {
		validationErrors = append(validationErrors, field.Required(field.NewPath("userUIDAttribute"), ""))
	}
	if len(config.UserNameAttributes) == 0 {
		validationErrors = append(validationErrors, field.Required(field.NewPath("userNameAttributes"), ""))
	}

	return validationErrors
}

func ValidateActiveDirectoryConfig(config *legacyconfigv1.ActiveDirectoryConfig) []error {
	validationErrors := []error{}

	validationErrors = append(validationErrors, ValidateLDAPQuery(config.AllUsersQuery, field.NewPath("usersQuery"))...)
	if len(config.UserNameAttributes) == 0 {
		validationErrors = append(validationErrors, field.Required(field.NewPath("userNameAttributes"), ""))
	}
	if len(config.GroupMembershipAttributes) == 0 {
		validationErrors = append(validationErrors, field.Required(field.NewPath("groupMembershipAttributes"), ""))
	}

	return validationErrors
}

func ValidateAugmentedActiveDirectoryConfig(config *legacyconfigv1.AugmentedActiveDirectoryConfig) []error {
	validationErrors := []error{}

	validationErrors = append(validationErrors, ValidateLDAPQuery(config.AllUsersQuery, field.NewPath("usersQuery"))...)
	if len(config.UserNameAttributes) == 0 {
		validationErrors = append(validationErrors, field.Required(field.NewPath("userNameAttributes"), ""))
	}
	if len(config.GroupMembershipAttributes) == 0 {
		validationErrors = append(validationErrors, field.Required(field.NewPath("groupMembershipAttributes"), ""))
	}

	isGroupDNQuery := strings.TrimSpace(strings.ToLower(config.GroupUIDAttribute)) == "dn"
	validationErrors = append(validationErrors, validateLDAPQuery(config.AllGroupsQuery, field.NewPath("groupsQuery"), isGroupDNQuery)...)
	if len(config.GroupUIDAttribute) == 0 {
		validationErrors = append(validationErrors, field.Required(field.NewPath("groupUIDAttribute"), ""))
	}
	if len(config.GroupNameAttributes) == 0 {
		validationErrors = append(validationErrors, field.Required(field.NewPath("groupNameAttributes"), ""))
	}

	return validationErrors
}

func ValidateLDAPQuery(query legacyconfigv1.LDAPQuery, fldPath *field.Path) []error {
	return validateLDAPQuery(query, fldPath, false)
}
func validateLDAPQuery(query legacyconfigv1.LDAPQuery, fldPath *field.Path, isDNOnly bool) []error {
	validationErrors := []error{}

	if _, err := ldap.ParseDN(query.BaseDN); err != nil {
		validationErrors = append(validationErrors, field.Invalid(fldPath.Child("baseDN"), query.BaseDN,
			fmt.Sprintf("invalid base DN for search: %v", err)))
	}

	if len(query.Scope) > 0 {
		if _, err := ldaputil.DetermineLDAPScope(query.Scope); err != nil {
			validationErrors = append(validationErrors, field.Invalid(fldPath.Child("scope"), query.Scope,
				"invalid LDAP search scope"))
		}
	}

	if len(query.DerefAliases) > 0 {
		if _, err := ldaputil.DetermineDerefAliasesBehavior(query.DerefAliases); err != nil {
			validationErrors = append(validationErrors, field.Invalid(fldPath.Child("derefAliases"),
				query.DerefAliases, "LDAP alias dereferencing instruction invalid"))
		}
	}

	if query.TimeLimit < 0 {
		validationErrors = append(validationErrors, field.Invalid(fldPath.Child("timeout"), query.TimeLimit,
			"timeout must be equal to or greater than zero"))
	}

	if isDNOnly {
		if len(query.Filter) != 0 {
			validationErrors = append(validationErrors, field.Invalid(fldPath.Child("filter"), query.Filter, `cannot specify a filter when using "dn" as the UID attribute`))
		}
		return validationErrors
	}

	if _, err := ldap.CompileFilter(query.Filter); err != nil {
		validationErrors = append(validationErrors, field.Invalid(fldPath.Child("filter"), query.Filter,
			fmt.Sprintf("invalid query filter: %v", err)))
	}

	return validationErrors
}

// CreateErrorHandler creates an error handler for the LDAP sync job
func (l *LdapSyncer) CreateErrorHandler() syncerror.Handler {
	components := []syncerror.Handler{}
	if l.Provider.RFC2307Config != nil {
		if l.Provider.RFC2307Config.TolerateMemberOutOfScopeErrors {
			components = append(components, syncerror.NewMemberLookupOutOfBoundsSuppressor(ldapLogger))
		}
		if l.Provider.RFC2307Config.TolerateMemberNotFoundErrors {
			components = append(components, syncerror.NewMemberLookupMemberNotFoundSuppressor(ldapLogger))
		}
	}

	return syncerror.NewCompoundHandler(components...)
}

func buildSyncBuilder(clientConfig ldapclient.Config, provider *redhatcopv1alpha1.LdapProvider, errorHandler syncerror.Handler) (ldapbuilders.SyncBuilder, error) {
	switch {
	case provider.RFC2307Config != nil:
		return &ldapbuilders.RFC2307Builder{ClientConfig: clientConfig, Config: provider.RFC2307Config, ErrorHandler: errorHandler}, nil
	case provider.ActiveDirectoryConfig != nil:
		return &ldapbuilders.ADBuilder{ClientConfig: clientConfig, Config: provider.ActiveDirectoryConfig}, nil
	case provider.AugmentedActiveDirectoryConfig != nil:
		return &ldapbuilders.AugmentedADBuilder{ClientConfig: clientConfig, Config: provider.AugmentedActiveDirectoryConfig}, nil
	default:
		return nil, errors.New("invalid sync config type")
	}
}

func getLDAPGroupLister(syncBuilder ldapbuilders.SyncBuilder, info builders.GroupNameRestrictions) (interfaces.LDAPGroupLister, error) {
	if len(info.GetWhitelist()) != 0 {
		ldapWhitelist := syncgroups.NewLDAPWhitelistGroupLister(info.GetWhitelist())
		if len(info.GetBlacklist()) == 0 {
			return ldapWhitelist, nil
		}
		return syncgroups.NewLDAPBlacklistGroupLister(info.GetBlacklist(), ldapWhitelist), nil
	}

	syncLister, err := syncBuilder.GetGroupLister()
	if err != nil {
		return nil, err
	}
	if len(info.GetBlacklist()) == 0 {
		return syncLister, nil
	}

	return syncgroups.NewLDAPBlacklistGroupLister(info.GetBlacklist(), syncLister), nil
}

func getGroupNameMapper(syncBuilder builders.SyncBuilder, info builders.MappedNameRestrictions) (interfaces.LDAPGroupNameMapper, error) {
	syncNameMapper, err := syncBuilder.GetGroupNameMapper()
	if err != nil {
		return nil, err
	}

	// if the mapping is specified, union the specified mapping with the default mapping.  The specified mapping is checked first
	if len(info.GetGroupNameMappings()) > 0 {
		userDefinedMapper := syncgroups.NewUserDefinedGroupNameMapper(info.GetGroupNameMappings())
		if syncNameMapper == nil {
			return userDefinedMapper, nil
		}
		return &syncgroups.UnionGroupNameMapper{GroupNameMappers: []interfaces.LDAPGroupNameMapper{userDefinedMapper, syncNameMapper}}, nil
	}
	return syncNameMapper, nil
}

func (l *LdapSyncer) GetWhitelist() []string {
	return l.Whitelist
}

func (l *LdapSyncer) GetBlacklist() []string {
	return l.Blacklist
}

func (l *LdapSyncer) GetGroupNameMappings() map[string]string {
	return l.Provider.LDAPGroupUIDToOpenShiftGroupNameMapping
}

func (l *LdapSyncer) getLdapCredentialValue(key string) string {

	if l.Provider.CredentialsSecret != nil {
		if value, ok := l.CredentialsSecret.Data[key]; ok {
			return string(value)
		}
	}

	return ""
}

func (l *LdapSyncer) GetPrune() bool {
	return false
}
