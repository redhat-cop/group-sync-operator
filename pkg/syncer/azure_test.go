package syncer

import (
	"testing"

	graph "github.com/microsoftgraph/msgraph-sdk-go/models"
	redhatcopv1alpha1 "github.com/redhat-cop/group-sync-operator/api/v1alpha1"
)

// Helper function to create a pointer to a string
func strPtr(s string) *string {
	return &s
}

// Helper function to create a pointer to a bool
func boolPtr(b bool) *bool {
	return &b
}

// TestCompileClientFilter tests the CEL filter compilation
func TestCompileClientFilter(t *testing.T) {
	tests := []struct {
		name        string
		filter      string
		expectError bool
	}{
		{
			name:        "valid filter - simple equality",
			filter:      "group.mailNickname == group.displayName",
			expectError: false,
		},
		{
			name:        "valid filter - boolean logic",
			filter:      "group.securityEnabled && !group.mailEnabled",
			expectError: false,
		},
		{
			name:        "valid filter - has operator",
			filter:      "has(group.description)",
			expectError: false,
		},
		{
			name:        "valid filter - complex expression",
			filter:      "group.mailNickname == group.displayName && group.securityEnabled && !has(group.description)",
			expectError: false,
		},
		{
			name:        "invalid filter - syntax error",
			filter:      "group.mailNickname ==",
			expectError: true,
		},
		{
			name:        "invalid filter - unknown field",
			filter:      "group.unknownField == 'value'",
			expectError: false, // compile can't catch this with map<string,dyn>; probeClientFilter() will
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			syncer := &AzureSyncer{
				Provider: &redhatcopv1alpha1.AzureProvider{
					ClientFilter: tt.filter,
				},
			}

			err := syncer.compileClientFilter()
			if (err != nil) != tt.expectError {
				t.Errorf("compileClientFilter() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

// TestProbeClientFilter tests that the probe catches unknown fields missed by compilation
func TestProbeClientFilter(t *testing.T) {
	tests := []struct {
		name        string
		filter      string
		expectError bool
	}{
		{
			name:        "valid filter - known field",
			filter:      "group.displayName == 'test'",
			expectError: false,
		},
		{
			name:        "invalid filter - unknown field caught by probe",
			filter:      "group.unknownField == 'value'",
			expectError: true,
		},
		{
			name:        "invalid filter - typo in field name",
			filter:      "group.securityEnable == true",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			syncer := &AzureSyncer{
				Provider: &redhatcopv1alpha1.AzureProvider{
					ClientFilter: tt.filter,
				},
			}
			if err := syncer.compileClientFilter(); err != nil {
				t.Fatalf("unexpected compile error: %v", err)
			}
			err := syncer.probeClientFilter()
			if (err != nil) != tt.expectError {
				t.Errorf("probeClientFilter() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

// TestEvaluateClientFilter tests the CEL filter evaluation
func TestEvaluateClientFilter(t *testing.T) {
	tests := []struct {
		name           string
		filter         string
		group          graph.Group
		expectedResult bool
		expectError    bool
	}{
		{
			name:   "matching mailNickname and displayName",
			filter: "group.mailNickname == group.displayName",
			group: func() graph.Group {
				g := graph.NewGroup()
				g.SetDisplayName(strPtr("test-group"))
				g.SetMailNickname(strPtr("test-group"))
				return *g
			}(),
			expectedResult: true,
			expectError:    false,
		},
		{
			name:   "non-matching mailNickname and displayName",
			filter: "group.mailNickname == group.displayName",
			group: func() graph.Group {
				g := graph.NewGroup()
				g.SetDisplayName(strPtr("test-group"))
				g.SetMailNickname(strPtr("different-nickname"))
				return *g
			}(),
			expectedResult: false,
			expectError:    false,
		},
		{
			name:   "security enabled group",
			filter: "group.securityEnabled",
			group: func() graph.Group {
				g := graph.NewGroup()
				g.SetDisplayName(strPtr("test-group"))
				g.SetSecurityEnabled(boolPtr(true))
				return *g
			}(),
			expectedResult: true,
			expectError:    false,
		},
		{
			name:   "complex filter - all conditions met",
			filter: "group.mailNickname == group.displayName && group.securityEnabled && !group.mailEnabled",
			group: func() graph.Group {
				g := graph.NewGroup()
				g.SetDisplayName(strPtr("test-group"))
				g.SetMailNickname(strPtr("test-group"))
				g.SetSecurityEnabled(boolPtr(true))
				g.SetMailEnabled(boolPtr(false))
				return *g
			}(),
			expectedResult: true,
			expectError:    false,
		},
		{
			name:   "complex filter - one condition not met",
			filter: "group.mailNickname == group.displayName && group.securityEnabled && !group.mailEnabled",
			group: func() graph.Group {
				g := graph.NewGroup()
				g.SetDisplayName(strPtr("test-group"))
				g.SetMailNickname(strPtr("different"))
				g.SetSecurityEnabled(boolPtr(true))
				g.SetMailEnabled(boolPtr(false))
				return *g
			}(),
			expectedResult: false,
			expectError:    false,
		},
		{
			name:   "has operator - description present",
			filter: "has(group.description) && group.description != ''",
			group: func() graph.Group {
				g := graph.NewGroup()
				g.SetDisplayName(strPtr("test-group"))
				g.SetDescription(strPtr("A test description"))
				return *g
			}(),
			expectedResult: true,
			expectError:    false,
		},
		{
			name:   "has operator - description absent",
			filter: "group.description == \"\"",
			group: func() graph.Group {
				g := graph.NewGroup()
				g.SetDisplayName(strPtr("test-group"))
				return *g
			}(),
			expectedResult: true,
			expectError:    false,
		},
		{
			name:   "no filter - should include all",
			filter: "",
			group: func() graph.Group {
				g := graph.NewGroup()
				g.SetDisplayName(strPtr("test-group"))
				return *g
			}(),
			expectedResult: true,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			syncer := &AzureSyncer{
				Provider: &redhatcopv1alpha1.AzureProvider{
					ClientFilter: tt.filter,
				},
			}

			if tt.filter != "" {
				err := syncer.compileClientFilter()
				if err != nil {
					t.Fatalf("compileClientFilter() failed: %v", err)
				}
			}

			result, err := syncer.evaluateClientFilter(tt.group)
			if (err != nil) != tt.expectError {
				t.Errorf("evaluateClientFilter() error = %v, expectError %v", err, tt.expectError)
				return
			}

			if result != tt.expectedResult {
				t.Errorf("evaluateClientFilter() = %v, want %v", result, tt.expectedResult)
			}
		})
	}
}

// TestExtractGroupFields tests the dynamic field extraction
func TestExtractGroupFields(t *testing.T) {
	g := graph.NewGroup()
	g.SetDisplayName(strPtr("test-group"))
	g.SetMailNickname(strPtr("test-nickname"))
	g.SetSecurityEnabled(boolPtr(true))
	g.SetMailEnabled(boolPtr(false))
	g.SetDescription(strPtr("Test description"))
	id := "12345-67890"
	g.SetId(&id)

	fields := extractGroupFields(*g)

	// Check that expected fields are present
	expectedFields := map[string]interface{}{
		"displayName":     "test-group",
		"mailNickname":    "test-nickname",
		"securityEnabled": true,
		"mailEnabled":     false,
		"description":     "Test description",
		"id":              id,
	}

	for key, expectedValue := range expectedFields {
		actualValue, found := fields[key]
		if !found {
			t.Errorf("Expected field %s not found in extracted fields", key)
			continue
		}
		if actualValue != expectedValue {
			t.Errorf("Field %s = %v, want %v", key, actualValue, expectedValue)
		}
	}

	// Verify that fields are lowercase camelCase
	if _, found := fields["DisplayName"]; found {
		t.Errorf("Field should be displayName, not DisplayName")
	}

	// Verify that Get prefix is removed
	if _, found := fields["getDisplayName"]; found {
		t.Errorf("Field should be displayName, not getDisplayName")
	}
}

// TestExtractGroupFieldsWithNilValues tests that nil values are handled correctly
func TestExtractGroupFieldsWithNilValues(t *testing.T) {
	g := graph.NewGroup()
	g.SetDisplayName(strPtr("test-group"))
	// mailNickname intentionally not set (nil)

	fields := extractGroupFields(*g)

	// displayName should be present
	if displayName, found := fields["displayName"]; !found {
		t.Errorf("displayName should be present")
	} else if displayName != "test-group" {
		t.Errorf("displayName = %v, want test-group", displayName)
	}

	// mailNickname should be present with zero value (empty string, not nil)
	if mailNickname, found := fields["mailNickname"]; !found {
		t.Errorf("mailNickname should be present in fields map even when not set")
	} else if mailNickname != "" {
		t.Errorf("mailNickname should be empty string when not set, got %v", mailNickname)
	}
}

// TestAzureBaseGroupFilter verifies that a baseGroups entry is resolved by
// object ID when it is a GUID and by display name otherwise. Combining both
// into a single OData filter is not possible because "id eq '<non-guid>'" is
// rejected by Microsoft Graph.
func TestAzureBaseGroupFilter(t *testing.T) {
	tests := []struct {
		name      string
		baseGroup string
		expected  string
	}{
		{
			name:      "object ID (GUID) is matched by id",
			baseGroup: "b60f04c2-b6f3-4086-8570-059e7aa446bd",
			expected:  "id eq 'b60f04c2-b6f3-4086-8570-059e7aa446bd'",
		},
		{
			name:      "display name is matched by displayName",
			baseGroup: "my-group",
			expected:  "displayName eq 'my-group'",
		},
		{
			name:      "non-ASCII display name is matched by displayName",
			baseGroup: "グループ002",
			expected:  "displayName eq 'グループ002'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := azureBaseGroupFilter(tt.baseGroup); got != tt.expected {
				t.Errorf("azureBaseGroupFilter(%q) = %q, want %q", tt.baseGroup, got, tt.expected)
			}
		})
	}
}

// TestAzureGroupMatches verifies that the groups filter matches on either the
// display name or the object ID, and that an empty filter matches everything.
func TestAzureGroupMatches(t *testing.T) {
	newGroup := func(id, displayName string) graph.Groupable {
		g := graph.NewGroup()
		if id != "" {
			g.SetId(strPtr(id))
		}
		if displayName != "" {
			g.SetDisplayName(strPtr(displayName))
		}
		return g
	}
	guid := "b60f04c2-b6f3-4086-8570-059e7aa446bd"

	tests := []struct {
		name    string
		group   graph.Groupable
		allowed []string
		want    bool
	}{
		{"empty filter allows any group", newGroup(guid, "team"), nil, true},
		{"match by display name", newGroup(guid, "team"), []string{"team"}, true},
		{"match by object ID", newGroup(guid, "team"), []string{guid}, true},
		{"no match", newGroup(guid, "team"), []string{"other"}, false},
		{"match by ID when display name differs", newGroup(guid, "renamed"), []string{guid}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := azureGroupMatches(tt.group, tt.allowed); got != tt.want {
				t.Errorf("azureGroupMatches() = %v, want %v", got, tt.want)
			}
		})
	}
}
