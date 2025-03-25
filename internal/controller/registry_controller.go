package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
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
			r.logger.Info("Registry resource not found; it may have been deleted")
			return ctrl.Result{}, nil
		}
		r.logger.Error(err, "Failed to get Registry")
		return ctrl.Result{}, err
	}

	// Check if the resource is being deleted.
	if !registry.GetDeletionTimestamp().IsZero() {
		// Only perform deletion handling if the finalizer is still present.
		if controllerutil.ContainsFinalizer(&registry, finalizerName) {
			// Call your deletion cleanup function (e.g., to delete the Harbor resource).
			if err := r.deleteHarborRegistry(&registry); err != nil {
				return ctrl.Result{}, err
			}
			// Remove the finalizer.
			controllerutil.RemoveFinalizer(&registry, finalizerName)
			if err := r.Update(ctx, &registry); err != nil {
				return ctrl.Result{}, err
			}
		}
		// Since the resource is being deleted, no further reconciliation is needed.
		return ctrl.Result{}, nil
	}

	// If not being deleted, ensure the finalizer is present.
	if !controllerutil.ContainsFinalizer(&registry, finalizerName) {
		if controllerutil.AddFinalizer(&registry, finalizerName) {
			if err := r.Update(ctx, &registry); err != nil {
				return ctrl.Result{}, err
			}
		}
	}

	// Retrieve the HarborConnection referenced by the Registry.
	harborConn, err := r.getHarborConnection(ctx, registry.Namespace, registry.Spec.HarborConnectionRef)
	if err != nil {
		r.logger.Error(err, "Failed to get HarborConnection", "HarborConnectionRef", registry.Spec.HarborConnectionRef)
		return ctrl.Result{}, err
	}

	// Build the registry creation payload.
	registryRequest := r.buildRegistryRequest(&registry)

	// Build the Harbor API URL for creating a registry.
	registriesURL := fmt.Sprintf("%s/api/v2.0/registries", harborConn.Spec.BaseURL)
	r.logger.Info("Sending registry creation request", "url", registriesURL)

	// Marshal the payload to JSON.
	payloadBytes, err := json.Marshal(registryRequest)
	if err != nil {
		r.logger.Error(err, "Failed to marshal registry payload")
		return ctrl.Result{}, err
	}

	// Create the HTTP POST request.
	reqHTTP, err := http.NewRequest("POST", registriesURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		r.logger.Error(err, "Failed to create HTTP request for registry creation")
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
		r.logger.Error(err, "Failed to perform HTTP request for registry creation")
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
		err := fmt.Errorf("failed to create registry: status %d, body: %s", resp.StatusCode, string(body))
		r.logger.Error(err, "Harbor registry creation failed")
		return ctrl.Result{}, err
	}

	r.logger.Info("Successfully created registry on Harbor", "RegistryName", registry.Spec.Name)
	// Optionally update Registry status here if needed.
	return ctrl.Result{}, nil
}

type createRegistryRequest struct {
	URL         string `json:"url"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Insecure    bool   `json:"insecure"`
}

// buildRegistryRequest constructs the JSON request for the registry creation request.
func (r *RegistryReconciler) buildRegistryRequest(registry *harborv1alpha1.Registry) createRegistryRequest {
	return createRegistryRequest{
		URL:         registry.Spec.URL,
		Name:        registry.Spec.Name,
		Description: registry.Spec.Description,
		Type:        registry.Spec.Type,
		Insecure:    registry.Spec.Insecure,
	}
}

// getHarborAuth returns the username and password for authenticating to Harbor.
func (r *RegistryReconciler) getHarborAuth(ctx context.Context, harborConn *harborv1alpha1.HarborConnection) (string, string, error) {
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

func (r *RegistryReconciler) deleteHarborRegistry(registry *harborv1alpha1.Registry) error {
	// Implement the deletion logic here.
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RegistryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&harborv1alpha1.Registry{}).
		Named("registry").
		Complete(r)
}
