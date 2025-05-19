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
// Registry - Spec
// -----------------------------------------------------------------------------

// RegistrySpec defines the desired state of Registry.
type RegistrySpec struct {
	HarborSpecBase `json:",inline"`

	// Type of the registry, e.g. "github-ghcr".
	// +kubebuilder:validation:Enum=github-ghcr;other-types-if-needed
	Type string `json:"type"`

	// Name to give the registry inside Harbor.
	// If omitted, the operator defaults to `.metadata.name`.
	// +optional
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name,omitempty"`

	// Description is an optional description.
	// +optional
	Description string `json:"description,omitempty"`

	// URL of the remote registry, including scheme.
	// +kubebuilder:validation:Format=url
	URL string `json:"url"`

	// Insecure indicates if the Harbor instance should skip TLS verification
	// when contacting the remote registry.
	// +kubebuilder:default:=false
	Insecure bool `json:"insecure"`
}

// -----------------------------------------------------------------------------
// Registry - Status
// -----------------------------------------------------------------------------

// RegistryStatus defines the observed state of Registry.
type RegistryStatus struct {
	// HarborRegistryID is the numeric ID in Harbor.
	// +optional
	HarborRegistryID int `json:"harborRegistryID,omitempty"`

	// ObservedGeneration is the spec generation last processed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions represent the latest observations of the Registry’s state.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="Type",type="string",JSONPath=".spec.type"
// +kubebuilder:printcolumn:name="URL",type="string",priority=1,JSONPath=".spec.url"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Registry is the Schema for the registries API.
type Registry struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RegistrySpec   `json:"spec,omitempty"`
	Status RegistryStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RegistryList contains a list of Registry.
type RegistryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Registry `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Registry{}, &RegistryList{})
}
