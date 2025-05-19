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
// Project - Spec
// -----------------------------------------------------------------------------

// ProjectSpec defines the desired state of Project.
type ProjectSpec struct {
	HarborSpecBase `json:",inline"`

	// Name of the project in Harbor.
	// If omitted, the operator will default to `.metadata.name` when reconciling.
	// +optional
	Name string `json:"name,omitempty"`

	// Public indicates whether the project is public.
	// +kubebuilder:default:=true
	Public bool `json:"public"`

	// Owner is the Harbor username that will be set as project owner.
	// +optional
	Owner string `json:"owner,omitempty"`

	// Metadata holds additional configuration for the Harbor project.
	// +optional
	Metadata *ProjectMetadata `json:"metadata,omitempty"`

	// CVEAllowlist holds the configuration for the CVE allowlist.
	// +optional
	CVEAllowlist *CVEAllowlist `json:"cveAllowlist,omitempty"`

	// StorageLimit in bytes.  nil means no limit.
	// +optional
	// +kubebuilder:validation:Minimum=1
	StorageLimit *int64 `json:"storageLimit,omitempty"`

	// RegistryName is the name of the proxy-cache registry to link with.
	// +optional
	RegistryName string `json:"registryName,omitempty"`
}

// -----------------------------------------------------------------------------
// Project - Sub-structs
// -----------------------------------------------------------------------------

// ProjectMetadata defines additional metadata for the project.
type ProjectMetadata struct {
	Public                   string `json:"public,omitempty"`
	EnableContentTrust       string `json:"enable_content_trust,omitempty"`
	EnableContentTrustCosign string `json:"enable_content_trust_cosign,omitempty"`
	PreventVul               string `json:"prevent_vul,omitempty"`
	Severity                 string `json:"severity,omitempty"`
	AutoScan                 string `json:"auto_scan,omitempty"`
	AutoSBOMGeneration       string `json:"auto_sbom_generation,omitempty"`
	ReuseSysCVEAllowlist     string `json:"reuse_sys_cve_allowlist,omitempty"`
	RetentionID              string `json:"retention_id,omitempty"`
	ProxySpeedKB             string `json:"proxy_speed_kb,omitempty"`
}

// CVEAllowlistItem defines a single CVE allowlist entry.
type CVEAllowlistItem struct {
	CveID string `json:"cve_id"`
}

// CVEAllowlist defines the CVE allowlist configuration.
type CVEAllowlist struct {
	ID           int                `json:"id,omitempty"`
	ProjectID    int                `json:"project_id,omitempty"`
	ExpiresAt    int                `json:"expires_at,omitempty"`
	Items        []CVEAllowlistItem `json:"items,omitempty"`
	CreationTime metav1.Time        `json:"creation_time,omitempty"`
	UpdateTime   metav1.Time        `json:"update_time,omitempty"`
}

// -----------------------------------------------------------------------------
// Project - Status
// -----------------------------------------------------------------------------

// ProjectStatus defines the observed state of Project.
type ProjectStatus struct {
	// HarborProjectID is the numeric ID of the project in Harbor.
	// +optional
	HarborProjectID int `json:"harborProjectID,omitempty"`

	// ObservedGeneration is the .metadata.generation last processed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions represent the latest available observations of a Project's state.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status"
// +kubebuilder:printcolumn:name="Public",type="boolean",JSONPath=".spec.public"
// +kubebuilder:printcolumn:name="Owner",type="string",priority=1,JSONPath=".spec.owner"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Project is the Schema for the projects API.
type Project struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ProjectSpec   `json:"spec,omitempty"`
	Status ProjectStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ProjectList contains a list of Project.
type ProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Project `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Project{}, &ProjectList{})
}
