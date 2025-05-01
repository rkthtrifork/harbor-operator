package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
)

// ProjectReconciler reconciles a Project object.
type ProjectReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	logger logr.Logger
}

// harborProjectResponse represents the full project as returned by Harbor.
type harborProjectResponse struct {
	ProjectID          int             `json:"project_id"`
	OwnerID            int             `json:"owner_id"`
	Name               string          `json:"name"`
	RegistryID         int             `json:"registry_id"`
	CreationTime       string          `json:"creation_time"`
	UpdateTime         string          `json:"update_time"`
	Deleted            bool            `json:"deleted"`
	OwnerName          string          `json:"owner_name"`
	Togglable          bool            `json:"togglable"`
	CurrentUserRoleID  int             `json:"current_user_role_id"`
	CurrentUserRoleIDs []int           `json:"current_user_role_ids"`
	RepoCount          int             `json:"repo_count"`
	Metadata           projectMetadata `json:"metadata"`
	CVEAllowlist       cveAllowlist    `json:"cve_allowlist"`
}

// createProjectRequest is the payload sent to Harbor when creating or updating a project.
type createProjectRequest struct {
	ProjectName  string          `json:"project_name,omitempty"`
	Public       bool            `json:"public,omitempty"`
	Owner        string          `json:"owner,omitempty"`
	Metadata     projectMetadata `json:"metadata,omitempty"`
	CVEAllowlist cveAllowlist    `json:"cve_allowlist,omitempty"`
	StorageLimit *int            `json:"storage_limit,omitempty"`
	RegistryID   *int            `json:"registry_id,omitempty"`
}

// projectMetadata mirrors the structure expected by Harbor.
type projectMetadata struct {
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

// cveAllowlistItem mirrors an individual CVE allowlist entry.
type cveAllowlistItem struct {
	CveID string `json:"cve_id,omitempty"`
}

// cveAllowlist mirrors the structure of Harbor’s CVE allowlist.
type cveAllowlist struct {
	ID           int                `json:"id,omitempty"`
	ProjectID    int                `json:"project_id,omitempty"`
	ExpiresAt    int                `json:"expires_at,omitempty"`
	Items        []cveAllowlistItem `json:"items,omitempty"`
	CreationTime string             `json:"creation_time,omitempty"`
	UpdateTime   string             `json:"update_time,omitempty"`
}

// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=projects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=projects/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections,verbs=get;list;watch

// Reconcile is the reconciliation loop for the Project resource.
func (r *ProjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.FromContext(ctx).WithName(fmt.Sprintf("[Project:%s]", req.NamespacedName))

	// Fetch the Project instance.
	var project harborv1alpha1.Project
	if err := r.Get(ctx, req.NamespacedName, &project); err != nil {
		if errors.IsNotFound(err) {
			r.logger.V(1).Info("Project resource not found; it may have been deleted")
			return ctrl.Result{}, nil
		}
		r.logger.Error(err, "Failed to get Project")
		return ctrl.Result{}, err
	}

	// Retrieve the HarborConnection referenced by the Project.
	harborConn, err := getHarborConnection(ctx, r.Client, project.Namespace, project.Spec.HarborConnectionRef)
	if err != nil {
		r.logger.Error(err, "Failed to get HarborConnection", "HarborConnectionRef", project.Spec.HarborConnectionRef)
		return ctrl.Result{}, err
	}

	// Handle deletion.
	if !project.GetDeletionTimestamp().IsZero() {
		if controllerutil.ContainsFinalizer(&project, finalizerName) {
			if err := r.deleteHarborProject(ctx, harborConn, &project); err != nil {
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(&project, finalizerName)
			if err := r.Update(ctx, &project); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Ensure the finalizer is present.
	if !controllerutil.ContainsFinalizer(&project, finalizerName) {
		controllerutil.AddFinalizer(&project, finalizerName)
		if err := r.Update(ctx, &project); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Default project name to metadata name if not specified.
	if project.Spec.Name == "" {
		project.Spec.Name = project.ObjectMeta.Name
		r.logger.V(1).Info("No project name specified; using metadata name", "ProjectName", project.Spec.Name)
	}

	// Adoption logic: if no HarborProjectID is set and AllowTakeover is enabled,
	// try to adopt an existing project by project name.
	if project.Status.HarborProjectID == 0 && project.Spec.AllowTakeover {
		adopted, adoptErr := r.adoptExistingProject(ctx, harborConn, &project)
		if adoptErr != nil {
			r.logger.Error(adoptErr, "Failed to adopt existing project", "ProjectName", project.Spec.Name)
			return ctrl.Result{}, adoptErr
		}
		if adopted != nil {
			r.logger.Info("Successfully adopted existing project", "ProjectName", project.Spec.Name, "HarborProjectID", adopted.ProjectID)
		}
	}

	// Retrieve the existing project using HarborProjectID if available.
	var existing *harborProjectResponse
	if project.Status.HarborProjectID != 0 {
		existing, err = r.getHarborProjectByID(ctx, harborConn, project.Status.HarborProjectID)
		if err != nil {
			r.logger.Error(err, "Failed to get project by ID from Harbor", "HarborProjectID", project.Status.HarborProjectID)
		}
	}

	// Build the desired project payload from the CR.
	desired, err := r.buildProjectRequest(ctx, harborConn, &project)
	if err != nil {
		r.logger.Error(err, "Failed to build project request")
		return ctrl.Result{}, err
	}

	r.logger.Info("Desired project configuration", "DesiredProject", desired)

	// If a project exists and its configuration differs from the desired state, update it.
	if existing != nil {
		if projectNeedsUpdate(desired, *existing) {
			r.logger.Info("Project in Harbor differs from desired state, updating", "ProjectName", project.Spec.Name)
			if err := r.updateHarborProject(ctx, harborConn, project.Status.HarborProjectID, desired); err != nil {
				return ctrl.Result{}, err
			}
			r.logger.Info("Successfully updated project on Harbor", "ProjectName", project.Spec.Name)
		} else {
			r.logger.V(1).Info("Project is already in sync with desired state", "ProjectName", project.Spec.Name)
		}
		return returnWithDriftDetection(&project.Spec.HarborSpecBase)
	}

	// If the project is not found, create a new one.
	if project.Status.HarborProjectID != 0 {
		r.logger.Info("Project with stored ID not found. Assuming it was deleted externally. Creating new project", "ProjectName", project.Spec.Name)
	} else {
		r.logger.Info("Creating new project", "ProjectName", project.Spec.Name)
	}
	newID, err := r.createHarborProject(ctx, harborConn, desired)
	if err != nil {
		return ctrl.Result{}, err
	}
	project.Status.HarborProjectID = newID
	if err := r.Status().Update(ctx, &project); err != nil {
		r.logger.Error(err, "Failed to update Project status with Harbor project ID", "HarborProjectID", newID)
		return ctrl.Result{}, err
	}
	r.logger.Info("Successfully created project on Harbor", "ProjectName", project.Spec.Name, "HarborProjectID", newID)

	return returnWithDriftDetection(&project.Spec.HarborSpecBase)
}

func (r *ProjectReconciler) adoptExistingProject(ctx context.Context, harborConn *harborv1alpha1.HarborConnection, project *harborv1alpha1.Project) (*harborProjectResponse, error) {
	existing, err := r.getHarborProject(ctx, harborConn, project.Spec.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup project for adoption: %w", err)
	}
	if existing != nil {
		project.Status.HarborProjectID = existing.ProjectID
		if err := r.Status().Update(ctx, project); err != nil {
			return nil, fmt.Errorf("failed to update project status during adoption: %w", err)
		}
	}
	return existing, nil
}

// getRegistryByName queries Harbor for a registry with the given name.
func (r *ProjectReconciler) getRegistryByName(ctx context.Context, harborConn *harborv1alpha1.HarborConnection, registryName string) (int, error) {
	// Construct the URL with a query parameter for an exact match.
	url := fmt.Sprintf("%s/api/v2.0/registries?q=name=%s", harborConn.Spec.BaseURL, registryName)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Set("Content-Type", "application/json")

	username, password, err := getHarborAuth(ctx, r.Client, harborConn)
	if err != nil {
		return 0, err
	}
	req.SetBasicAuth(username, password)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("failed to get registries: status %d, body: %s", resp.StatusCode, string(body))
	}

	var registries []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&registries); err != nil {
		return 0, err
	}

	for _, reg := range registries {
		if strings.EqualFold(reg.Name, registryName) {
			return reg.ID, nil
		}
	}

	return 0, fmt.Errorf("registry with name %s not found", registryName)
}

// buildProjectRequest constructs the JSON request for project creation/update.
// It looks up the registry by name if provided.
func (r *ProjectReconciler) buildProjectRequest(ctx context.Context, harborConn *harborv1alpha1.HarborConnection, project *harborv1alpha1.Project) (createProjectRequest, error) {
	var meta projectMetadata
	if project.Spec.Metadata != nil {
		meta = projectMetadata{
			Public:                   project.Spec.Metadata.Public,
			EnableContentTrust:       project.Spec.Metadata.EnableContentTrust,
			EnableContentTrustCosign: project.Spec.Metadata.EnableContentTrustCosign,
			PreventVul:               project.Spec.Metadata.PreventVul,
			Severity:                 project.Spec.Metadata.Severity,
			AutoScan:                 project.Spec.Metadata.AutoScan,
			AutoSBOMGeneration:       project.Spec.Metadata.AutoSBOMGeneration,
			ReuseSysCVEAllowlist:     project.Spec.Metadata.ReuseSysCVEAllowlist,
			RetentionID:              project.Spec.Metadata.RetentionID,
			ProxySpeedKB:             project.Spec.Metadata.ProxySpeedKB,
		}
	}

	var CVEAllowlist cveAllowlist
	if project.Spec.CVEAllowlist != nil {
		CVEAllowlist = cveAllowlist{
			ID:           project.Spec.CVEAllowlist.ID,
			ProjectID:    project.Spec.CVEAllowlist.ProjectID,
			ExpiresAt:    project.Spec.CVEAllowlist.ExpiresAt,
			Items:        make([]cveAllowlistItem, len(project.Spec.CVEAllowlist.Items)),
			CreationTime: project.Spec.CVEAllowlist.CreationTime.Format(time.RFC3339),
			UpdateTime:   project.Spec.CVEAllowlist.UpdateTime.Format(time.RFC3339),
		}
		for i, item := range project.Spec.CVEAllowlist.Items {
			CVEAllowlist.Items[i] = cveAllowlistItem{
				CveID: item.CveID,
			}
		}
	}

	// Convert StorageLimit: if your spec defines 0 as "not set", set to nil.
	var storageLimit *int
	if project.Spec.StorageLimit != 0 {
		storageLimit = &project.Spec.StorageLimit
	}

	// Lookup RegistryID based on the user-supplied RegistryName.
	var registryID *int
	if project.Spec.RegistryName != "" {
		id, err := r.getRegistryByName(ctx, harborConn, project.Spec.RegistryName)
		if err != nil {
			return createProjectRequest{}, err
		}
		registryID = &id
	}

	return createProjectRequest{
		ProjectName:  project.Spec.Name,
		Public:       project.Spec.Public,
		Owner:        project.Spec.Owner,
		Metadata:     meta,
		CVEAllowlist: CVEAllowlist,
		StorageLimit: storageLimit,
		RegistryID:   registryID,
	}, nil
}

// createHarborProject sends a POST request to Harbor to create a new project.
func (r *ProjectReconciler) createHarborProject(ctx context.Context, harborConn *harborv1alpha1.HarborConnection, payload createProjectRequest) (int, error) {
	projectsURL := fmt.Sprintf("%s/api/v2.0/projects", harborConn.Spec.BaseURL)
	r.logger.V(1).Info("Sending project creation request", "url", projectsURL)

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal project payload: %w", err)
	}

	reqHTTP, err := http.NewRequest("POST", projectsURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return 0, fmt.Errorf("failed to create HTTP request for project creation: %w", err)
	}
	reqHTTP.Header.Set("Content-Type", "application/json")

	username, password, err := getHarborAuth(ctx, r.Client, harborConn)
	if err != nil {
		return 0, fmt.Errorf("failed to get Harbor auth credentials: %w", err)
	}
	reqHTTP.SetBasicAuth(username, password)

	resp, err := http.DefaultClient.Do(reqHTTP)
	if err != nil {
		return 0, fmt.Errorf("failed to perform HTTP request for project creation: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("failed to create project: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Extract the project ID from the Location header.
	location := resp.Header.Get("location")
	if location == "" {
		return 0, fmt.Errorf("no location header received")
	}
	// Assuming the location header is like "/api/v2.0/projects/1"
	idStr := path.Base(location)
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse project id from location header %s: %w", location, err)
	}

	return id, nil
}

// updateHarborProject sends a PUT request to Harbor to update an existing project.
func (r *ProjectReconciler) updateHarborProject(ctx context.Context, harborConn *harborv1alpha1.HarborConnection, id int, payload createProjectRequest) error {
	updateURL := fmt.Sprintf("%s/api/v2.0/projects/%d", harborConn.Spec.BaseURL, id)
	r.logger.V(1).Info("Sending project update request", "url", updateURL)

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal project payload for update: %w", err)
	}

	reqHTTP, err := http.NewRequest("PUT", updateURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request for project update: %w", err)
	}
	reqHTTP.Header.Set("Content-Type", "application/json")

	username, password, err := getHarborAuth(ctx, r.Client, harborConn)
	if err != nil {
		return fmt.Errorf("failed to get Harbor auth credentials: %w", err)
	}
	reqHTTP.SetBasicAuth(username, password)

	resp, err := http.DefaultClient.Do(reqHTTP)
	if err != nil {
		return fmt.Errorf("failed to perform HTTP request for project update: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update project: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// getHarborProject retrieves the project from Harbor by listing projects and searching by project name.
func (r *ProjectReconciler) getHarborProject(ctx context.Context, harborConn *harborv1alpha1.HarborConnection, projectName string) (*harborProjectResponse, error) {
	projectsURL := fmt.Sprintf("%s/api/v2.0/projects", harborConn.Spec.BaseURL)
	reqHTTP, err := http.NewRequest("GET", projectsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request for projects: %w", err)
	}
	reqHTTP.Header.Set("Content-Type", "application/json")

	username, password, err := getHarborAuth(ctx, r.Client, harborConn)
	if err != nil {
		return nil, fmt.Errorf("failed to get Harbor auth credentials: %w", err)
	}
	reqHTTP.SetBasicAuth(username, password)

	resp, err := http.DefaultClient.Do(reqHTTP)
	if err != nil {
		return nil, fmt.Errorf("failed to perform GET request for projects: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list projects: status %d, body: %s", resp.StatusCode, string(body))
	}

	var projects []harborProjectResponse
	if err := json.NewDecoder(resp.Body).Decode(&projects); err != nil {
		return nil, fmt.Errorf("failed to decode projects response: %w", err)
	}

	for _, proj := range projects {
		if strings.EqualFold(proj.Name, projectName) {
			return &proj, nil
		}
	}
	return nil, nil
}

// getHarborProjectByID retrieves the project from Harbor using its ID.
func (r *ProjectReconciler) getHarborProjectByID(ctx context.Context, harborConn *harborv1alpha1.HarborConnection, id int) (*harborProjectResponse, error) {
	getURL := fmt.Sprintf("%s/api/v2.0/projects/%d", harborConn.Spec.BaseURL, id)
	reqHTTP, err := http.NewRequest("GET", getURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request for project by ID: %w", err)
	}
	reqHTTP.Header.Set("Content-Type", "application/json")

	username, password, err := getHarborAuth(ctx, r.Client, harborConn)
	if err != nil {
		return nil, fmt.Errorf("failed to get Harbor auth credentials: %w", err)
	}
	reqHTTP.SetBasicAuth(username, password)

	resp, err := http.DefaultClient.Do(reqHTTP)
	if err != nil {
		return nil, fmt.Errorf("failed to perform GET request for project by ID: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get project by ID: status %d, body: %s", resp.StatusCode, string(body))
	}

	var proj harborProjectResponse
	if err := json.NewDecoder(resp.Body).Decode(&proj); err != nil {
		return nil, fmt.Errorf("failed to decode project response by ID: %w", err)
	}

	return &proj, nil
}

// projectNeedsUpdate compares the desired project configuration with the existing project.
func projectNeedsUpdate(desired createProjectRequest, current harborProjectResponse) bool {
	// Compare project name.
	if desired.ProjectName != current.Name {
		return true
	}

	// Compare public flag: desired is boolean, while GET response has it as a string in metadata.
	desiredPublicStr := "false"
	if desired.Public {
		desiredPublicStr = "true"
	}
	if desiredPublicStr != current.Metadata.Public {
		return true
	}

	// Compare owner: desired uses Owner while GET response uses OwnerName.
	if !strings.EqualFold(desired.Owner, current.OwnerName) {
		return true
	}

	// Compare registry ID.
	if desired.RegistryID == nil {
		if current.RegistryID != 0 {
			return true
		}
	} else if *desired.RegistryID != current.RegistryID {
		return true
	}

	// StorageLimit is not included in the GET response so we skip it.

	// Compare metadata fields.
	if desired.Metadata.EnableContentTrust != current.Metadata.EnableContentTrust {
		return true
	}
	if desired.Metadata.EnableContentTrustCosign != current.Metadata.EnableContentTrustCosign {
		return true
	}
	if desired.Metadata.PreventVul != current.Metadata.PreventVul {
		return true
	}
	if desired.Metadata.Severity != current.Metadata.Severity {
		return true
	}
	if desired.Metadata.AutoScan != current.Metadata.AutoScan {
		return true
	}
	if desired.Metadata.AutoSBOMGeneration != current.Metadata.AutoSBOMGeneration {
		return true
	}
	if desired.Metadata.ReuseSysCVEAllowlist != current.Metadata.ReuseSysCVEAllowlist {
		return true
	}
	if desired.Metadata.RetentionID != current.Metadata.RetentionID {
		return true
	}
	if desired.Metadata.ProxySpeedKB != current.Metadata.ProxySpeedKB {
		return true
	}

	// Compare CVE allowlist fields.
	if desired.CVEAllowlist.ID != current.CVEAllowlist.ID {
		return true
	}
	if desired.CVEAllowlist.ProjectID != current.CVEAllowlist.ProjectID {
		return true
	}
	if desired.CVEAllowlist.ExpiresAt != current.CVEAllowlist.ExpiresAt {
		return true
	}
	if desired.CVEAllowlist.CreationTime != current.CVEAllowlist.CreationTime {
		return true
	}
	if desired.CVEAllowlist.UpdateTime != current.CVEAllowlist.UpdateTime {
		return true
	}
	if len(desired.CVEAllowlist.Items) != len(current.CVEAllowlist.Items) {
		return true
	}
	for i, item := range desired.CVEAllowlist.Items {
		if item.CveID != current.CVEAllowlist.Items[i].CveID {
			return true
		}
	}

	return false
}

// deleteHarborProject implements the deletion logic for a project in Harbor.
func (r *ProjectReconciler) deleteHarborProject(ctx context.Context, harborConn *harborv1alpha1.HarborConnection, project *harborv1alpha1.Project) error {
	// If no HarborProjectID is set, there's nothing to delete.
	if project.Status.HarborProjectID == 0 {
		r.logger.V(1).Info("No HarborProjectID present, nothing to delete")
		return nil
	}

	deleteURL := fmt.Sprintf("%s/api/v2.0/projects/%d", harborConn.Spec.BaseURL, project.Status.HarborProjectID)
	reqHTTP, err := http.NewRequest("DELETE", deleteURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create DELETE request: %w", err)
	}
	reqHTTP.Header.Set("Content-Type", "application/json")

	username, password, err := getHarborAuth(ctx, r.Client, harborConn)
	if err != nil {
		return err
	}
	reqHTTP.SetBasicAuth(username, password)

	resp, err := http.DefaultClient.Do(reqHTTP)
	if err != nil {
		return fmt.Errorf("failed to perform DELETE request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// If the project is already deleted, log at debug verbosity.
	if resp.StatusCode == http.StatusNotFound {
		r.logger.V(1).Info("Project not found during deletion; assuming it was already deleted", "HarborProjectID", project.Status.HarborProjectID)
		return nil
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete project: status %d, body: %s", resp.StatusCode, string(body))
	}

	r.logger.Info("Successfully deleted project from Harbor", "ProjectName", project.Spec.Name)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&harborv1alpha1.Project{}).
		Named("project").
		Complete(r)
}
