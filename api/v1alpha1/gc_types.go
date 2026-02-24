package v1alpha1

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GCScheduleSpec defines the desired schedule for garbage collection.
type GCScheduleSpec struct {
	HarborSpecBase `json:",inline"`

	// Schedule defines when GC runs.
	Schedule ScheduleSpec `json:"schedule"`

	// Parameters define GC settings passed to Harbor.
	// +optional
	Parameters map[string]apiextensionsv1.JSON `json:"parameters,omitempty"`
}

// GCScheduleStatus defines the observed state of GCSchedule.
type GCScheduleStatus struct {
	HarborStatusBase `json:",inline"`

	// LastAppliedScheduleHash is the hash of the applied schedule.
	// +optional
	LastAppliedScheduleHash string `json:"lastAppliedScheduleHash,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.schedule.type`
// +kubebuilder:printcolumn:name="Cron",type=string,JSONPath=`.spec.schedule.cron`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Message",type=string,priority=1,JSONPath=`.status.conditions[?(@.type=="Ready")].message`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// GCSchedule is the Schema for the gcschedules API.
type GCSchedule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   GCScheduleSpec   `json:"spec,omitempty"`
	Status GCScheduleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// GCScheduleList contains a list of GCSchedule.
type GCScheduleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []GCSchedule `json:"items"`
}

func init() {
	SchemeBuilder.Register(&GCSchedule{}, &GCScheduleList{})
}
