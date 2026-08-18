package controller

import (
	"testing"
)

// TestShouldSyncExistingGroup tests the decision of whether an existing Group
// may be written by the current reconcile, given its sync-provider label and
// the set of live GroupSync names.
func TestShouldSyncExistingGroup(t *testing.T) {
	tests := []struct {
		name               string
		foundLabel         string
		labelExists        bool
		providerLabel      string
		liveGroupSyncNames map[string]bool
		expected           bool
	}{
		{
			name:               "exact match - group owned by this reconcile",
			foundLabel:         "cluster-config_ldap",
			labelExists:        true,
			providerLabel:      "cluster-config_ldap",
			liveGroupSyncNames: map[string]bool{"cluster-config": true},
			expected:           true,
		},
		{
			name:               "mismatch and old owner absent - adopt orphan",
			foundLabel:         "old-name_ldap",
			labelExists:        true,
			providerLabel:      "new-name_ldap",
			liveGroupSyncNames: map[string]bool{"new-name": true},
			expected:           true,
		},
		{
			name:               "mismatch but old owner still live - contention, skip",
			foundLabel:         "other-groupsync_ldap",
			labelExists:        true,
			providerLabel:      "my-groupsync_ldap",
			liveGroupSyncNames: map[string]bool{"my-groupsync": true, "other-groupsync": true},
			expected:           false,
		},
		{
			name:               "two providers of the same live GroupSync - contention, skip",
			foundLabel:         "my-groupsync_azure",
			labelExists:        true,
			providerLabel:      "my-groupsync_ldap",
			liveGroupSyncNames: map[string]bool{"my-groupsync": true},
			expected:           false,
		},
		{
			name:               "label absent - unmanaged group, never adopt",
			foundLabel:         "",
			labelExists:        false,
			providerLabel:      "my-groupsync_ldap",
			liveGroupSyncNames: map[string]bool{"my-groupsync": true},
			expected:           false,
		},
		{
			name:               "label value has no underscore - unparseable, skip",
			foundLabel:         "not-a-provider-label",
			labelExists:        true,
			providerLabel:      "my-groupsync_ldap",
			liveGroupSyncNames: map[string]bool{"my-groupsync": true},
			expected:           false,
		},
		{
			name:               "provider name contains underscore, exact match",
			foundLabel:         "my-groupsync_ldap_primary",
			labelExists:        true,
			providerLabel:      "my-groupsync_ldap_primary",
			liveGroupSyncNames: map[string]bool{"my-groupsync": true},
			expected:           true,
		},
		{
			name:               "provider name contains underscore, old owner absent - adopt",
			foundLabel:         "old-name_ldap_primary",
			labelExists:        true,
			providerLabel:      "new-name_ldap_primary",
			liveGroupSyncNames: map[string]bool{"new-name": true},
			expected:           true,
		},
		{
			name:               "provider name contains underscore, old owner still live - skip",
			foundLabel:         "old-name_ldap_primary",
			labelExists:        true,
			providerLabel:      "new-name_ldap_primary",
			liveGroupSyncNames: map[string]bool{"new-name": true, "old-name": true},
			expected:           false,
		},
		{
			name:               "empty label value - skip",
			foundLabel:         "",
			labelExists:        true,
			providerLabel:      "my-groupsync_ldap",
			liveGroupSyncNames: map[string]bool{"my-groupsync": true},
			expected:           false,
		},
		{
			name:               "empty groupsync name in label value - skip",
			foundLabel:         "_ldap",
			labelExists:        true,
			providerLabel:      "my-groupsync_ldap",
			liveGroupSyncNames: map[string]bool{"my-groupsync": true},
			expected:           false,
		},
		{
			name:               "no live GroupSyncs at all, mismatched label - adopt",
			foundLabel:         "old-name_ldap",
			labelExists:        true,
			providerLabel:      "new-name_ldap",
			liveGroupSyncNames: map[string]bool{},
			expected:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldSyncExistingGroup(tt.foundLabel, tt.labelExists, tt.providerLabel, tt.liveGroupSyncNames)
			if result != tt.expected {
				t.Errorf("shouldSyncExistingGroup(%q, %v, %q, %v) = %v, want %v", tt.foundLabel, tt.labelExists, tt.providerLabel, tt.liveGroupSyncNames, result, tt.expected)
			}
		})
	}
}
