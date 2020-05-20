package v1alpha1

import (
	"github.com/operator-framework/operator-sdk/pkg/status"
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
	Providers []Provider `json:"providers,omitempty" patchStrategy:"merge,retainKeys" patchMergeKey:"name" protobuf:"bytes,1,rep,name=providers"`

	Schedule string `json:"schedule,omitempty"`
}

// GroupSyncStatus defines the observed state of GroupSync
type GroupSyncStatus struct {
	// +kubebuilder:validation:Required
	Conditions status.Conditions `json:"conditions"`
	// +kubebuilder:validation:Optional
	LastSyncSuccessTime *metav1.Time `json:"lastSyncSuccessTime,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GroupSync is the Schema for the groupsyncs API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=groupsyncs,scope=Cluster
type GroupSync struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GroupSyncSpec   `json:"spec,omitempty"`
	Status GroupSyncStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GroupSyncList contains a list of GroupSync
type GroupSyncList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GroupSync `json:"items"`
}

// Provider represents the container for a single provider
type Provider struct {
	// Name represents the name of the provider
	// +kubebuilder:validation:Optional
	Name string `json:"name"`

	*ProviderType `json:",inline"`
}

// ProviderType represents the provider to synchronize against
type ProviderType struct {
	// Azure represents the Azure provider
	// +kubebuilder:validation:Optional
	Azure *AzureProvider `json:"azure,omitempty"`
	// GitHub represents the GitHub provider
	// +kubebuilder:validation:Optional
	GitHub *GitHubProvider `json:"github,omitempty"`
	// GitLab represents the GitLab provider
	// +kubebuilder:validation:Optional
	GitLab *GitLabProvider `json:"gitlab,omitempty"`
	// Keycloak represents the Keycloak provider
	// +kubebuilder:validation:Optional
	Keycloak *KeycloakProvider `json:"keycloak,omitempty"`
}

// KeycloakProvider represents integration with Keycloak
type KeycloakProvider struct {
	// CaSecret is a reference to a secret containing a CA certificate to communicate to the Keycloak server
	// +kubebuilder:validation:Optional
	CaSecret *SecretRef `json:"caSecret,omitempty"`
	// CredentialsSecret is a reference to a secret containing authentication details for the Keycloak server
	// +kubebuilder:validation:Required
	CredentialsSecret *SecretRef `json:"credentialsSecret"`
	// Groups represents a filtered list of groups to synchronize
	// +kubebuilder:validation:Optional
	Groups []string `json:"groups,omitempty"`
	// Insecure specifies whether to allow for unverified certificates to be used when communicating to Keycloak
	// +kubebuilder:validation:Optional
	Insecure bool `json:"insecure,omitempty"`
	// LoginRealm is the Keycloak realm to authenticate against
	// +kubebuilder:validation:Optional
	LoginRealm string `json:"loginRealm,omitempty"`
	// Realm is the realm containing the groups to synchronize against
	// +kubebuilder:validation:Required
	Realm string `json:"realm"`
	// Scope represents the depth for which groups will be synchronized
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Enum=one;sub
	Scope SyncScope `json:"scope,omitempty"`
	// URL is the location of the Keycloak server
	// +kubebuilder:validation:Required
	URL string `json:"url"`
}

// GitHubProvider represents integration with GitHub
type GitHubProvider struct {
	// CaSecret is a reference to a secret containing a CA certificate to communicate to the GitHub server
	// +kubebuilder:validation:Optional
	CaSecret *SecretRef `json:"caSecret,omitempty"`
	// CredentialsSecret is a reference to a secret containing authentication details for the GitHub server
	// +kubebuilder:validation:Required
	CredentialsSecret *SecretRef `json:"credentialsSecret"`
	// Insecure specifies whether to allow for unverified certificates to be used when communicating to GitHab
	// +kubebuilder:validation:Optional
	Insecure bool `json:"insecure,omitempty"`
	// Organization represents the location to source teams to synchronize
	// +kubebuilder:validation:Optional
	Organization string `json:"organization,omitempty"`
	// Teams represents a filtered list of teams to synchronize
	// +kubebuilder:validation:Optional
	Teams []string `json:"teams,omitempty"`
	// URL is the location of the GitHub server
	// +kubebuilder:validation:Optional
	URL *string `json:"url,omitempty"`
}

// GitLabProvider represents integration with GitLab
type GitLabProvider struct {
	// CaSecret is a reference to a secret containing a CA certificate to communicate to the GitLab server
	// +kubebuilder:validation:Optional
	CaSecret *SecretRef `json:"caSecret,omitempty"`
	// CredentialsSecret is a reference to a secret containing authentication details for the GitLab server
	// +kubebuilder:validation:Required
	CredentialsSecret *SecretRef `json:"credentialsSecret"`
	// Insecure specifies whether to allow for unverified certificates to be used when communicating to GitLab
	// +kubebuilder:validation:Optional
	Insecure bool `json:"insecure,omitempty"`
	// Groups represents a filtered list of groups to synchronize
	// +kubebuilder:validation:Optional
	Groups []string `json:"groups,omitempty"`
	// URL is the location of the GitLub server
	// +kubebuilder:validation:Optional
	URL *string `json:"url,omitempty"`
}

// AzureProvider represents integration with Azure
type AzureProvider struct {
	// CredentialsSecret is a reference to a secret containing authentication details for communicating to Azure
	// +kubebuilder:validation:Required
	CredentialsSecret *SecretRef `json:"credentialsSecret"`
	// Insecure specifies whether to allow for unverified certificates to be used when communicating to Azure
	// +kubebuilder:validation:Optional
	Insecure bool `json:"insecure,omitempty"`
	// Groups represents a filtered list of groups to synchronize
	// +kubebuilder:validation:Optional
	Groups []string `json:"groups,omitempty"`
	// URL is the location of the Azure platform
	// +kubebuilder:validation:Optional
	URL *string `json:"url,omitempty"`
}

// SecretRef represents a reference to an item within a Secret
type SecretRef struct {
	// Name represents the name of the secret
	// +kubebuilder:validation:Required
	Name string `json:"name"`
	// Namespace represents the namespace containing the secret
	// +kubebuilder:validation:Required
	Namespace string `json:"namespace"`
	// Key represents the specific key to reference from the secret
	// +kubebuilder:validation:Optional
	Key string `json:"key,omitempty"`
}

func (s *GroupSync) GetReconcileStatus() status.Conditions {
	return s.Status.Conditions
}

func (s *GroupSync) SetReconcileStatus(reconcileStatus status.Conditions) {
	s.Status.Conditions = reconcileStatus
}

func init() {
	SchemeBuilder.Register(&GroupSync{}, &GroupSyncList{})
}
