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

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
)

// RegistryReconciler reconciles a Registry object.
type RegistryReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	logger logr.Logger
}

// harborRegistryResponse represents a registry as returned by Harbor.
type harborRegistryResponse struct {
	ID          int    `json:"id"`
	URL         string `json:"url"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Insecure    bool   `json:"insecure"`
}

// createRegistryRequest is the payload sent to Harbor when creating or updating a registry.
type createRegistryRequest struct {
	URL         string `json:"url,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type,omitempty"`
	Insecure    bool   `json:"insecure,omitempty"`
}

// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=registries,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=registries/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections,verbs=get;list;watch

// Reconcile is the reconciliation loop for the Registry resource.
func (r *RegistryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.FromContext(ctx).WithName(fmt.Sprintf("[Registry:%s]", req.NamespacedName))

	// Fetch the Registry instance.
	var registry harborv1alpha1.Registry
	if err := r.Get(ctx, req.NamespacedName, &registry); err != nil {
		if errors.IsNotFound(err) {
			r.logger.V(1).Info("Registry resource not found; it may have been deleted")
			return ctrl.Result{}, nil
		}
		r.logger.Error(err, "Failed to get Registry")
		return ctrl.Result{}, err
	}

	// Retrieve the HarborConnection referenced by the Registry.
	harborConn, err := getHarborConnection(ctx, r.Client, registry.Namespace, registry.Spec.HarborConnectionRef)
	if err != nil {
		r.logger.Error(err, "Failed to get HarborConnection", "HarborConnectionRef", registry.Spec.HarborConnectionRef)
		return ctrl.Result{}, err
	}

	// Handle deletion.
	if !registry.GetDeletionTimestamp().IsZero() {
		if controllerutil.ContainsFinalizer(&registry, finalizerName) {
			if err := r.deleteHarborRegistry(ctx, harborConn, &registry); err != nil {
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(&registry, finalizerName)
			if err := r.Update(ctx, &registry); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Ensure the finalizer is present.
	if !controllerutil.ContainsFinalizer(&registry, finalizerName) {
		if controllerutil.AddFinalizer(&registry, finalizerName) {
			if err := r.Update(ctx, &registry); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	if registry.Spec.Name == "" {
		registry.Spec.Name = registry.ObjectMeta.Name
		r.logger.V(1).Info("No name specified; using metadata name", "Name", registry.Spec.Name)
	}

	// Adoption logic: if no HarborRegistryID is set and AllowTakeover is enabled,
	// try to adopt an existing registry by name.
	if registry.Status.HarborRegistryID == 0 && registry.Spec.AllowTakeover {
		adopted, adoptErr := r.adoptExistingRegistry(ctx, harborConn, &registry)
		if adoptErr != nil {
			r.logger.Error(adoptErr, "Failed to adopt existing registry", "RegistryName", registry.Spec.Name)
			return ctrl.Result{}, adoptErr
		}
		if adopted != nil {
			r.logger.Info("Successfully adopted existing registry", "RegistryName", registry.Spec.Name, "HarborRegistryID", adopted.ID)
		}
	}

	// Retrieve the existing registry using HarborRegistryID if available.
	var existing *harborRegistryResponse
	if registry.Status.HarborRegistryID != 0 {
		existing, err = r.getHarborRegistryByID(ctx, harborConn, registry.Status.HarborRegistryID)
		if err != nil {
			r.logger.Error(err, "Failed to get registry by ID from Harbor", "HarborRegistryID", registry.Status.HarborRegistryID)
		}
	}

	// Build the desired registry payload from the CR.
	desired := r.buildRegistryRequest(&registry)

	// If a registry exists and its configuration differs from the desired state, update it.
	if existing != nil {
		if registryNeedsUpdate(desired, *existing) {
			r.logger.Info("Registry in Harbor differs from desired state, updating", "RegistryName", registry.Spec.Name)
			if err := r.updateHarborRegistry(ctx, harborConn, existing.ID, desired); err != nil {
				return ctrl.Result{}, err
			}
			r.logger.Info("Successfully updated registry on Harbor", "RegistryName", registry.Spec.Name)
		} else {
			r.logger.V(1).Info("Registry is already in sync with desired state", "RegistryName", registry.Spec.Name)
		}

		return returnWithDriftDetection(&registry.Spec.HarborSpecBase)
	}

	// If the registry is not found, create a new one.
	if registry.Status.HarborRegistryID != 0 {
		r.logger.Info("Registry with stored ID not found. Assuming it was deleted externally. Creating new registry", "RegistryName", registry.Spec.Name)
	} else {
		r.logger.Info("Creating new registry", "RegistryName", registry.Spec.Name)
	}
	newID, err := r.createHarborRegistry(ctx, harborConn, desired)
	if err != nil {
		return ctrl.Result{}, err
	}
	registry.Status.HarborRegistryID = newID
	if err := r.Status().Update(ctx, &registry); err != nil {
		r.logger.Error(err, "Failed to update Registry status with Harbor registry ID", "HarborRegistryID", newID)
		return ctrl.Result{}, err
	}
	r.logger.Info("Successfully created registry on Harbor", "RegistryName", registry.Spec.Name, "HarborRegistryID", newID)

	return returnWithDriftDetection(&registry.Spec.HarborSpecBase)
}

// adoptExistingRegistry attempts to adopt an existing registry from Harbor by name.
// If a registry is found, it updates the CR's status with the Harbor registry ID.
func (r *RegistryReconciler) adoptExistingRegistry(ctx context.Context, harborConn *harborv1alpha1.HarborConnection, registry *harborv1alpha1.Registry) (*harborRegistryResponse, error) {
	existing, err := r.getHarborRegistry(ctx, harborConn, registry.Spec.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup registry for adoption: %w", err)
	}
	if existing != nil {
		registry.Status.HarborRegistryID = existing.ID
		if err := r.Status().Update(ctx, registry); err != nil {
			return nil, fmt.Errorf("failed to update registry status during adoption: %w", err)
		}
	}
	return existing, nil
}

// buildRegistryRequest constructs the JSON request for the registry creation/update.
func (r *RegistryReconciler) buildRegistryRequest(registry *harborv1alpha1.Registry) createRegistryRequest {
	return createRegistryRequest{
		URL:         registry.Spec.URL,
		Name:        registry.Spec.Name,
		Description: registry.Spec.Description,
		Type:        registry.Spec.Type,
		Insecure:    registry.Spec.Insecure,
	}
}

// createHarborRegistry sends a POST request to Harbor to create a new registry.
func (r *RegistryReconciler) createHarborRegistry(ctx context.Context, harborConn *harborv1alpha1.HarborConnection, payload createRegistryRequest) (int, error) {
	registriesURL := fmt.Sprintf("%s/api/v2.0/registries", harborConn.Spec.BaseURL)
	r.logger.V(1).Info("Sending registry creation request", "url", registriesURL)

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal registry payload: %w", err)
	}

	reqHTTP, err := http.NewRequest("POST", registriesURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return 0, fmt.Errorf("failed to create HTTP request for registry creation: %w", err)
	}
	reqHTTP.Header.Set("Content-Type", "application/json")

	username, password, err := getHarborAuth(ctx, r.Client, harborConn)
	if err != nil {
		return 0, fmt.Errorf("failed to get Harbor auth credentials: %w", err)
	}
	reqHTTP.SetBasicAuth(username, password)

	resp, err := http.DefaultClient.Do(reqHTTP)
	if err != nil {
		return 0, fmt.Errorf("failed to perform HTTP request for registry creation: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("failed to create registry: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Extract the registry ID from the Location header.
	location := resp.Header.Get("location")
	if location == "" {
		return 0, fmt.Errorf("no location header received")
	}
	// Assuming the location header is like "/api/v2.0/registries/1"
	idStr := path.Base(location)
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse registry id from location header %s: %w", location, err)
	}

	return id, nil
}

// updateHarborRegistry sends a PUT request to Harbor to update an existing registry.
func (r *RegistryReconciler) updateHarborRegistry(ctx context.Context, harborConn *harborv1alpha1.HarborConnection, id int, payload createRegistryRequest) error {
	updateURL := fmt.Sprintf("%s/api/v2.0/registries/%d", harborConn.Spec.BaseURL, id)
	r.logger.V(1).Info("Sending registry update request", "url", updateURL)

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal registry payload for update: %w", err)
	}

	reqHTTP, err := http.NewRequest("PUT", updateURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request for registry update: %w", err)
	}
	reqHTTP.Header.Set("Content-Type", "application/json")

	username, password, err := getHarborAuth(ctx, r.Client, harborConn)
	if err != nil {
		return fmt.Errorf("failed to get Harbor auth credentials: %w", err)
	}
	reqHTTP.SetBasicAuth(username, password)

	resp, err := http.DefaultClient.Do(reqHTTP)
	if err != nil {
		return fmt.Errorf("failed to perform HTTP request for registry update: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update registry: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// getHarborRegistry retrieves the registry from Harbor by listing registries and searching by name.
func (r *RegistryReconciler) getHarborRegistry(ctx context.Context, harborConn *harborv1alpha1.HarborConnection, registryName string) (*harborRegistryResponse, error) {
	registriesURL := fmt.Sprintf("%s/api/v2.0/registries", harborConn.Spec.BaseURL)
	reqHTTP, err := http.NewRequest("GET", registriesURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request for registries: %w", err)
	}
	reqHTTP.Header.Set("Content-Type", "application/json")

	username, password, err := getHarborAuth(ctx, r.Client, harborConn)
	if err != nil {
		return nil, fmt.Errorf("failed to get Harbor auth credentials: %w", err)
	}
	reqHTTP.SetBasicAuth(username, password)

	resp, err := http.DefaultClient.Do(reqHTTP)
	if err != nil {
		return nil, fmt.Errorf("failed to perform GET request for registries: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list registries: status %d, body: %s", resp.StatusCode, string(body))
	}

	var registries []harborRegistryResponse
	if err := json.NewDecoder(resp.Body).Decode(&registries); err != nil {
		return nil, fmt.Errorf("failed to decode registries response: %w", err)
	}

	for _, reg := range registries {
		if strings.EqualFold(reg.Name, registryName) {
			return &reg, nil
		}
	}
	return nil, nil
}

// getHarborRegistryByID retrieves the registry from Harbor using its ID.
func (r *RegistryReconciler) getHarborRegistryByID(ctx context.Context, harborConn *harborv1alpha1.HarborConnection, id int) (*harborRegistryResponse, error) {
	getURL := fmt.Sprintf("%s/api/v2.0/registries/%d", harborConn.Spec.BaseURL, id)
	reqHTTP, err := http.NewRequest("GET", getURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request for registry by ID: %w", err)
	}
	reqHTTP.Header.Set("Content-Type", "application/json")

	username, password, err := getHarborAuth(ctx, r.Client, harborConn)
	if err != nil {
		return nil, fmt.Errorf("failed to get Harbor auth credentials: %w", err)
	}
	reqHTTP.SetBasicAuth(username, password)

	resp, err := http.DefaultClient.Do(reqHTTP)
	if err != nil {
		return nil, fmt.Errorf("failed to perform GET request for registry by ID: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get registry by ID: status %d, body: %s", resp.StatusCode, string(body))
	}

	var reg harborRegistryResponse
	if err := json.NewDecoder(resp.Body).Decode(&reg); err != nil {
		return nil, fmt.Errorf("failed to decode registry response by ID: %w", err)
	}

	return &reg, nil
}

// registryNeedsUpdate compares the desired registry configuration with the existing registry.
func registryNeedsUpdate(desired createRegistryRequest, current harborRegistryResponse) bool {
	return desired.URL != current.URL ||
		desired.Name != current.Name ||
		desired.Description != current.Description ||
		!strings.EqualFold(desired.Type, current.Type) ||
		desired.Insecure != current.Insecure
}

// deleteHarborRegistry implements the deletion logic for a registry in Harbor.
func (r *RegistryReconciler) deleteHarborRegistry(ctx context.Context, harborConn *harborv1alpha1.HarborConnection, registry *harborv1alpha1.Registry) error {
	// If no HarborRegistryID is set, there's nothing to delete.
	if registry.Status.HarborRegistryID == 0 {
		r.logger.V(1).Info("No HarborRegistryID present, nothing to delete")
		return nil
	}

	deleteURL := fmt.Sprintf("%s/api/v2.0/registries/%d", harborConn.Spec.BaseURL, registry.Status.HarborRegistryID)
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

	// If the registry is already deleted, log at debug verbosity.
	if resp.StatusCode == http.StatusNotFound {
		r.logger.V(1).Info("Registry not found during deletion; assuming it was already deleted", "HarborRegistryID", registry.Status.HarborRegistryID)
		return nil
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete registry: status %d, body: %s", resp.StatusCode, string(body))
	}

	r.logger.Info("Successfully deleted registry from Harbor", "RegistryName", registry.Spec.Name)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RegistryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&harborv1alpha1.Registry{}).
		Named("registry").
		Complete(r)
}
