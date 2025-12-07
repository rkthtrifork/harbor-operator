package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UserSpec defines the desired state of User.
type UserSpec struct {
	HarborSpecBase `json:",inline"`

	// Username is the Harbor username.
	// It is recommended to leave this field empty so that the operator defaults it
	// to the custom resource's metadata name.
	// +optional
	Username string `json:"username,omitempty"`

	// Email address of the user.
	// +kubebuilder:validation:Format=email
	Email string `json:"email"`

	// Realname is an optional full name.
	// +optional
	Realname string `json:"realname,omitempty"`

	// Comment is an optional comment for the user.
	// +optional
	Comment string `json:"comment,omitempty"`

	// PasswordSecretRef references a secret key that contains the password for the user.
	PasswordSecretRef corev1.SecretKeySelector `json:"passwordSecretRef,omitempty"`
}

// UserStatus defines the observed state of User.
type UserStatus struct {
	// HarborUserID is the ID of the user in Harbor.
	HarborUserID int `json:"harborUserID,omitempty"`

	// Conditions represent the latest available observations of the User's state.
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
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
