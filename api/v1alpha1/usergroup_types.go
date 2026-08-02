package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// UserGroupClaimSpec describes an external identity that must be registered
// in Harbor. The claim is intentionally non-owning: deleting it never deletes
// the global Harbor UserGroup because that would remove memberships belonging
// to other claims.
// +kubebuilder:validation:XValidation:rule="!has(oldSelf.groupName) || (self.groupName == oldSelf.groupName && self.groupType == oldSelf.groupType && has(self.ldapGroupDN) == has(oldSelf.ldapGroupDN) && (!has(self.ldapGroupDN) || self.ldapGroupDN == oldSelf.ldapGroupDN))",message="group identity fields are immutable"
type UserGroupClaimSpec struct {
	HarborClaimSpecBase `json:",inline"`

	// GroupName is the exact external group name stored in Harbor. For OIDC
	// groups, this is commonly the identity provider's group ID.
	// +kubebuilder:validation:MinLength=1
	GroupName string `json:"groupName"`

	// GroupType is the group type (1=LDAP, 2=HTTP, 3=OIDC).
	// +kubebuilder:validation:Enum=1;2;3
	GroupType int `json:"groupType"`

	// LDAPGroupDN is the DN of the LDAP group when GroupType is LDAP.
	// +optional
	LDAPGroupDN string `json:"ldapGroupDN,omitempty"`
}

// UserGroupClaimStatus defines the observed state of UserGroupClaim.
type UserGroupClaimStatus struct {
	HarborStatusBase `json:",inline"`

	// HarborGroupID is the shared global UserGroup ID in Harbor.
	HarborGroupID int `json:"harborGroupID,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:categories=harbor
// +kubebuilder:printcolumn:name="Group",type=string,JSONPath=`.spec.groupName`
// +kubebuilder:printcolumn:name="Type",type=integer,JSONPath=`.spec.groupType`
// +kubebuilder:printcolumn:name="Ready",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].reason`
// +kubebuilder:printcolumn:name="Message",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].message`,priority=1
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// UserGroupClaim is the Schema for the usergroupclaims API.
type UserGroupClaim struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UserGroupClaimSpec   `json:"spec,omitempty"`
	Status UserGroupClaimStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// UserGroupClaimList contains a list of UserGroupClaim.
type UserGroupClaimList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []UserGroupClaim `json:"items"`
}

func init() {
	SchemeBuilder.Register(&UserGroupClaim{}, &UserGroupClaimList{})
}
