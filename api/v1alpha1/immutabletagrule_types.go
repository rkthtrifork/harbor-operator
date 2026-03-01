package v1alpha1

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ImmutableSelector defines an immutable tag rule selector.
type ImmutableSelector struct {
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

// ImmutableTagRuleSpec defines the desired state of ImmutableTagRule.
type ImmutableTagRuleSpec struct {
	HarborSpecBase `json:",inline"`

	// AllowTakeover indicates whether the operator is allowed to adopt an
	// existing immutable tag rule in Harbor that matches this spec.
	// +optional
	AllowTakeover bool `json:"allowTakeover,omitempty"`

	// ProjectRef references a Project CR to derive the Harbor project ID.
	// +optional
	ProjectRef *ProjectReference `json:"projectRef,omitempty"`

	// ProjectNameOrID is the Harbor project name or numeric ID.
	// +optional
	ProjectNameOrID string `json:"projectNameOrID,omitempty"`

	// Disabled indicates whether the rule is disabled.
	// +optional
	Disabled bool `json:"disabled,omitempty"`

	// Action defines the rule action.
	// +optional
	Action string `json:"action,omitempty"`

	// Template defines the rule template.
	// +optional
	Template string `json:"template,omitempty"`

	// Params holds template parameters.
	// +optional
	Params map[string]apiextensionsv1.JSON `json:"params,omitempty"`

	// TagSelectors define tag selectors.
	// +optional
	TagSelectors []ImmutableSelector `json:"tagSelectors,omitempty"`

	// ScopeSelectors define scope selectors.
	// +optional
	ScopeSelectors map[string][]ImmutableSelector `json:"scopeSelectors,omitempty"`

	// Priority defines the rule priority.
	// +optional
	Priority int `json:"priority,omitempty"`
}

// ImmutableTagRuleStatus defines the observed state of ImmutableTagRule.
type ImmutableTagRuleStatus struct {
	HarborStatusBase `json:",inline"`

	// HarborImmutableRuleID is the ID of the rule in Harbor.
	HarborImmutableRuleID int `json:"harborImmutableRuleID,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Project",type=string,JSONPath=`.spec.projectRef.name`
// +kubebuilder:printcolumn:name="Disabled",type=boolean,JSONPath=`.spec.disabled`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].message`,priority=1
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ImmutableTagRule is the Schema for the immutabletagrules API.
type ImmutableTagRule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ImmutableTagRuleSpec   `json:"spec,omitempty"`
	Status ImmutableTagRuleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ImmutableTagRuleList contains a list of ImmutableTagRule.
type ImmutableTagRuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ImmutableTagRule `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ImmutableTagRule{}, &ImmutableTagRuleList{})
}
