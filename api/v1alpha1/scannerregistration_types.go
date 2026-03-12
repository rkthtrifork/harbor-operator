package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// ScannerRegistrationSpec defines the desired state of ScannerRegistration.
// +kubebuilder:validation:XValidation:rule="!(has(self.accessCredentialSecretRef) && size(self.accessCredential) > 0)",message="accessCredential and accessCredentialSecretRef are mutually exclusive"
type ScannerRegistrationSpec struct {
	HarborSpecBase `json:",inline"`

	// AllowTakeover indicates whether the operator is allowed to adopt an
	// existing scanner registration in Harbor with the same name.
	// +optional
	AllowTakeover bool `json:"allowTakeover,omitempty"`

	// Name is the registration name.
	// Defaults to metadata.name when omitted.
	// +optional
	Name string `json:"name,omitempty"`

	// Description is an optional description.
	// +optional
	Description string `json:"description,omitempty"`

	// URL is the scanner adapter base URL.
	// +kubebuilder:validation:Format=uri
	URL string `json:"url"`

	// Auth defines the authentication approach (e.g. Basic, Bearer, X-ScannerAdapter-API-Key).
	// +optional
	Auth string `json:"auth,omitempty"`

	// AccessCredential is the credential value sent in the auth header.
	// +optional
	AccessCredential string `json:"accessCredential,omitempty"`

	// AccessCredentialSecretRef references a secret value holding the credential.
	// +optional
	AccessCredentialSecretRef *SecretReference `json:"accessCredentialSecretRef,omitempty"`

	// SkipCertVerify indicates whether to skip certificate verification.
	// +optional
	SkipCertVerify bool `json:"skipCertVerify,omitempty"`

	// UseInternalAddr indicates whether the scanner uses Harbor's internal address.
	// +optional
	UseInternalAddr bool `json:"useInternalAddr,omitempty"`

	// Disabled indicates whether the registration is disabled.
	// +optional
	Disabled bool `json:"disabled,omitempty"`

	// Default indicates whether this scanner should be set as system default.
	// +optional
	Default bool `json:"default,omitempty"`
}

// ScannerRegistrationStatus defines the observed state of ScannerRegistration.
type ScannerRegistrationStatus struct {
	HarborStatusBase `json:",inline"`

	// HarborScannerID is the ID of the registration in Harbor.
	HarborScannerID string `json:"harborScannerID,omitempty"`

	// CredentialHash is a hash of the configured credential to detect changes.
	// +optional
	CredentialHash string `json:"credentialHash,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=harbor
// +kubebuilder:printcolumn:name="URL",type=string,JSONPath=`.spec.url`
// +kubebuilder:printcolumn:name="Default",type=boolean,JSONPath=`.spec.default`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].message`,priority=1
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ScannerRegistration is the Schema for the scannerregistrations API.
type ScannerRegistration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ScannerRegistrationSpec   `json:"spec,omitempty"`
	Status ScannerRegistrationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ScannerRegistrationList contains a list of ScannerRegistration.
type ScannerRegistrationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ScannerRegistration `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ScannerRegistration{}, &ScannerRegistrationList{})
}
