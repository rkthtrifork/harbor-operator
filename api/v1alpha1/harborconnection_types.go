package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// HarborConnectionSpec defines the desired state of HarborConnection.
// +kubebuilder:validation:XValidation:rule="!(has(self.caBundleSecretRef) && size(self.caBundle) > 0)",message="caBundle and caBundleSecretRef are mutually exclusive"
type HarborConnectionSpec struct {
	// BaseURL is the Harbor API endpoint.
	// +kubebuilder:validation:Format=url
	BaseURL string `json:"baseURL"`

	// Credentials holds the default credentials for Harbor API calls.
	Credentials *Credentials `json:"credentials,omitempty"`

	// CABundle is a PEM-encoded CA bundle for validating Harbor TLS certificates.
	// +optional
	CABundle string `json:"caBundle,omitempty"`

	// CABundleSecretRef references a Secret containing a PEM-encoded CA bundle.
	// When set, it is mutually exclusive with caBundle.
	// +optional
	CABundleSecretRef *SecretReference `json:"caBundleSecretRef,omitempty"`
}

// Credentials holds default authentication details.
type Credentials struct {
	// Type of the credential, e.g., "basic".
	// +kubebuilder:default=basic
	// +kubebuilder:validation:Enum=basic
	Type string `json:"type"`

	// Username for authentication.
	// +kubebuilder:validation:MinLength=1
	Username string `json:"username"`

	// PasswordSecretRef points to the Kubernetes Secret that stores the password / token.
	PasswordSecretRef SecretReference `json:"passwordSecretRef"`
}

// HarborConnectionStatus defines the observed state of HarborConnection.
type HarborConnectionStatus struct {
	HarborStatusBase `json:",inline"`

	// Authenticated indicates whether the connection was successfully authenticated.
	Authenticated bool `json:"authenticated,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=harbor
// +kubebuilder:printcolumn:name="BaseURL",type=string,JSONPath=`.spec.baseURL`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].message`,priority=1
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

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
	SchemeBuilder.Register(&HarborConnection{}, &HarborConnectionList{}, &ClusterHarborConnection{}, &ClusterHarborConnectionList{})
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,categories=harbor
// +kubebuilder:printcolumn:name="BaseURL",type=string,JSONPath=`.spec.baseURL`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].message`,priority=1
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ClusterHarborConnection is the Schema for the clusterharborconnections API.
type ClusterHarborConnection struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HarborConnectionSpec   `json:"spec,omitempty"`
	Status HarborConnectionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterHarborConnectionList contains a list of ClusterHarborConnection.
type ClusterHarborConnectionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterHarborConnection `json:"items"`
}
