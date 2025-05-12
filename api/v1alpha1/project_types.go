package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ProjectSpec defines the desired state of Project.
type ProjectSpec struct {
	HarborSpecBase `json:",inline"`

	// Name is the name of the project.
	// It is recommended to leave this field empty so that the operator defaults it
	// to the custom resource’s metadata name.
	// +optional
	Name string `json:"name,omitempty"`

	// Public indicates whether the project is public.
	// +kubebuilder:default:=true
	Public bool `json:"public"`

	// Owner is an optional field for the project owner.
	// +optional
	Owner string `json:"owner,omitempty"`

	// Metadata holds additional configuration for the Harbor project.
	// +optional
	Metadata *ProjectMetadata `json:"metadata,omitempty"`

	// CVEAllowlist holds the configuration for the CVE allowlist.
	// +optional
	CVEAllowlist *CVEAllowlist `json:"cve_allowlist,omitempty"`

	// StorageLimit is the storage limit for the project.
	// +optional
	StorageLimit int `json:"storage_limit,omitempty"`

	// RegistryName is the name of the registry to use for proxy cache projects.
	// The operator will search Harbor for a registry with this name.
	// +optional
	RegistryName string `json:"registryName,omitempty"`
}

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

// ProjectStatus defines the observed state of Project.
type ProjectStatus struct {
	// HarborProjectID is the ID of the project in Harbor.
	HarborProjectID int `json:"harborProjectID,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

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
