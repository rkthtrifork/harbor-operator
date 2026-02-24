package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// PurgeAuditParameters defines parameters for purge audit schedules.
type PurgeAuditParameters struct {
	// AuditRetentionHour is the retention period in hours.
	// +optional
	AuditRetentionHour int `json:"auditRetentionHour,omitempty"`

	// IncludeEventTypes is a comma-separated list of event types to include.
	// +optional
	IncludeEventTypes string `json:"includeEventTypes,omitempty"`

	// DryRun indicates whether to run in dry-run mode.
	// +optional
	DryRun bool `json:"dryRun,omitempty"`
}

// PurgeAuditScheduleSpec defines the desired schedule for audit purge.
type PurgeAuditScheduleSpec struct {
	HarborSpecBase `json:",inline"`

	// Schedule defines when purge runs.
	Schedule ScheduleSpec `json:"schedule"`

	// Parameters define purge settings.
	// +optional
	Parameters PurgeAuditParameters `json:"parameters,omitempty"`
}

// PurgeAuditScheduleStatus defines the observed state of PurgeAuditSchedule.
type PurgeAuditScheduleStatus struct {
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

// PurgeAuditSchedule is the Schema for the purgeauditschedules API.
type PurgeAuditSchedule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PurgeAuditScheduleSpec   `json:"spec,omitempty"`
	Status PurgeAuditScheduleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// PurgeAuditScheduleList contains a list of PurgeAuditSchedule.
type PurgeAuditScheduleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PurgeAuditSchedule `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PurgeAuditSchedule{}, &PurgeAuditScheduleList{})
}
