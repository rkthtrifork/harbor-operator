package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RegistrySpec defines the desired state of Registry.
type RegistrySpec struct {
	HarborSpecBase `json:",inline"`

	// AllowTakeover indicates whether the operator is allowed to adopt an
	// existing registry in Harbor with the same name.
	// +optional
	AllowTakeover bool `json:"allowTakeover,omitempty"`

	// Type of the registry, e.g., "github-ghcr".
	// +kubebuilder:validation:Enum=github-ghcr;ali-acr;aws-ecr;azure-acr;docker-hub;docker-registry;google-gcr;harbor;huawei-SWR;jfrog-artifactory;tencent-tcr;volcengine-cr
	Type string `json:"type"`

	// Name is the registry name.
	// It is recommended to leave this field empty so that the operator defaults it
	// to the custom resource's metadata name.
	// +optional
	Name string `json:"name,omitempty"`

	// Description is an optional description.
	// +optional
	Description string `json:"description,omitempty"`

	// URL is the registry URL.
	// +kubebuilder:validation:Format=url
	URL string `json:"url"`

	// Credential holds authentication details for the registry.
	// +optional
	Credential *RegistryCredentialSpec `json:"credential,omitempty"`

	// CACertificate is the PEM-encoded CA certificate for this registry endpoint.
	// +optional
	CACertificate string `json:"caCertificate,omitempty"`

	// CACertificateRef references a secret value holding the PEM-encoded CA certificate.
	// If set, it overrides CACertificate.
	// +optional
	CACertificateRef *SecretReference `json:"caCertificateRef,omitempty"`

	// Insecure indicates if remote certificates should be verified.
	Insecure bool `json:"insecure"`
}

// RegistryCredentialSpec defines registry authentication details.
type RegistryCredentialSpec struct {
	// Type of the credential, e.g. "basic" or "oauth".
	// +kubebuilder:validation:Enum=basic;oauth
	Type string `json:"type"`

	// AccessKeySecretRef references the secret key holding the access key (username).
	AccessKeySecretRef SecretReference `json:"accessKeySecretRef"`

	// AccessSecretSecretRef references the secret key holding the access secret (password/token).
	AccessSecretSecretRef SecretReference `json:"accessSecretSecretRef"`
}

// RegistryStatus defines the observed state of Registry.
type RegistryStatus struct {
	HarborStatusBase `json:",inline"`

	// HarborRegistryID is the ID of the registry in Harbor.
	HarborRegistryID int `json:"harborRegistryID,omitempty"`

	// CredentialHash is a hash of the configured credential to detect changes.
	// +optional
	CredentialHash string `json:"credentialHash,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.type`
// +kubebuilder:printcolumn:name="URL",type=string,JSONPath=`.spec.url`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].message`,priority=1
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Registry is the Schema for the registries API.
type Registry struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RegistrySpec   `json:"spec,omitempty"`
	Status RegistryStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RegistryList contains a list of Registry.
type RegistryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Registry `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Registry{}, &RegistryList{})
}
