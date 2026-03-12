package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// QuotaSpec defines the desired state of Quota.
type QuotaSpec struct {
	HarborSpecBase `json:",inline"`

	// ProjectRef references a Project CR to derive the Harbor project ID.
	// +optional
	ProjectRef *ProjectReference `json:"projectRef,omitempty"`

	// ProjectNameOrID is the Harbor project name or numeric ID.
	// +optional
	ProjectNameOrID string `json:"projectNameOrID,omitempty"`

	// Hard defines the quota hard limits (resource name -> limit).
	// +optional
	Hard map[string]int64 `json:"hard,omitempty"`
}

// QuotaStatus defines the observed state of Quota.
type QuotaStatus struct {
	HarborStatusBase `json:",inline"`

	// HarborQuotaID is the ID of the quota in Harbor.
	HarborQuotaID int `json:"harborQuotaID,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=harbor
// +kubebuilder:printcolumn:name="Project",type=string,JSONPath=`.spec.projectRef.name`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].message`,priority=1
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// Quota is the Schema for the quotas API.
type Quota struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   QuotaSpec   `json:"spec,omitempty"`
	Status QuotaStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// QuotaList contains a list of Quota.
type QuotaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Quota `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Quota{}, &QuotaList{})
}
