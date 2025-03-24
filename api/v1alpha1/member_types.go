package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MemberUser defines a user-based member.
type MemberUser struct {
	// If the user already exists in Harbor, set UserID.
	// +optional
	UserID int `json:"user_id,omitempty"`
	// Username is used to onboard a user if not already present.
	// +optional
	Username string `json:"username,omitempty"`
}

// MemberGroup defines a group-based member.
type MemberGroup struct {
	// If the group already exists in Harbor, set its ID.
	// +optional
	ID int `json:"id,omitempty"`
	// GroupName is the name of the group.
	// +optional
	GroupName string `json:"group_name,omitempty"`
	// GroupType is the type of the group.
	// +optional
	GroupType int `json:"group_type,omitempty"`
	// LDAPGroupDN is used for LDAP groups.
	// +optional
	LDAPGroupDN string `json:"ldap_group_dn,omitempty"`
}

// MemberSpec defines the desired state of Member.
type MemberSpec struct {
	// HarborConnectionRef references the HarborConnection resource to use.
	// +kubebuilder:validation:Required
	HarborConnectionRef string `json:"harborConnectionRef"`

	// ProjectRef is the name (or ID) of the project in Harbor where the member should be added.
	// +kubebuilder:validation:Required
	ProjectRef string `json:"projectRef"`

	// Role is the human‑readable name of the role.
	// Allowed values: "admin", "maintainer", "developer", "guest"
	// +kubebuilder:validation:Required
	Role string `json:"role"`

	// MemberUser defines the member if it is a user.
	// +optional
	MemberUser *MemberUser `json:"member_user,omitempty"`

	// MemberGroup defines the member if it is a group.
	// +optional
	MemberGroup *MemberGroup `json:"member_group,omitempty"`
}

// MemberStatus defines the observed state of Member.
type MemberStatus struct {
	// Optionally add status fields, e.g. to track creation state or Harbor member ID.
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

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
