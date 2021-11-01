package syncer

import (
	"strings"

	"github.com/Nerzal/gocloak/v5"
	userv1 "github.com/openshift/api/user/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"

	redhatcopv1alpha1 "github.com/redhat-cop/group-sync-operator/api/v1alpha1"
	"github.com/redhat-cop/group-sync-operator/pkg/constants"
)

type KeycloakGroupMapper struct {
	GetGroupMembers func(groupID string) ([]*gocloak.User, error)

	AllowedGroups         []string
	Scope                 redhatcopv1alpha1.SyncScope
	SubGroupProcessing    redhatcopv1alpha1.SubGroupProcessing
	SubGroupJoinSeparator string

	cachedGroups       map[string]*gocloak.Group
	cachedGroupMembers map[string][]*gocloak.User
}

func (k *KeycloakGroupMapper) Map(groups []*gocloak.Group) ([]userv1.Group, error) {
	k.cachedGroups = make(map[string]*gocloak.Group)
	k.cachedGroupMembers = make(map[string][]*gocloak.User)

	for _, group := range groups {

		if _, groupFound := k.cachedGroups[*group.ID]; !groupFound {
			k.processGroupsAndMembers(group, nil, k.Scope, k.SubGroupProcessing, k.SubGroupJoinSeparator)
		}
	}

	ocpGroups := []userv1.Group{}

	for _, cachedGroup := range k.cachedGroups {

		groupAttributes := map[string]string{}

		for key, value := range cachedGroup.Attributes {
			// we add the annotation that qualify for OCP annotations and log for the ones that don't
			if errs := validation.IsQualifiedName(key); len(errs) == 0 {
				groupAttributes[key] = strings.Join(value, "'")
			} else {
				keycloakLogger.Info("unable to add annotation to", "group", cachedGroup.Name, "key", key, "value", value)
			}
		}

		ocpGroup := userv1.Group{
			TypeMeta: v1.TypeMeta{
				Kind:       "Group",
				APIVersion: userv1.GroupVersion.String(),
			},
			ObjectMeta: v1.ObjectMeta{
				Name:        *cachedGroup.Name,
				Annotations: groupAttributes,
				Labels:      map[string]string{},
			},
			Users: []string{},
		}

		childrenGroups := []string{}

		for _, subgroup := range cachedGroup.SubGroups {
			childrenGroups = append(childrenGroups, *subgroup.Name)
		}

		parentGroups := []string{}

		for _, group := range k.cachedGroups {
			for _, subgroup := range group.SubGroups {
				if *subgroup.Name == *cachedGroup.Name {
					parentGroups = append(parentGroups, *group.Name)
				}
			}
		}

		// Set Host Specific Details
		ocpGroup.GetAnnotations()[constants.SyncSourceUID] = *cachedGroup.ID
		if len(childrenGroups) > 0 {
			ocpGroup.GetAnnotations()[constants.HierarchyChildren] = strings.Join(childrenGroups, ",")
		}
		if len(parentGroups) == 1 {
			ocpGroup.GetAnnotations()[constants.HierarchyParent] = parentGroups[0]
		}
		if len(parentGroups) > 1 {
			ocpGroup.GetAnnotations()[constants.HierarchyParents] = strings.Join(parentGroups, ",")
		}

		for _, user := range k.cachedGroupMembers[*cachedGroup.ID] {
			ocpGroup.Users = append(ocpGroup.Users, *user.Username)
		}

		ocpGroups = append(ocpGroups, ocpGroup)

	}

	return ocpGroups, nil
}

func (k *KeycloakGroupMapper) processGroupsAndMembers(group, parentGroup *gocloak.Group, scope redhatcopv1alpha1.SyncScope, subGroupProcessing redhatcopv1alpha1.SubGroupProcessing, subGroupJoinSeparator string) error {

	if parentGroup == nil && !isGroupAllowed(*group.Name, k.AllowedGroups) {
		return nil
	}

	if redhatcopv1alpha1.JoinSubGroupProcessing == subGroupProcessing &&
		subGroupJoinSeparator != "" &&
		strings.Contains(*group.Name, subGroupJoinSeparator) {
		keycloakLogger.Error(
			errGroupNameContainsSeparator,
			"error processing group",
			"group", *group.Name,
			"separator", subGroupJoinSeparator,
		)
		return errGroupNameContainsSeparator
	}

	if parentGroup != nil && redhatcopv1alpha1.JoinSubGroupProcessing == k.SubGroupProcessing {
		name := *parentGroup.Name + subGroupJoinSeparator + *group.Name
		group.Name = &name
	}

	k.cachedGroups[*group.ID] = group

	groupMembers, err := k.GetGroupMembers(*group.ID)

	if err != nil {
		return err
	}

	k.cachedGroupMembers[*group.ID] = groupMembers

	// Add Group Members to Primary Group
	if parentGroup != nil && redhatcopv1alpha1.JoinSubGroupProcessing != subGroupProcessing {
		usersToAdd, _ := k.diff(groupMembers, k.cachedGroupMembers[*parentGroup.ID])
		k.cachedGroupMembers[*parentGroup.ID] = append(k.cachedGroupMembers[*parentGroup.ID], usersToAdd...)
	}

	// Process Subgroups
	if redhatcopv1alpha1.SubSyncScope == scope {
		for _, subGroup := range group.SubGroups {
			if _, subGroupFound := k.cachedGroups[*subGroup.ID]; !subGroupFound {
				k.processGroupsAndMembers(subGroup, group, scope, subGroupProcessing, subGroupJoinSeparator)
			}
		}
	}

	return nil
}

func (k *KeycloakGroupMapper) diff(lhsSlice, rhsSlice []*gocloak.User) (lhsOnly []*gocloak.User, rhsOnly []*gocloak.User) {
	return k.singleDiff(lhsSlice, rhsSlice), k.singleDiff(rhsSlice, lhsSlice)
}

func (k *KeycloakGroupMapper) singleDiff(lhsSlice, rhsSlice []*gocloak.User) (lhsOnly []*gocloak.User) {
	for _, lhs := range lhsSlice {
		found := false
		for _, rhs := range rhsSlice {
			if *lhs.ID == *rhs.ID {
				found = true
				break
			}
		}

		if !found {
			lhsOnly = append(lhsOnly, lhs)
		}
	}

	return lhsOnly
}
