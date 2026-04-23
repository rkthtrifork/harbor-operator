package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// UserGroupSpec defines the desired state of UserGroup.
type UserGroupSpec struct {
	HarborSpecBase `json:",inline"`

	// AllowTakeover indicates whether the operator is allowed to adopt an
	// existing user group in Harbor with the same name.
	// +optional
	AllowTakeover bool `json:"allowTakeover,omitempty"`

	// GroupType is the group type (1=LDAP, 2=HTTP, 3=OIDC).
	// +kubebuilder:validation:Enum=1;2;3
	GroupType int `json:"groupType"`

	// LDAPGroupDN is the DN of the LDAP group when GroupType is LDAP.
	// +optional
	LDAPGroupDN string `json:"ldapGroupDN,omitempty"`
}

// UserGroupStatus defines the observed state of UserGroup.
type UserGroupStatus struct {
	HarborStatusBase `json:",inline"`

	// HarborGroupID is the ID of the user group in Harbor.
	HarborGroupID int `json:"harborGroupID,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=harbor
// +kubebuilder:printcolumn:name="Type",type=integer,JSONPath=`.spec.groupType`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].message`,priority=1
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// UserGroup is the Schema for the usergroups API.
type UserGroup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UserGroupSpec   `json:"spec,omitempty"`
	Status UserGroupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// UserGroupList contains a list of UserGroup.
type UserGroupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []UserGroup `json:"items"`
}

func init() {
	SchemeBuilder.Register(&UserGroup{}, &UserGroupList{})
}
