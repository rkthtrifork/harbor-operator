package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// WebhookTargetSpec defines a single webhook target.
type WebhookTargetSpec struct {
	// Type defines the webhook notify type.
	// +optional
	Type string `json:"type,omitempty"`

	// Address is the webhook target address.
	// +optional
	Address string `json:"address,omitempty"`

	// AuthHeader is the auth header to send to the webhook target.
	// +optional
	AuthHeader string `json:"authHeader,omitempty"`

	// AuthHeaderSecretRef references a secret value holding the auth header.
	// +optional
	AuthHeaderSecretRef *SecretReference `json:"authHeaderSecretRef,omitempty"`

	// PayloadFormat is the payload format (e.g. CloudEvents).
	// +optional
	PayloadFormat string `json:"payloadFormat,omitempty"`

	// SkipCertVerify indicates whether to skip TLS certificate verification.
	// +optional
	SkipCertVerify bool `json:"skipCertVerify,omitempty"`
}

// WebhookPolicySpec defines the desired state of WebhookPolicy.
type WebhookPolicySpec struct {
	HarborSpecBase `json:",inline"`

	// AllowTakeover indicates whether the operator is allowed to adopt an
	// existing webhook policy in Harbor with the same name.
	// +optional
	AllowTakeover bool `json:"allowTakeover,omitempty"`

	// ProjectRef references a Project CR to derive the Harbor project ID.
	// +optional
	ProjectRef *ProjectReference `json:"projectRef,omitempty"`

	// ProjectNameOrID is the Harbor project name or numeric ID.
	// +optional
	ProjectNameOrID string `json:"projectNameOrID,omitempty"`

	// Name is the webhook policy name.
	// Defaults to metadata.name when omitted.
	// +optional
	Name string `json:"name,omitempty"`

	// Description is an optional policy description.
	// +optional
	Description string `json:"description,omitempty"`

	// Enabled indicates whether the policy is enabled.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// EventTypes lists the webhook event types.
	// +kubebuilder:validation:MinItems=1
	EventTypes []string `json:"eventTypes"`

	// Targets lists the webhook targets.
	// +kubebuilder:validation:MinItems=1
	Targets []WebhookTargetSpec `json:"targets"`
}

// WebhookPolicyStatus defines the observed state of WebhookPolicy.
type WebhookPolicyStatus struct {
	HarborStatusBase `json:",inline"`

	// HarborWebhookPolicyID is the ID of the webhook policy in Harbor.
	HarborWebhookPolicyID int `json:"harborWebhookPolicyID,omitempty"`

	// TargetsHash is a hash of the configured targets to detect changes.
	// +optional
	TargetsHash string `json:"targetsHash,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Project",type=string,JSONPath=`.spec.projectRef.name`
// +kubebuilder:printcolumn:name="Enabled",type=boolean,JSONPath=`.spec.enabled`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].message`,priority=1
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// WebhookPolicy is the Schema for the webhookpolicies API.
type WebhookPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WebhookPolicySpec   `json:"spec,omitempty"`
	Status WebhookPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// WebhookPolicyList contains a list of WebhookPolicy.
type WebhookPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WebhookPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WebhookPolicy{}, &WebhookPolicyList{})
}
