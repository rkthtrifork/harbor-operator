package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MemberUser defines a user-based member.
type MemberUser struct {
	// UserRef references the User to grant membership to.
	UserRef UserReference `json:"userRef"`
}

// MemberGroup defines a group-based member.
type MemberGroup struct {
	// GroupRef references the UserGroup to grant membership to.
	GroupRef UserGroupReference `json:"groupRef"`
}

// MemberSpec defines the desired state of Member.
// +kubebuilder:validation:XValidation:rule="has(self.memberUser) != has(self.memberGroup)",message="exactly one of memberUser or memberGroup must be set"
// +kubebuilder:validation:XValidation:rule="!has(self.memberUser) || size(self.memberUser.userRef.name) > 0",message="memberUser.userRef.name must be set when memberUser is provided"
// +kubebuilder:validation:XValidation:rule="!has(self.memberGroup) || size(self.memberGroup.groupRef.name) > 0",message="memberGroup.groupRef.name must be set when memberGroup is provided"
type MemberSpec struct {
	HarborSpecBase `json:",inline"`

	// AllowTakeover indicates whether the operator is allowed to adopt an
	// existing project membership in Harbor for the same identity.
	// +optional
	AllowTakeover bool `json:"allowTakeover,omitempty"`

	// ProjectRef references the project where the member should be added.
	ProjectRef ProjectReference `json:"projectRef"`

	// Role is the human‑readable name of the role.
	// Allowed values: "admin", "maintainer", "developer", "guest"
	// +kubebuilder:validation:Enum=admin;maintainer;developer;guest
	// +kubebuilder:validation:Required
	Role string `json:"role"`

	// MemberUser defines the member if it is a user.
	// +optional
	MemberUser *MemberUser `json:"memberUser,omitempty"`

	// MemberGroup defines the member if it is a group.
	// +optional
	MemberGroup *MemberGroup `json:"memberGroup,omitempty"`
}

// MemberStatus defines the observed state of Member.
type MemberStatus struct {
	HarborStatusBase `json:",inline"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=harbor
// +kubebuilder:printcolumn:name="Project",type=string,JSONPath=`.spec.projectRef.name`
// +kubebuilder:printcolumn:name="User",type=string,JSONPath=`.spec.memberUser.userRef.name`
// +kubebuilder:printcolumn:name="Group",type=string,JSONPath=`.spec.memberGroup.groupRef.name`
// +kubebuilder:printcolumn:name="Role",type=string,JSONPath=`.spec.role`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].message`,priority=1
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

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
