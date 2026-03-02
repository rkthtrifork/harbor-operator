package v1alpha1

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ScanAllScheduleSpec defines the desired schedule for scan all.
type ScanAllScheduleSpec struct {
	HarborSpecBase `json:",inline"`

	// Schedule defines when scan all runs.
	Schedule ScheduleSpec `json:"schedule"`

	// Parameters define scan all settings passed to Harbor.
	// +optional
	Parameters map[string]apiextensionsv1.JSON `json:"parameters,omitempty"`
}

// ScanAllScheduleStatus defines the observed state of ScanAllSchedule.
type ScanAllScheduleStatus struct {
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
// +kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].message`,priority=1
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ScanAllSchedule is the Schema for the scanallschedules API.
type ScanAllSchedule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ScanAllScheduleSpec   `json:"spec,omitempty"`
	Status ScanAllScheduleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ScanAllScheduleList contains a list of ScanAllSchedule.
type ScanAllScheduleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ScanAllSchedule `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ScanAllSchedule{}, &ScanAllScheduleList{})
}
