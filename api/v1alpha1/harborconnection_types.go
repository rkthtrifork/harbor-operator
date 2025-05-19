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
// HarborConnection - Spec
// -----------------------------------------------------------------------------

// HarborConnectionSpec defines the desired state of HarborConnection.
type HarborConnectionSpec struct {
	// BaseURL is the Harbor API endpoint, e.g. https://harbor.example.com
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Format=url
	BaseURL string `json:"baseURL"`

	// Credentials holds the default credentials for Harbor API calls.
	// +optional
	Credentials *Credentials `json:"credentials,omitempty"`
}

// Credentials holds default authentication details.
type Credentials struct {
	// Type of the credential.
	// +kubebuilder:validation:Enum=basic
	Type string `json:"type"`

	// AccessKey for authentication (username / token id).
	// +kubebuilder:validation:MinLength=1
	AccessKey string `json:"accessKey"`

	// AccessSecretRef points to the Kubernetes Secret holding the password / token.
	AccessSecretRef SecretReference `json:"accessSecretRef"`
}

// SecretReference is similar to corev1.SecretKeySelector, but supports
// *cross-namespace* references when the Harbor-operator RBAC permits it.
type SecretReference struct {
	// Name of the Secret.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Key inside the Secret data.
	// +optional
	// +kubebuilder:default=access_secret
	Key string `json:"key,omitempty"`

	// Namespace of the Secret. If omitted, the HarborConnection namespace is used.
	// +optional
	Namespace string `json:"namespace,omitempty"`
}

// -----------------------------------------------------------------------------
// HarborConnection - Status
// -----------------------------------------------------------------------------

// HarborConnectionStatus defines the observed state of HarborConnection.
//
// The status fields follow the kstatus conventions
// (Reconciling / Stalled / Ready conditions + observedGeneration).
type HarborConnectionStatus struct {
	// ObservedGeneration is the generation of the spec that has been acted upon.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions represent the latest available observations of an object's state.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:shortName=hc
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="Endpoint",type="string",priority=1,JSONPath=".spec.baseURL"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// HarborConnection is the Schema for the harborconnections API.
type HarborConnection struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HarborConnectionSpec   `json:"spec,omitempty"`
	Status HarborConnectionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// HarborConnectionList contains a list of HarborConnection.
type HarborConnectionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []HarborConnection `json:"items"`
}

func init() {
	SchemeBuilder.Register(&HarborConnection{}, &HarborConnectionList{})
}
