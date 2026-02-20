package v1alpha1

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RetentionPolicySpec defines the desired state of a retention policy.
type RetentionPolicySpec struct {
	HarborSpecBase `json:",inline"`

	// Algorithm defines the retention algorithm, e.g. "or".
	// +optional
	Algorithm string `json:"algorithm,omitempty"`

	// Rules defines the retention rules.
	// +kubebuilder:validation:MinItems=1
	Rules []RetentionRule `json:"rules"`

	// Trigger defines when the retention policy runs.
	// +optional
	Trigger *RetentionTrigger `json:"trigger,omitempty"`

	// Scope defines the policy scope.
	// +optional
	Scope *RetentionScope `json:"scope,omitempty"`
}

// RetentionRule defines a retention rule.
type RetentionRule struct {
	// Disabled indicates whether the rule is disabled.
	// +optional
	Disabled bool `json:"disabled,omitempty"`

	// Action defines the rule action, e.g. "delete".
	// +optional
	Action string `json:"action,omitempty"`

	// Template defines the rule template.
	// +optional
	Template string `json:"template,omitempty"`

	// Params holds template parameters.
	// +optional
	Params map[string]apiextensionsv1.JSON `json:"params,omitempty"`

	// TagSelectors define the tag selectors.
	// +optional
	TagSelectors []RetentionSelector `json:"tagSelectors,omitempty"`

	// ScopeSelectors define the scope selectors.
	// +optional
	ScopeSelectors map[string][]RetentionSelector `json:"scopeSelectors,omitempty"`
}

// RetentionSelector defines a selector.
type RetentionSelector struct {
	// Kind defines selector kind.
	// +optional
	Kind string `json:"kind,omitempty"`

	// Decoration defines selector decoration.
	// +optional
	Decoration string `json:"decoration,omitempty"`

	// Pattern defines selector pattern.
	// +optional
	Pattern string `json:"pattern,omitempty"`

	// Extras defines extra selector details.
	// +optional
	Extras string `json:"extras,omitempty"`
}

// RetentionTrigger defines when a policy runs.
type RetentionTrigger struct {
	// Kind defines trigger kind.
	// +optional
	Kind string `json:"kind,omitempty"`

	// Settings holds trigger settings.
	// +optional
	Settings map[string]apiextensionsv1.JSON `json:"settings,omitempty"`

	// References holds trigger references.
	// +optional
	References map[string]apiextensionsv1.JSON `json:"references,omitempty"`
}

// RetentionScope defines policy scope.
type RetentionScope struct {
	// Level defines scope level, e.g. "project".
	// +optional
	Level string `json:"level,omitempty"`

	// Ref is the scope reference.
	// +optional
	Ref int `json:"ref,omitempty"`
}

// RetentionPolicyStatus defines the observed state of RetentionPolicy.
type RetentionPolicyStatus struct {
	HarborStatusBase `json:",inline"`

	// HarborRetentionID is the ID of the retention policy in Harbor.
	HarborRetentionID int `json:"harborRetentionID,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// RetentionPolicy is the Schema for the retentionpolicies API.
type RetentionPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RetentionPolicySpec   `json:"spec,omitempty"`
	Status RetentionPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RetentionPolicyList contains a list of RetentionPolicy.
type RetentionPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RetentionPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RetentionPolicy{}, &RetentionPolicyList{})
}
