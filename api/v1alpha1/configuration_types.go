package v1alpha1

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConfigurationValue selects a literal or Secret-backed Harbor configuration
// value.
// +kubebuilder:validation:MinProperties=1
// +kubebuilder:validation:MaxProperties=1
type ConfigurationValue struct {
	// Value is the literal configuration value.
	// +optional
	Value *apiextensionsv1.JSON `json:"value,omitempty"`

	// ValueFrom selects an external source for the configuration value.
	// +optional
	ValueFrom *ConfigurationValueSource `json:"valueFrom,omitempty"`
}

// ConfigurationValueSource selects an external source for a Harbor
// configuration value.
type ConfigurationValueSource struct {
	// SecretKeyRef references the Secret key containing the configuration value.
	SecretKeyRef SecretReference `json:"secretKeyRef"`
}

// ConfigurationSpec defines the desired state of Harbor system configuration.
type ConfigurationSpec struct {
	HarborSpecBase `json:",inline"`

	// Settings contains Harbor configuration keys and their desired value sources.
	// +optional
	Settings map[string]ConfigurationValue `json:"settings,omitempty"`
}

// ConfigurationStatus defines the observed state of Configuration.
type ConfigurationStatus struct {
	HarborStatusBase `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=harbor
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].message`,priority=1
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Configuration is the Schema for the configurations API.
type Configuration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConfigurationSpec   `json:"spec,omitempty"`
	Status ConfigurationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ConfigurationList contains a list of Configuration.
type ConfigurationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Configuration `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Configuration{}, &ConfigurationList{})
}
