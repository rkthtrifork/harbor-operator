// Copyright 2025 The Harbor-Operator Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// -----------------------------------------------------------------------------
// Member – Spec
// -----------------------------------------------------------------------------

// MemberSpec defines the desired state of Member.
//
// NOTE: Only status/conditions were added in this commit; the Spec is kept
// exactly as you already had it.
type MemberSpec struct {
	HarborSpecBase `json:",inline"`

	// Project that the member should belong to (Harbor project name, not ID).
	Project string `json:"project"`

	// Kind of member: user, group, robot.
	// +kubebuilder:validation:Enum=user;group;robot
	Kind string `json:"kind"`

	// Name of the user / group / robot account in Harbor.
	Name string `json:"name"`

	// Role to grant.  For users & groups this is the project role;
	// for robots it will be the robot permission template.
	// +kubebuilder:validation:MinLength=1
	Role string `json:"role"`
}

// -----------------------------------------------------------------------------
// Member – Status
// -----------------------------------------------------------------------------

// MemberStatus defines the observed state of Member.
type MemberStatus struct {
	// HarborMemberID is the internal numeric ID for the membership in Harbor.
	// +optional
	HarborMemberID int `json:"harborMemberID,omitempty"`

	// ObservedGeneration is the most recent generation that has been reconciled.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions represent the latest observations of the Member's state.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="Project",type="string",JSONPath=".spec.project"
// +kubebuilder:printcolumn:name="Kind",type="string",JSONPath=".spec.kind"
// +kubebuilder:printcolumn:name="Name",type="string",priority=1,JSONPath=".spec.name"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Member is the Schema for the members API.
type Member struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MemberSpec   `json:"spec,omitempty"`
	Status MemberStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MemberList contains a list of Member.
type MemberList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Member `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Member{}, &MemberList{})
}
