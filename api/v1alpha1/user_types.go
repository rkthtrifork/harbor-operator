package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UserSpec defines the desired state of User.
type UserSpec struct {
	// HarborConnectionRef references the HarborConnection resource to use.
	// +kubebuilder:validation:Required
	HarborConnectionRef string `json:"harborConnectionRef"`

	// Email is the email address of the user.
	// +kubebuilder:validation:Required
	Email string `json:"email"`

	// RealName is the real name of the user.
	// +kubebuilder:validation:Required
	RealName string `json:"realname"`

	// Comment holds additional information or a comment about the user.
	// +optional
	Comment string `json:"comment,omitempty"`

	// Password is the password for the new user.
	// +kubebuilder:validation:Required
	Password string `json:"password"`

	// Username is the unique username for the user.
	// +kubebuilder:validation:Required
	Username string `json:"username"`
}

// UserStatus defines the observed state of User.
type UserStatus struct {
	// Add any additional status fields if needed.
	// For example, you might add a "Created" flag or record the Harbor user ID.
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// User is the Schema for the users API.
type User struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UserSpec   `json:"spec,omitempty"`
	Status UserStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// UserList contains a list of User.
type UserList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []User `json:"items"`
}

func init() {
	SchemeBuilder.Register(&User{}, &UserList{})
}
