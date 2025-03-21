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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SyncScope string
type ObjectRefKind string

const (
	OneSyncScope SyncScope = "one"
	SubSyncScope SyncScope = "sub"

	ConfigMapObjectRefKind ObjectRefKind = "ConfigMap"
	SecretMapObjectRefKind ObjectRefKind = "Secret"
)

// GroupSyncSpec defines the desired state of GroupSync
// +k8s:openapi-gen=true
type GroupSyncSpec struct {

	// List of Providers that can be mounted by containers belonging to the pod.
	// +patchMergeKey=name
	// +patchStrategy=merge,retainKeys
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Providers"
	Providers []Provider `json:"providers,omitempty" patchStrategy:"merge,retainKeys" patchMergeKey:"name" protobuf:"bytes,1,rep,name=providers"`

	// Schedule represents a cron based configuration for synchronization
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Schedule",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Optional
	Schedule string `json:"schedule,omitempty"`

	// ExcludeInvalidGroupNames excludes Groups with names that are not RFC 1035 compliant.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Exclude Invalid Group Names",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	// +kubebuilder:validation:Optional
	ExcludeInvalidGroupNames bool `json:"excludeInvalidGroupNames,omitempty"`
}

// GroupSyncStatus defines the observed state of GroupSync
// +k8s:openapi-gen=true
type GroupSyncStatus struct {
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=status,displayName="Conditions",xDescriptors={"urn:alm:descriptor:io.kubernetes.conditions"}
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// LastSyncSuccessTime represents the time last synchronization completed successfully
	// +kubebuilder:validation:Optional
	// +operator-sdk:csv:customresourcedefinitions:type=status,displayName="Last Sync Success Time"
	LastSyncSuccessTime *metav1.Time `json:"lastSyncSuccessTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// GroupSync is the Schema for the groupsyncs API
// +operator-sdk:csv:customresourcedefinitions:displayName="Group Sync"
// +kubebuilder:resource:path=groupsyncs,scope=Namespaced
// +k8s:openapi-gen=true
type GroupSync struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GroupSyncSpec   `json:"spec,omitempty"`
	Status GroupSyncStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GroupSyncList contains a list of GroupSync
// +k8s:openapi-gen=true
type GroupSyncList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GroupSync `json:"items"`
}

// Provider represents the container for a single provider
// +k8s:openapi-gen=true
type Provider struct {
	// Name represents the name of the provider
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Name of the Provider"
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	*ProviderType `json:",inline"`
}

// ProviderType represents the provider to synchronize against
// +k8s:openapi-gen=true
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

	// Okta represents the Okta provider
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Okta Provider"
	// +kubebuilder:validation:Optional
	Okta *OktaProvider `json:"okta,omitempty"`

	// IbmSecurityVerify represents the IBM Security Verify provider
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="IBM Security Verify"
	// +kubebuilder:validation:Optional
	IbmSecurityVerify *IbmSecurityVerifyProvider `json:"ibmsecurityverify,omitempty"`
}

// KeycloakProvider represents integration with Keycloak
// +k8s:openapi-gen=true
type KeycloakProvider struct {

	// Ca is a reference to a Secret or ConfigMap containing a CA certificate to communicate to the Keycloak server
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Resource Containing the CA Certificate",xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret"}
	// +kubebuilder:validation:Optional
	Ca *ObjectRef `json:"ca,omitempty"`

	// CaSecret is a reference to a secret containing a CA certificate to communicate to the Keycloak server
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Resource Containing the CA Certificate",xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret"}
	// +kubebuilder:validation:Optional
	// Deprecated: Use Ca instead.
	CaSecret *ObjectRef `json:"caSecret,omitempty"`

	// CredentialsSecret is a reference to a secret containing authentication details for the Keycloak server
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Secret Containing the Credentials",xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret"}
	// +kubebuilder:validation:Required
	CredentialsSecret *ObjectRef `json:"credentialsSecret"`

	// Groups represents a filtered list of groups to synchronize
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Groups to Synchronize"
	// +kubebuilder:validation:Optional
	Groups []string `json:"groups,omitempty"`

	// Insecure specifies whether to allow for unverified certificates to be used when communicating to Keycloak
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Ignore SSL Verification",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	// +kubebuilder:validation:Optional
	Insecure bool `json:"insecure,omitempty"`

	// LoginRealm is the Keycloak realm to authenticate against
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Realm to Login Against"
	// +kubebuilder:validation:Optional
	LoginRealm string `json:"loginRealm,omitempty"`

	// Realm is the realm containing the groups to synchronize against
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Realm to Synchronize",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Required
	Realm string `json:"realm"`

	// Scope represents the depth for which groups will be synchronized
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Scope to synchronize against"
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=one;sub
	Scope SyncScope `json:"scope,omitempty"`

	// URL is the location of the Keycloak server
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Keycloak URL",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Required
	URL string `json:"url"`

	// Prune Whether to prune groups that are no longer in Keycloak. Default is false
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Prune",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	// +kubebuilder:validation:Optional
	Prune bool `json:"prune"`
}

// GitHubProvider represents integration with GitHub
// +k8s:openapi-gen=true
type GitHubProvider struct {

	// Ca is a reference to a Secret or ConfigMap containing a CA certificate to communicate to the GitHub server
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Resource Containing the CA Certificate",xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret"}
	// +kubebuilder:validation:Optional
	Ca *ObjectRef `json:"ca,omitempty"`

	// CaSecret is a reference to a secret containing a CA certificate to communicate to the GitHub server
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Resource Containing the CA Certificate",xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret"}
	// +kubebuilder:validation:Optional
	// Deprecated: Use Ca instead.
	CaSecret *ObjectRef `json:"caSecret,omitempty"`

	// CredentialsSecret is a reference to a secret containing authentication details for the GitHub server
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Secret Containing the Credentials",xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret"}
	// +kubebuilder:validation:Required
	CredentialsSecret *ObjectRef `json:"credentialsSecret"`

	// Insecure specifies whether to allow for unverified certificates to be used when communicating to GitHab
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Ignore SSL Verification",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	// +kubebuilder:validation:Optional
	Insecure bool `json:"insecure,omitempty"`

	// Organization represents the location to source teams to synchronize
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Organization to Synchronize",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Optional
	Organization string `json:"organization,omitempty"`

	// Teams represents a filtered list of teams to synchronize
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Teams to Synchronize",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Optional
	Teams []string `json:"teams,omitempty"`

	// Map users by SCIM Id. This will usually match your IDP id, like UPN when using AAD.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Map users by SCIM-ID",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Optional
	MapByScimId bool `json:"mapByScimId,omitempty"`

	// URL is the location of the GitHub server
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="GitHub URL",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Required
	// +kubebuilder:default="https://api.github.com/"
	URL *string `json:"url,omitempty"`

	// V4URL is the location of the GitHub server graphql endpoint.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="GitHub v4URL (graphql)",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="https://api.github.com/graphql"
	V4URL *string `json:"v4url,omitempty"`

	// Prune Whether to prune groups that are no longer in GitHub. Default is false
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Prune",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	// +kubebuilder:validation:Optional
	Prune bool `json:"prune"`
}

// GitLabProvider represents integration with GitLab
// +k8s:openapi-gen=true
type GitLabProvider struct {

	// Ca is a reference to a Secret or ConfigMap containing a CA certificate to communicate to the GitLab server
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Resource Containing the CA Certificate",xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret"}
	// +kubebuilder:validation:Optional
	Ca *ObjectRef `json:"ca,omitempty"`

	// CaSecret is a reference to a secret containing a CA certificate to communicate to the GitLab server
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Resource Containing the CA Certificate",xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret"}
	// +kubebuilder:validation:Optional
	// Deprecated: Use Ca instead.
	CaSecret *ObjectRef `json:"caSecret,omitempty"`

	// CredentialsSecret is a reference to a secret containing authentication details for the GitLab server
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Secret Containing the Credentials",xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret"}
	// +kubebuilder:validation:Required
	CredentialsSecret *ObjectRef `json:"credentialsSecret"`

	// Groups represents a filtered list of groups to synchronize
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Groups to Synchronize",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Optional
	Groups []string `json:"groups,omitempty"`

	// Insecure specifies whether to allow for unverified certificates to be used when communicating to GitLab
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Ignore SSL Verification",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	// +kubebuilder:validation:Optional
	Insecure bool `json:"insecure,omitempty"`

	// Prune Whether to prune groups that are no longer in GitLab. Default is false
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Prune",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	// +kubebuilder:validation:Optional
	Prune bool `json:"prune"`

	// Scope represents the depth for which groups will be synchronized
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Scope to synchronize against"
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=one;sub
	Scope SyncScope `json:"scope,omitempty"`

	// URL is the location of the GitLab server
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="GitLab URL",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Optional
	// +kubebuilder:default="https://gitlab.com"
	URL *string `json:"url,omitempty"`
}

// LdapProvider represents integration with an LDAP server
// +k8s:openapi-gen=true
type LdapProvider struct {

	// Ca is a reference to a Secret or ConfigMap containing a CA certificate to communicate to LDAP
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Resource Containing the CA Certificate",xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret"}
	// +kubebuilder:validation:Optional
	Ca *ObjectRef `json:"ca,omitempty"`

	// CaSecret is a reference to a secret containing a CA certificate to communicate to LDAP
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Resource Containing the CA Certificate",xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret"}
	// +kubebuilder:validation:Optional
	// Deprecated: Use Ca instead.
	CaSecret *ObjectRef `json:"caSecret,omitempty"`

	// CredentialsSecret is a reference to a secret containing authentication details for communicating to LDAP
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Secret Containing the Credentials",xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret"}
	// +kubebuilder:validation:Optional
	CredentialsSecret *ObjectRef `json:"credentialsSecret,omitempty"`

	// Insecure specifies whether to allow for unverified certificates to be used when communicating to LDAP
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Ignore SSL Verification",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
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
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="LDAP URL",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Required
	URL *string `json:"url"`

	// Whitelist represents a list of groups to synchronize
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Whitelisted groups to synchronize",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Optional
	Whitelist *[]string `json:"whitelist,omitempty"`

	// Blacklist represents a list of groups to not synchronize
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Blacklisted groups to not synchronize",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Optional
	Blacklist *[]string `json:"blacklist,omitempty"`

	// Prune Whether to prune groups that are no longer in LDAP. Default is false
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Prune",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	// +kubebuilder:validation:Optional
	Prune bool `json:"prune"`
}

// AzureProvider represents integration with Azure
// +k8s:openapi-gen=true
type AzureProvider struct {
	// BaseGroups allows for a set of groups to be specified to start searching from instead of searching all groups in the directory
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Base Groups",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Optional
	BaseGroups []string `json:"baseGroups,omitempty"`

	// Ca is a reference to a Secret or ConfigMap containing a CA certificate to communicate to Azure
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Resource Containing the CA Certificate",xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret"}
	// +kubebuilder:validation:Optional
	Ca *ObjectRef `json:"ca,omitempty"`

	// CaSecret is a reference to a secret containing a CA certificate to communicate to Azure
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Resource Containing the CA Certificate",xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret"}
	// +kubebuilder:validation:Optional
	// Deprecated: Use Ca instead.
	CaSecret *ObjectRef `json:"caSecret,omitempty"`

	// CredentialsSecret is a reference to a secret containing authentication details for communicating to Azure
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Secret Containing the Credentials",xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret"}
	// +kubebuilder:validation:Required
	CredentialsSecret *ObjectRef `json:"credentialsSecret"`

	// Filter allows for limiting the results from the groups response using the Filter feature of the Azure Graph API
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Filter",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Optional
	Filter string `json:"filter,omitempty"`

	// Insecure specifies whether to allow for unverified certificates to be used when communicating to Azure
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Ignore SSL Verification",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	// +kubebuilder:validation:Optional
	Insecure bool `json:"insecure,omitempty"`

	// Groups represents a filtered list of groups to synchronize
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Groups to Synchronize",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Optional
	Groups []string `json:"groups,omitempty"`

	// AuthorityHost is the location of the Azure Active Directory endpoint
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Azure URL",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Optional
	AuthorityHost *string `json:"authorityHost,omitempty"`

	// UserNameAttributes are the fields to consider on the User object containing the username
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Azure UserName Attributes",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Optional
	UserNameAttributes *[]string `json:"userNameAttributes,omitempty"`

	// Prune Whether to prune groups that are no longer in Azure. Default is false
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Prune",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	// +kubebuilder:validation:Optional
	Prune bool `json:"prune"`
}

// OktaProvider represents integration with Okta
// +k8s:openapi-gen=true
type OktaProvider struct {
	// CredentialsSecret is a reference to a secret containing authentication details for the Okta server
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Secret Containing the Credentials",xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret"}
	// +kubebuilder:validation:Required
	CredentialsSecret *ObjectRef `json:"credentialsSecret"`
	// Groups represents a filtered list of groups to synchronize
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Groups to Synchronize",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Optional
	Groups []string `json:"groups,omitempty"`
	// URL is the location of the Okta domain server
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Okta URL",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Required
	URL string `json:"url"`
	// AppId is the id of the application we are syncing groups for
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="App ID",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Required
	AppId string `json:"appId"`
	// ExtractLoginUsername is true if Okta username's are defaulted to emails and you would like the username only
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Extract Login Username",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Optional
	ExtractLoginUsername bool `json:"extractLoginUsername"`
	// ProfileKey the attribute from Okta you would like to use as the user identifier.  Default is "login"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Profile Key",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Optional
	ProfileKey string `json:"profileKey"`
	// GroupLimit is the maximum number of groups that are requested from OKTA per request.  Multiple requests will be made using pagination if you have more groups than this limit. Default is "1000"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Group Limit",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number"}
	// +kubebuilder:validation:Optional
	GroupLimit int `json:"groupLimit"`

	// Prune Whether to prune groups that are no longer in OKTA. Default is false
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Prune",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	// +kubebuilder:validation:Optional
	Prune bool `json:"prune"`
}

// IbmSecurityVerifyProvider represents integration with IBM Security Verify
// +k8s:openapi-gen=true
type IbmSecurityVerifyProvider struct {
	// CredentialsSecret is a reference to a secret containing authentication details for the IBM Security Verify server
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Secret Containing the Credentials",xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret"}
	// +kubebuilder:validation:Required
	CredentialsSecret *ObjectRef `json:"credentialsSecret"`
	// Groups is the list of ISV groups to synchronize
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Groups to Synchronize",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Required
	Groups []IsvGroupSpec `json:"groups,omitempty"`
	// TenantURL is the location of the IBM Security Verify tenant
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Tenant URL",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Required
	TenantURL string `json:"tenantUrl"`
}

// +k8s:openapi-gen=true
type IsvGroupSpec struct {
	// The display name of the group as defined in IBM Security Verify
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Name",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Required
	Name string `json:"name,omitempty"`
	// The ID of the group as defined in IBM Security Verify. This value can be found by using the API.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Id",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Required
	Id string `json:"id,omitempty"`
}

// ObjectRef represents a reference to an item within a Secret
// +k8s:openapi-gen=true
type ObjectRef struct {

	// Key represents the specific key to reference from the resource
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Key",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Optional
	Key string `json:"key,omitempty"`

	// Name represents the name of the resource
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Name",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Namespace represents the namespace containing the resource
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Namespace",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	// +kubebuilder:validation:Required
	Namespace string `json:"namespace"`

	// Kind is a string value representing the resource type
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Kind",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:select:ConfigMap","urn:alm:descriptor:com.tectonic.ui:select:Secret"}
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum:={"ConfigMap","Secret"}
	// +kubebuilder:default="Secret"
	Kind ObjectRefKind `json:"kind,omitempty"`
}

func (g *GroupSync) GetConditions() []metav1.Condition {
	return g.Status.Conditions
}

func (g *GroupSync) SetConditions(conditions []metav1.Condition) {
	g.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&GroupSync{}, &GroupSyncList{})
}
