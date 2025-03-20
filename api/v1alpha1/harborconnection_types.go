package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// HarborConnectionSpec defines the desired state of HarborConnection.
type HarborConnectionSpec struct {
	// BaseURL is the Harbor API endpoint.
	// +kubebuilder:validation:Format=url
	BaseURL string `json:"baseURL"`

	// Credentials holds the default credentials for Harbor API calls.
	Credentials *Credentials `json:"credentials,omitempty"`
}

// Credentials holds default authentication details.
type Credentials struct {
	// Type of the credential, e.g., "basic".
	// +kubebuilder:validation:Enum=basic
	Type string `json:"type"`

	// AccessKey for authentication.
	// +kubebuilder:validation:MinLength=1
	AccessKey string `json:"accessKey"`

	// AccessSecretRef is a reference to a Kubernetes Secret containing the access secret.
	AccessSecretRef string `json:"accessSecretRef"`
}

// HarborConnectionStatus defines the observed state of HarborConnection.
type HarborConnectionStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=hc

// HarborConnection is the Schema for the harborconnections API.
type HarborConnection struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HarborConnectionSpec   `json:"spec,omitempty"`
	Status HarborConnectionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// HarborConnectionList contains a list of HarborConnection.
type HarborConnectionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HarborConnection `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HarborConnection{}, &HarborConnectionList{})
}
