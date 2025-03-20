package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

// RBAC permissions.
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=projects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=projects/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections,verbs=get;list;watch

// Reconcile implements the reconciliation loop for the Project resource.
func (r *ProjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.FromContext(ctx).WithName(fmt.Sprintf("[Project:%s]", req.NamespacedName))

	// Fetch the Project instance.
	var project harborv1alpha1.Project
	if err := r.Get(ctx, req.NamespacedName, &project); err != nil {
		if errors.IsNotFound(err) {
			r.logger.Info("Project resource not found; it may have been deleted")
			return ctrl.Result{}, nil
		}
		r.logger.Error(err, "Failed to get Project")
		return ctrl.Result{}, err
	}

	// Retrieve the HarborConnection referenced by the Project.
	harborConn, err := r.getHarborConnection(ctx, project.Namespace, project.Spec.HarborConnectionRef)
	if err != nil {
		r.logger.Error(err, "Failed to get HarborConnection", "HarborConnectionRef", project.Spec.HarborConnectionRef)
		return ctrl.Result{}, err
	}

	// Validate the Harbor BaseURL.
	if err := r.validateBaseURL(harborConn.Spec.BaseURL); err != nil {
		r.logger.Error(err, "Invalid Harbor BaseURL", "BaseURL", harborConn.Spec.BaseURL)
		return ctrl.Result{}, err
	}

	// Lookup the registry's numeric ID from Harbor using the provided registry name.
	registryID, err := r.lookupRegistryID(ctx, harborConn, project.Spec.RegistryName)
	if err != nil {
		// If the registry is not found, log a warning and requeue.
		if strings.Contains(err.Error(), "not found") {
			r.logger.Info("Registry not (yet) available in Harbor; requeuing", "RegistryName", project.Spec.RegistryName)
			// Requeue after a delay to give time for the registry resource to be applied.
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}
		r.logger.Error(err, "Failed to lookup registry by name", "RegistryName", project.Spec.RegistryName)
		return ctrl.Result{}, err
	}

	// Build the project creation payload.
	projectRequest := r.buildProjectRequest(&project, registryID)

	// Build the Harbor API URL for creating a project.
	projectsURL := fmt.Sprintf("%s/api/v2.0/projects", harborConn.Spec.BaseURL)
	r.logger.Info("Sending project creation request", "url", projectsURL)

	// Marshal the payload to JSON.
	payloadBytes, err := json.Marshal(projectRequest)
	if err != nil {
		r.logger.Error(err, "Failed to marshal project payload")
		return ctrl.Result{}, err
	}

	// Create the HTTP POST request.
	reqHTTP, err := http.NewRequest("POST", projectsURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		r.logger.Error(err, "Failed to create HTTP request for project creation")
		return ctrl.Result{}, err
	}
	reqHTTP.Header.Set("Content-Type", "application/json")

	// Set authentication using HarborConnection credentials.
	username, password, err := r.getHarborAuth(ctx, harborConn)
	if err != nil {
		r.logger.Error(err, "Failed to get Harbor authentication credentials")
		return ctrl.Result{}, err
	}
	reqHTTP.SetBasicAuth(username, password)

	// Perform the HTTP request.
	resp, err := http.DefaultClient.Do(reqHTTP)
	if err != nil {
		r.logger.Error(err, "Failed to perform HTTP request for project creation")
		return ctrl.Result{}, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			r.logger.Error(err, "failed to close response body")
		}
	}()

	// Check for a successful status code (e.g., 201 Created).
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("failed to create project: status %d, body: %s", resp.StatusCode, string(body))
		r.logger.Error(err, "Harbor project creation failed")
		return ctrl.Result{}, err
	}

	r.logger.Info("Successfully created project on Harbor", "ProjectName", project.Spec.Name)
	// Optionally update Project status here if needed.
	return ctrl.Result{}, nil
}

// getHarborConnection retrieves the HarborConnection referenced in the Project.
func (r *ProjectReconciler) getHarborConnection(ctx context.Context, namespace, name string) (*harborv1alpha1.HarborConnection, error) {
	var harborConn harborv1alpha1.HarborConnection
	if err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &harborConn); err != nil {
		return nil, err
	}
	return &harborConn, nil
}

// validateBaseURL verifies that the provided URL is valid and contains a scheme.
func (r *ProjectReconciler) validateBaseURL(baseURL string) error {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return err
	}
	if parsedURL.Scheme == "" {
		return fmt.Errorf("baseURL %s is missing a protocol scheme", baseURL)
	}
	return nil
}

// lookupRegistryID uses the Harbor API to find the numeric ID of a registry by its name.
func (r *ProjectReconciler) lookupRegistryID(ctx context.Context, harborConn *harborv1alpha1.HarborConnection, registryName string) (int, error) {
	registriesURL := fmt.Sprintf("%s/api/v2.0/registries", harborConn.Spec.BaseURL)
	reqHTTP, err := http.NewRequest("GET", registriesURL, nil)
	if err != nil {
		return 0, err
	}
	reqHTTP.Header.Set("Content-Type", "application/json")

	// Authenticate using HarborConnection credentials.
	username, password, err := r.getHarborAuth(ctx, harborConn)
	if err != nil {
		return 0, err
	}
	reqHTTP.SetBasicAuth(username, password)

	resp, err := http.DefaultClient.Do(reqHTTP)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("failed to list registries: status %d, body: %s", resp.StatusCode, string(body))
	}

	// The response is expected to be a JSON array of registries.
	var registries []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&registries); err != nil {
		return 0, err
	}

	// Search for the registry by name.
	for _, reg := range registries {
		if reg.Name == registryName {
			return reg.ID, nil
		}
	}
	return 0, fmt.Errorf("registry %s not found", registryName)
}

// createProjectRequest represents the payload sent to Harbor to create a project.
type createProjectRequest struct {
	ProjectName  string            `json:"project_name"`
	Public       bool              `json:"public"`
	Metadata     map[string]string `json:"metadata"`
	CveAllowlist interface{}       `json:"cve_allowlist,omitempty"`
	StorageLimit int64             `json:"storage_limit,omitempty"`
	RegistryID   int               `json:"registry_id,omitempty"`
}

// buildProjectRequest constructs the JSON payload for the project creation request.
func (r *ProjectReconciler) buildProjectRequest(project *harborv1alpha1.Project, registryID int) createProjectRequest {
	return createProjectRequest{
		ProjectName:  project.Spec.Name,
		Public:       project.Spec.Public,
		Metadata:     project.Spec.Metadata,
		CveAllowlist: project.Spec.CveAllowlist, // assuming CRD structure matches Harbor's expectations
		StorageLimit: project.Spec.StorageLimit,
		RegistryID:   registryID,
	}
}

// getHarborAuth retrieves the Harbor authentication credentials from the HarborConnection.
func (r *ProjectReconciler) getHarborAuth(ctx context.Context, harborConn *harborv1alpha1.HarborConnection) (string, string, error) {
	secretKey := types.NamespacedName{
		Namespace: harborConn.Namespace,
		Name:      harborConn.Spec.Credentials.AccessSecretRef,
	}
	var secret corev1.Secret
	if err := r.Get(ctx, secretKey, &secret); err != nil {
		return "", "", err
	}

	accessSecretBytes, ok := secret.Data["access_secret"]
	if !ok {
		return "", "", fmt.Errorf("access_secret not found in secret %s/%s", harborConn.Namespace, harborConn.Spec.Credentials.AccessSecretRef)
	}
	return harborConn.Spec.Credentials.AccessKey, string(accessSecretBytes), nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&harborv1alpha1.Project{}).
		Named("project").
		Complete(r)
}
