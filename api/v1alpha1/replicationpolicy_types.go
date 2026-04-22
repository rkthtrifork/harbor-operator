package v1alpha1

import (
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ReplicationTriggerSettings defines settings for a replication trigger.
type ReplicationTriggerSettings struct {
	// Cron is the cron expression for scheduled triggers.
	// +optional
	Cron string `json:"cron,omitempty"`
}

// ReplicationTriggerSpec defines when the replication policy runs.
type ReplicationTriggerSpec struct {
	// Type defines the trigger type (manual, event_based, scheduled).
	// +kubebuilder:validation:Enum=manual;event_based;scheduled
	// +optional
	Type string `json:"type,omitempty"`

	// Settings holds trigger settings.
	// +optional
	Settings *ReplicationTriggerSettings `json:"settings,omitempty"`
}

// ReplicationFilterSpec defines a replication filter.
type ReplicationFilterSpec struct {
	// Type defines the filter type.
	// +optional
	Type string `json:"type,omitempty"`

	// Value defines the filter value.
	// +optional
	Value apiextensionsv1.JSON `json:"value,omitempty"`

	// Decoration defines how to interpret the filter.
	// +optional
	Decoration string `json:"decoration,omitempty"`
}

// ReplicationPolicySpec defines the desired state of ReplicationPolicy.
// +kubebuilder:validation:XValidation:rule="has(self.sourceRegistryRef)",message="sourceRegistryRef is required"
// +kubebuilder:validation:XValidation:rule="has(self.destinationRegistryRef)",message="destinationRegistryRef is required"
// +kubebuilder:validation:XValidation:rule="!has(self.trigger) || self.trigger.type != 'scheduled' || (has(self.trigger.settings) && has(self.trigger.settings.cron))",message="trigger.settings.cron must be set when trigger.type is scheduled"
type ReplicationPolicySpec struct {
	HarborSpecBase `json:",inline"`

	// AllowTakeover indicates whether the operator is allowed to adopt an
	// existing replication policy in Harbor with the same name.
	// +optional
	AllowTakeover bool `json:"allowTakeover,omitempty"`

	// Description is an optional policy description.
	// +optional
	Description string `json:"description,omitempty"`

	// SourceRegistryRef references a Registry CR to use as the source.
	// +optional
	SourceRegistryRef *RegistryReference `json:"sourceRegistryRef,omitempty"`

	// DestinationRegistryRef references a Registry CR to use as the destination.
	// +optional
	DestinationRegistryRef *RegistryReference `json:"destinationRegistryRef,omitempty"`

	// DestNamespace is the destination namespace.
	// +optional
	DestNamespace string `json:"destNamespace,omitempty"`

	// DestNamespaceReplaceCount controls namespace replacement behavior.
	// +optional
	DestNamespaceReplaceCount *int `json:"destNamespaceReplaceCount,omitempty"`

	// Trigger defines when the replication policy runs.
	// +optional
	Trigger *ReplicationTriggerSpec `json:"trigger,omitempty"`

	// Filters defines the replication filters.
	// +optional
	Filters []ReplicationFilterSpec `json:"filters,omitempty"`

	// ReplicateDeletion indicates whether delete operations are replicated.
	// +optional
	ReplicateDeletion *bool `json:"replicateDeletion,omitempty"`

	// Override indicates whether to overwrite destination resources.
	// +optional
	Override *bool `json:"override,omitempty"`

	// Enabled indicates whether the policy is enabled.
	// +optional
	Enabled *bool `json:"enabled,omitempty"`

	// Speed is the speed limit for each task.
	// +optional
	Speed *int `json:"speed,omitempty"`

	// CopyByChunk indicates whether to enable copy by chunk.
	// +optional
	CopyByChunk *bool `json:"copyByChunk,omitempty"`

	// SingleActiveReplication avoids overlapping executions.
	// +optional
	SingleActiveReplication *bool `json:"singleActiveReplication,omitempty"`
}

// ReplicationPolicyStatus defines the observed state of ReplicationPolicy.
type ReplicationPolicyStatus struct {
	HarborStatusBase `json:",inline"`

	// HarborReplicationPolicyID is the ID of the policy in Harbor.
	HarborReplicationPolicyID int `json:"harborReplicationPolicyID,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=harbor
// +kubebuilder:printcolumn:name="Source",type=string,JSONPath=`.spec.sourceRegistryRef.name`
// +kubebuilder:printcolumn:name="Destination",type=string,JSONPath=`.spec.destinationRegistryRef.name`
// +kubebuilder:printcolumn:name="Enabled",type=boolean,JSONPath=`.spec.enabled`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].message`,priority=1
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ReplicationPolicy is the Schema for the replicationpolicies API.
type ReplicationPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ReplicationPolicySpec   `json:"spec,omitempty"`
	Status ReplicationPolicyStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ReplicationPolicyList contains a list of ReplicationPolicy.
type ReplicationPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ReplicationPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ReplicationPolicy{}, &ReplicationPolicyList{})
}
