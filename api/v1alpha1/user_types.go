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
// User - Spec
// -----------------------------------------------------------------------------

// UserSpec defines the desired state of User.
type UserSpec struct {
	HarborSpecBase `json:",inline"`

	// Username in Harbor.
	// If omitted, defaults to `.metadata.name` at reconcile time.
	// +optional
	// +kubebuilder:validation:MinLength=1
	Username string `json:"username,omitempty"`

	// Email address of the user.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Format=email
	Email string `json:"email"`

	// Realname is an optional full name.
	// +optional
	Realname string `json:"realname,omitempty"`

	// Comment is an optional comment for the user.
	// +optional
	Comment string `json:"comment,omitempty"`

	// Password for the user. Used only when the user is created.
	// +optional
	Password string `json:"password,omitempty"`
}

// -----------------------------------------------------------------------------
// User - Status
// -----------------------------------------------------------------------------

// UserStatus defines the observed state of User.
type UserStatus struct {
	// HarborUserID is the numeric user ID in Harbor.
	// +optional
	HarborUserID int `json:"harborUserID,omitempty"`

	// ObservedGeneration is the .metadata.generation last processed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions represent the latest observations of the User’s state.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="Email",type="string",JSONPath=".spec.email"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

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
