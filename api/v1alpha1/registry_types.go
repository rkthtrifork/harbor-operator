/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// RegistrySpec defines the desired state of Registry.
type RegistrySpec struct {
	// HarborConnectionRef references the HarborConnection resource to use.
	// +kubebuilder:validation:Required
	HarborConnectionRef ObjectRef `json:"harborConnectionRef"`

	// Type of the registry, e.g., "github-ghcr"
	// +kubebuilder:validation:Enum=github-ghcr;other-types-if-needed
	Type string `json:"type"`

	// Name is the registry name.
	// +kubebuilder:validation:MinLength=1
	Name string `json:"name"`

	// Description is an optional description.
	// +optional
	Description string `json:"description,omitempty"`

	// URL is the registry URL.
	// +kubebuilder:validation:Format=url
	URL string `json:"url"`

	// VerifyRemoteCert indicates if remote certificates should be verified.
	VerifyRemoteCert bool `json:"verify_remote_cert"`

	// Credential holds the authentication details.
	// +optional
	Credential *RegistryCredential `json:"credential"`
}

// RegistryCredential holds the credential information.
type RegistryCredential struct {
	// Type of credential, e.g., "basic"
	// +kubebuilder:validation:Enum=basic;other-credential-types-if-needed
	Type string `json:"type"`

	// AccessKey is the username or access key.
	// +kubebuilder:validation:MinLength=1
	AccessKey string `json:"access_key"`

	// AccessSecretRef is a reference to a Kubernetes Secret containing the access secret.
	AccessSecretRef ObjectRef `json:"accessSecretRef"`
}

// RegistryStatus defines the observed state of Registry.
type RegistryStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

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
