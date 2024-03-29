package syncgroups

import (
	"context"
	"fmt"
	"net"
	"time"

	"gopkg.in/ldap.v2"

	kapierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"

	"github.com/go-logr/logr"
	userv1 "github.com/openshift/api/user/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/openshift/library-go/pkg/security/ldapquery"
	"github.com/redhat-cop/group-sync-operator/pkg/provider/ldap/helpers/interfaces"
)

// GroupSyncer runs a Sync job on Groups
type GroupSyncer interface {
	// Sync syncs groups in OpenShift with records from an external source
	Sync() (groupsAffected []*userv1.Group, errors []error)
}

// LDAPGroupSyncer sync Groups with records on an external LDAP server
type LDAPGroupSyncer struct {
	// Lists all groups to be synced
	GroupLister interfaces.LDAPGroupLister
	// Fetches a group and extracts object metainformation and membership list from a group
	GroupMemberExtractor interfaces.LDAPMemberExtractor
	// Maps an LDAP user entry to an OpenShift User's Name
	UserNameMapper interfaces.LDAPUserNameMapper
	// Maps an LDAP group enrty to an OpenShift Group's Name
	GroupNameMapper interfaces.LDAPGroupNameMapper
	// Allows the Syncer to search for OpenShift Groups
	Client client.Client
	// Host stores the address:port of the LDAP server
	Host string
	// DryRun indicates that no changes should be made.
	DryRun bool

	Log logr.Logger
}

var _ GroupSyncer = &LDAPGroupSyncer{}

// Sync allows the LDAPGroupSyncer to be a GroupSyncer
func (s *LDAPGroupSyncer) Sync() ([]*userv1.Group, []error) {
	openshiftGroups := []*userv1.Group{}
	var errors []error

	// determine what to sync
	ldapGroupUIDs, err := s.GroupLister.ListGroups()
	if err != nil {
		errors = append(errors, err)
		return nil, errors
	}

	for _, ldapGroupUID := range ldapGroupUIDs {

		// get membership data
		memberEntries, err := s.GroupMemberExtractor.ExtractMembers(ldapGroupUID)
		if err != nil {
			s.Log.Error(err, "Error determining LDAP group membership for group", "LDAP Group UID", ldapGroupUID)
			errors = append(errors, err)
			continue
		}

		// determine OpenShift Users' usernames for LDAP group members
		usernames, err := s.determineUsernames(memberEntries)
		if err != nil {
			s.Log.Error(err, "Error determining usernames for LDAP group", "LDAP Group UID", ldapGroupUID)
			errors = append(errors, err)
			continue
		}

		// update the OpenShift Group corresponding to this record
		openshiftGroup, err := s.makeOpenShiftGroup(ldapGroupUID, usernames)
		if err != nil {
			if ldapquery.IsQueryOutOfBoundsError(err) {
				s.Log.Error(err, "LDAP Query is out of bounds")
				continue
			}
			s.Log.Error(err, "Error building OpenShift group for LDAP group", "LDAP Group UID", ldapGroupUID)
			errors = append(errors, err)
			continue
		}
		openshiftGroups = append(openshiftGroups, openshiftGroup)

		if !s.DryRun {
			s.Log.Info(fmt.Sprintf("group/%s", openshiftGroup.Name))
			if err := s.updateOpenShiftGroup(openshiftGroup); err != nil {
				s.Log.Error(err, "Error updating OpenShift group for LDAP group", "OpenShift group", openshiftGroup.Name, "LDAP group", ldapGroupUID)
				errors = append(errors, err)
				continue
			}
		}
	}

	return openshiftGroups, errors
}

// determineUsers determines the OpenShift Users that correspond to a list of LDAP member entries
func (s *LDAPGroupSyncer) determineUsernames(members []*ldap.Entry) ([]string, error) {
	var usernames []string
	for _, member := range members {
		username, err := s.UserNameMapper.UserNameFor(member)
		if err != nil {
			return nil, err
		}

		usernames = append(usernames, username)
	}
	return usernames, nil
}

// updateOpenShiftGroup creates the OpenShift Group in etcd
func (s *LDAPGroupSyncer) updateOpenShiftGroup(openshiftGroup *userv1.Group) error {
	if len(openshiftGroup.UID) > 0 {
		err := s.Client.Update(context.TODO(), openshiftGroup, &client.UpdateOptions{})
		return err
	}

	err := s.Client.Create(context.TODO(), openshiftGroup, &client.CreateOptions{})
	return err
}

// makeOpenShiftGroup creates the OpenShift Group object that needs to be updated, updates its data
func (s *LDAPGroupSyncer) makeOpenShiftGroup(ldapGroupUID string, usernames []string) (*userv1.Group, error) {
	hostIP, _, err := net.SplitHostPort(s.Host)
	if err != nil {
		return nil, err
	}
	groupName, err := s.GroupNameMapper.GroupNameFor(ldapGroupUID)
	if err != nil {
		return nil, err
	}

	group := &userv1.Group{}
	err = s.Client.Get(context.TODO(), types.NamespacedName{Name: groupName, Namespace: ""}, group)
	if kapierrors.IsNotFound(err) {
		group = &userv1.Group{}
		group.Name = groupName
		group.Annotations = map[string]string{
			LDAPURLAnnotation: s.Host,
			LDAPUIDAnnotation: ldapGroupUID,
		}
		group.Labels = map[string]string{
			LDAPHostLabel: hostIP,
		}

	} else if err != nil {
		return nil, err
	}

	// make sure we aren't taking over an OpenShift group that is already related to a different LDAP group
	if host, exists := group.Labels[LDAPHostLabel]; !exists || (host != hostIP) {
		return nil, fmt.Errorf("group %q: %s label did not match sync host: wanted %s, got %s",
			group.Name, LDAPHostLabel, hostIP, host)
	}
	if url, exists := group.Annotations[LDAPURLAnnotation]; !exists || (url != s.Host) {
		return nil, fmt.Errorf("group %q: %s annotation did not match sync host: wanted %s, got %s",
			group.Name, LDAPURLAnnotation, s.Host, url)
	}
	if uid, exists := group.Annotations[LDAPUIDAnnotation]; !exists || (uid != ldapGroupUID) {
		return nil, fmt.Errorf("group %q: %s annotation did not match LDAP UID: wanted %s, got %s",
			group.Name, LDAPUIDAnnotation, ldapGroupUID, uid)
	}

	// overwrite Group Users data
	group.Users = usernames
	group.Annotations[LDAPSyncTimeAnnotation] = time.Now().UTC().Format(time.RFC3339)

	return group, nil
}
