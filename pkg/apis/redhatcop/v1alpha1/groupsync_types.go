package v1alpha1

import (
	"github.com/operator-framework/operator-sdk/pkg/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

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

	ResyncPeriodMinutes *int `json:"resyncPeriodMinutes,omitempty"`
}

// GroupSyncStatus defines the observed state of GroupSync
type GroupSyncStatus struct {
	Conditions status.Conditions `json:"conditions"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// GroupSync is the Schema for the groupsyncs API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=groupsyncs,scope=Namespaced
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
	Name string `json:"name"`

	*ProviderType `json:",inline"`
}

// ProviderType represents the provider to synchronize against
type ProviderType struct {
	Keycloak *KeycloakProvider `json:"keycloak,omitempty"`
}

// KeycloakProvider represents integration with Keycloak
type KeycloakProvider struct {
	URL        string `json:"url"`
	LoginRealm string `json:"loginRealm,omitempty"`
	Realm      string `json:"realm"`
	SecretName string `json:"secretName"`
	Insecure   bool   `json:"insecure,omitempty"`
	// +kubebuilder:validation:Enum=one;sub
	Scope SyncScope `json:"scope,omitempty"`
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
