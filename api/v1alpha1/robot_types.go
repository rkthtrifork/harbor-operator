package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// RobotAccess defines a single access rule for a robot account.
type RobotAccess struct {
	// Resource defines the resource to grant access to.
	// +kubebuilder:validation:MinLength=1
	Resource string `json:"resource"`

	// Action defines the action to permit.
	// +kubebuilder:validation:MinLength=1
	Action string `json:"action"`

	// Effect defines the effect of the access rule, typically "allow".
	// +optional
	Effect string `json:"effect,omitempty"`
}

// RobotPermission defines a permission block for a robot account.
type RobotPermission struct {
	// Kind defines the permission scope, such as "project" or "system".
	// +kubebuilder:validation:MinLength=1
	Kind string `json:"kind"`

	// Namespace is the Harbor project name for project-scoped permissions.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Access lists the access rules for this permission.
	// +kubebuilder:validation:MinItems=1
	Access []RobotAccess `json:"access"`
}

// RobotSpec defines the desired state of Robot.
type RobotSpec struct {
	HarborSpecBase `json:",inline"`

	// Name is the robot account name (without Harbor's prefix).
	// Defaults to metadata.name when omitted.
	// +optional
	Name string `json:"name,omitempty"`

	// Description of the robot account.
	// +optional
	Description string `json:"description,omitempty"`

	// Level is the scope of the robot account.
	// Allowed values: "system", "project".
	// +kubebuilder:validation:Enum=system;project
	Level string `json:"level"`

	// Permissions define the access granted to the robot account.
	// +kubebuilder:validation:MinItems=1
	Permissions []RobotPermission `json:"permissions"`

	// Disable indicates whether the robot account is disabled.
	// +optional
	Disable *bool `json:"disable,omitempty"`

	// Duration is the token duration in days. Use -1 for never expires.
	// If omitted, it defaults to -1.
	// +kubebuilder:default=-1
	Duration int `json:"duration,omitempty"`

	// SecretRef references the secret key holding the robot secret.
	// The operator writes the generated robot secret to this location.
	// If omitted, the operator will create a Secret named "<metadata.name>-secret"
	// in the same namespace with key "secret".
	// +optional
	SecretRef *SecretReference `json:"secretRef,omitempty"`
}

// RobotStatus defines the observed state of Robot.
type RobotStatus struct {
	// HarborRobotID is the ID of the robot in Harbor.
	HarborRobotID int `json:"harborRobotID,omitempty"`

	// LastRotatedAt is the time when the robot secret was last rotated.
	LastRotatedAt *metav1.Time `json:"lastRotatedAt,omitempty"`

	// ExpiresAt is the expiration time reported by Harbor.
	// +optional
	ExpiresAt *metav1.Time `json:"expiresAt,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Robot is the Schema for the robots API.
type Robot struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RobotSpec   `json:"spec,omitempty"`
	Status RobotStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RobotList contains a list of Robot.
type RobotList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Robot `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Robot{}, &RobotList{})
}
