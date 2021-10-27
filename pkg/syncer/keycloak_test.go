package syncer

import (
	"testing"

	"github.com/Nerzal/gocloak/v5"
	"github.com/stretchr/testify/require"
)

func Test_findParentGroup(t *testing.T) {
	subject := &gocloak.Group{Name: stringPtr("child"), ID: stringPtr("child")}
	parent := &gocloak.Group{Name: stringPtr("parent"), ID: stringPtr("parent"), SubGroups: []*gocloak.Group{subject}}
	groups := []*gocloak.Group{
		parent,
		{Name: stringPtr("not-a-parent"), ID: stringPtr("not-a-parent")},
		subject,
	}

	require.Equal(t, parent, findParentGroup(subject, groups))
}

func Test_findAllParentGroups(t *testing.T) {
	subject := &gocloak.Group{Name: stringPtr("child"), ID: stringPtr("child")}
	directParent := &gocloak.Group{Name: stringPtr("direct-parent"), ID: stringPtr("direct-parent"), SubGroups: []*gocloak.Group{subject}}
	topLevelParent := &gocloak.Group{Name: stringPtr("tl-parent"), ID: stringPtr("tl-parent"), SubGroups: []*gocloak.Group{directParent}}
	groups := []*gocloak.Group{
		directParent,
		topLevelParent,
		{Name: stringPtr("not-a-parent"), ID: stringPtr("not-a-parent")},
		subject,
	}

	require.Equal(t, []*gocloak.Group{topLevelParent, directParent}, findAllParentGroups(subject, groups))
}

func stringPtr(str string) *string {
	return &str
}
