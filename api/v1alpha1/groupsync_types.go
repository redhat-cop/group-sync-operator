/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	legacyconfigv1 "github.com/openshift/api/legacyconfig/v1"
	"github.com/operator-framework/operator-lib/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SyncScope string

const (
	OneSyncScope SyncScope = "one"
	SubSyncScope SyncScope = "sub"
)

// GroupSyncSpec defines the desired state of GroupSync
type GroupSyncSpec struct {

	// List of Providers that can be mounted by containers belonging to the pod.
	// +patchMergeKey=name
	// +patchStrategy=merge,retainKeys
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Providers"
	Providers []Provider `json:"providers,omitempty" patchStrategy:"merge,retainKeys" patchMergeKey:"name" protobuf:"bytes,1,rep,name=providers"`

	// Schedule represents a cron based configuration for synchronization
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Schedule",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Optional
	Schedule string `json:"schedule,omitempty"`
}

// GroupSyncStatus defines the observed state of GroupSync
type GroupSyncStatus struct {
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=status,displayName="Conditions",xDescriptors="urn:alm:descriptor:io.kubernetes.conditions"
	Conditions status.Conditions `json:"conditions,omitempty"`

	// LastSyncSuccessTime represents the time last synchronization completed successfully
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=status,displayName="Last Sync Success Time"
	LastSyncSuccessTime *metav1.Time `json:"lastSyncSuccessTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// GroupSync is the Schema for the groupsyncs API
// +operator-sdk:csv:customresourcedefinitions:displayName="Group Sync"
type GroupSync struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GroupSyncSpec   `json:"spec,omitempty"`
	Status GroupSyncStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GroupSyncList contains a list of GroupSync
type GroupSyncList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GroupSync `json:"items"`
}

// Provider represents the container for a single provider
type Provider struct {
	// Name represents the name of the provider
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Name of the Provider"
	// +kubebuilder:validation:Optional
	Name string `json:"name"`

	*ProviderType `json:",inline"`
}

// ProviderType represents the provider to synchronize against
type ProviderType struct {
	// Azure represents the Azure provider
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Azure Provider"
	// +kubebuilder:validation:Optional
	Azure *AzureProvider `json:"azure,omitempty"`

	// GitHub represents the GitHub provider
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="GitHub Provider"
	// +kubebuilder:validation:Optional
	GitHub *GitHubProvider `json:"github,omitempty"`

	// GitLab represents the GitLab provider
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="GitLab Provider"
	// +kubebuilder:validation:Optional
	GitLab *GitLabProvider `json:"gitlab,omitempty"`

	// Ldap represents the LDAP provider
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="LDAP Provider"
	// +kubebuilder:validation:Optional
	Ldap *LdapProvider `json:"ldap,omitempty"`

	// Keycloak represents the Keycloak provider
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Keycloak Provider"
	// +kubebuilder:validation:Optional
	Keycloak *KeycloakProvider `json:"keycloak,omitempty"`
}

// KeycloakProvider represents integration with Keycloak
type KeycloakProvider struct {

	// CaSecret is a reference to a secret containing a CA certificate to communicate to the Keycloak server
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Secret Containing the CA Certificate",xDescriptors="urn:alm:descriptor:io.kubernetes:Secret"
	// +kubebuilder:validation:Optional
	CaSecret *SecretRef `json:"caSecret,omitempty"`

	// CredentialsSecret is a reference to a secret containing authentication details for the Keycloak server
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Secret Containing the Credentials",xDescriptors="urn:alm:descriptor:io.kubernetes:Secret"
	// +kubebuilder:validation:Required
	CredentialsSecret *SecretRef `json:"credentialsSecret"`

	// Groups represents a filtered list of groups to synchronize
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Groups to Synchronize"
	// +kubebuilder:validation:Optional
	Groups []string `json:"groups,omitempty"`

	// Insecure specifies whether to allow for unverified certificates to be used when communicating to Keycloak
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Ignore SSL Verification",xDescriptors="urn:alm:descriptor:com.tectonic.ui:booleanSwitch"
	// +kubebuilder:validation:Optional
	Insecure bool `json:"insecure,omitempty"`

	// LoginRealm is the Keycloak realm to authenticate against
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Realm to Login Against"
	// +kubebuilder:validation:Optional
	LoginRealm string `json:"loginRealm,omitempty"`

	// Realm is the realm containing the groups to synchronize against
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Realm to Synchronize",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Required
	Realm string `json:"realm"`

	// Scope represents the depth for which groups will be synchronized
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Scope to synchronize against"
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=one;sub
	Scope SyncScope `json:"scope,omitempty"`

	// URL is the location of the Keycloak server
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Keycloak URL",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Required
	URL string `json:"url"`
}

// GitHubProvider represents integration with GitHub
type GitHubProvider struct {
	// CaSecret is a reference to a secret containing a CA certificate to communicate to the GitHub server
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Secret Containing the CA Certificate",xDescriptors="urn:alm:descriptor:io.kubernetes:Secret"
	// +kubebuilder:validation:Optional
	CaSecret *SecretRef `json:"caSecret,omitempty"`

	// CredentialsSecret is a reference to a secret containing authentication details for the GitHub server
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Secret Containing the Credentials",xDescriptors="urn:alm:descriptor:io.kubernetes:Secret"
	// +kubebuilder:validation:Required
	CredentialsSecret *SecretRef `json:"credentialsSecret"`

	// Insecure specifies whether to allow for unverified certificates to be used when communicating to GitHab
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Ignore SSL Verification",xDescriptors="urn:alm:descriptor:com.tectonic.ui:booleanSwitch"
	// +kubebuilder:validation:Optional
	Insecure bool `json:"insecure,omitempty"`

	// Organization represents the location to source teams to synchronize
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Organization to Synchronize",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Optional
	Organization string `json:"organization,omitempty"`

	// Teams represents a filtered list of teams to synchronize
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Teams to Synchronize",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Optional
	Teams []string `json:"teams,omitempty"`

	// URL is the location of the GitHub server
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="GitHub URL",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Optional
	URL *string `json:"url,omitempty"`
}

// GitLabProvider represents integration with GitLab
type GitLabProvider struct {
	// CaSecret is a reference to a secret containing a CA certificate to communicate to the GitLab server
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Secret Containing the CA Certificate",xDescriptors="urn:alm:descriptor:io.kubernetes:Secret"
	// +kubebuilder:validation:Optional
	CaSecret *SecretRef `json:"caSecret,omitempty"`

	// CredentialsSecret is a reference to a secret containing authentication details for the GitLab server
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Secret Containing the Credentials",xDescriptors="urn:alm:descriptor:io.kubernetes:Secret"
	// +kubebuilder:validation:Required
	CredentialsSecret *SecretRef `json:"credentialsSecret"`

	// Insecure specifies whether to allow for unverified certificates to be used when communicating to GitLab
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Ignore SSL Verification",xDescriptors="urn:alm:descriptor:com.tectonic.ui:booleanSwitch"
	// +kubebuilder:validation:Optional
	Insecure bool `json:"insecure,omitempty"`

	// Groups represents a filtered list of groups to synchronize
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Groups to Synchronize",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Optional
	Groups []string `json:"groups,omitempty"`

	// URL is the location of the GitLub server
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="GitLab URL",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Optional
	URL *string `json:"url,omitempty"`
}

// LdapProvider represents integration with an LDAP server
type LdapProvider struct {
	// CaSecret is a reference to a secret containing a CA certificate to communicate to the GitLab server
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Secret Containing the CA Certificate",xDescriptors="urn:alm:descriptor:io.kubernetes:Secret"
	// +kubebuilder:validation:Optional
	CaSecret *SecretRef `json:"caSecret,omitempty"`

	// CredentialsSecret is a reference to a secret containing authentication details for communicating to LDAP
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Secret Containing the Credentials",xDescriptors="urn:alm:descriptor:io.kubernetes:Secret"
	// +kubebuilder:validation:Optional
	CredentialsSecret *SecretRef `json:"credentialsSecret,omitempty"`

	// Insecure specifies whether to allow for unverified certificates to be used when communicating to LDAP
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Ignore SSL Verification",xDescriptors="urn:alm:descriptor:com.tectonic.ui:booleanSwitch"
	// +kubebuilder:validation:Optional
	Insecure bool `json:"insecure,omitempty"`

	/// LDAPGroupUIDToOpenShiftGroupNameMapping is an optional direct mapping of LDAP group UIDs to OpenShift group names
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="LDAP group UID's to OpenShift group name mapping"
	// +kubebuilder:validation:Optional
	LDAPGroupUIDToOpenShiftGroupNameMapping map[string]string `json:"groupUIDNameMapping"`

	// RFC2307Config represents the configuration for a RFC2307 schema
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="RFC2307 configuration"
	// +kubebuilder:validation:Optional
	// +optional
	RFC2307Config *legacyconfigv1.RFC2307Config `json:"rfc2307,omitempty"`
	// ActiveDirectoryConfig represents the configuration for Active Directory
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Active Directory configuration"
	// +kubebuilder:validation:Optional
	ActiveDirectoryConfig *legacyconfigv1.ActiveDirectoryConfig `json:"activeDirectory,omitempty"`

	// ActiveDirectoryConfig represents the configuration for Augmented Active Directory
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Augmented Active Directory configuration"
	// +kubebuilder:validation:Optional
	AugmentedActiveDirectoryConfig *legacyconfigv1.AugmentedActiveDirectoryConfig `json:"augmentedActiveDirectory,omitempty"`

	// URL is the location of the LDAP Server
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Azure URL",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Required
	URL *string `json:"url"`

	// Whitelist represents a list of groups to synchronize
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Whitelisted groups to synchronize",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Optional
	Whitelist *[]string `json:"whitelist,omitempty"`

	// Blacklist represents a list of groups to not synchronize
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Blacklisted groups to not synchronize",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Optional
	Blacklist *[]string `json:"blacklist,omitempty"`
}

// AzureProvider represents integration with Azure
type AzureProvider struct {
	// CredentialsSecret is a reference to a secret containing authentication details for communicating to Azure
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Secret Containing the Credentials",xDescriptors="urn:alm:descriptor:io.kubernetes:Secret"
	// +kubebuilder:validation:Required
	CredentialsSecret *SecretRef `json:"credentialsSecret"`

	// Filter allows for limiting the results from the groups response using the Filter feature of the Azure Graph API
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Filter",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Optional
	Filter string `json:"filter,omitempty"`

	// Insecure specifies whether to allow for unverified certificates to be used when communicating to Azure
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Ignore SSL Verification",xDescriptors="urn:alm:descriptor:com.tectonic.ui:booleanSwitch"
	// +kubebuilder:validation:Optional
	Insecure bool `json:"insecure,omitempty"`

	// Groups represents a filtered list of groups to synchronize
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Groups to Synchronize",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Optional
	Groups []string `json:"groups,omitempty"`

	// URL is the location of the Azure platform
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Azure URL",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Optional
	URL *string `json:"url,omitempty"`
}

// SecretRef represents a reference to an item within a Secret
type SecretRef struct {
	// Name represents the name of the secret
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Name of the secret",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Namespace represents the namespace containing the secret
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Namespace containing the secret",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Required
	Namespace string `json:"namespace"`

	// Key represents the specific key to reference from the secret
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Key within the secret",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Optional
	Key string `json:"key,omitempty"`
}

func init() {
	SchemeBuilder.Register(&GroupSync{}, &GroupSyncList{})
}
