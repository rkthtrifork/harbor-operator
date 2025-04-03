package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// UserSpec defines the desired state of User.
type UserSpec struct {
	// HarborConnectionRef references the HarborConnection resource to use.
	// +kubebuilder:validation:Required
	HarborConnectionRef string `json:"harborConnectionRef"`

	// Username is the Harbor username.
	// It is recommended to leave this field empty so that the operator defaults it
	// to the custom resource's metadata name.
	// +optional
	Username string `json:"username,omitempty"`

	// Email is the email address of the user.
	// +kubebuilder:validation:Format=email
	Email string `json:"email"`

	// RealName is an optional field for the user's real name.
	// +optional
	RealName string `json:"realName,omitempty"`

	// Comment is an optional field for additional information.
	// +optional
	Comment string `json:"comment,omitempty"`

	// Password is the user's password.
	// Note: This field is used only on creation. Once a user is created,
	// password changes should be done via a dedicated endpoint.
	// +optional
	Password string `json:"password,omitempty"`

	// SysAdmin indicates whether the user should be a Harbor system administrator.
	// +optional
	SysAdmin bool `json:"sysAdmin,omitempty"`

	// AllowTakeover indicates whether the operator is allowed to adopt an existing user
	// in Harbor if one with the same username already exists.
	// +optional
	AllowTakeover bool `json:"allowTakeover,omitempty"`

	// DriftDetectionInterval is the interval at which the operator will check for drift.
	// A value of 0 (or omitted) disables periodic drift detection.
	// +optional
	DriftDetectionInterval metav1.Duration `json:"driftDetectionInterval,omitempty"`

	// ReconcileNonce is an optional field that, when updated, forces an immediate reconcile.
	// +optional
	ReconcileNonce string `json:"reconcileNonce,omitempty"`
}

// UserStatus defines the observed state of User.
type UserStatus struct {
	// HarborUserID is the ID of the user in Harbor.
	HarborUserID int `json:"harborUserID,omitempty"`
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

// GetDriftDetectionInterval returns the drift detection interval.
func (u *User) GetDriftDetectionInterval() metav1.Duration {
	return u.Spec.DriftDetectionInterval
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
