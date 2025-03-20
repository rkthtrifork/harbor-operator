package controller

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

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

// HarborConnectionReconciler reconciles a HarborConnection object.
type HarborConnectionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	logger logr.Logger
}

// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile fetches the HarborConnection resource, validates the BaseURL, and either performs a non-authenticated
// connectivity check or an authenticated check based on whether credentials are provided.
func (r *HarborConnectionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.FromContext(ctx).WithName(fmt.Sprintf("[HarborConnection:%s]", req.NamespacedName))

	// Fetch the HarborConnection instance.
	var conn harborv1alpha1.HarborConnection
	if err := r.Get(ctx, req.NamespacedName, &conn); err != nil {
		if errors.IsNotFound(err) {
			r.logger.Info("HarborConnection resource not found; it may have been deleted")
			return ctrl.Result{}, nil
		}
		r.logger.Error(err, "Failed to get HarborConnection")
		return ctrl.Result{}, err
	}

	// Validate the BaseURL.
	if err := r.validateBaseURL(ctx, &conn); err != nil {
		return ctrl.Result{}, err
	}

	// If no credentials are provided, perform a non-authenticated connectivity check.
	if conn.Spec.Credentials == nil {
		return r.checkNonAuthConnectivity(ctx, &conn)
	}

	// Otherwise, perform an authenticated check.
	return r.checkAuthenticatedConnection(ctx, &conn)
}

// validateBaseURL verifies that the BaseURL is a valid URL and includes a protocol scheme.
func (r *HarborConnectionReconciler) validateBaseURL(ctx context.Context, conn *harborv1alpha1.HarborConnection) error {
	parsedURL, err := url.Parse(conn.Spec.BaseURL)
	if err != nil {
		r.logger.Error(err, "Invalid baseURL format")
		return err
	}
	if parsedURL.Scheme == "" {
		err := fmt.Errorf("baseURL %s is missing a protocol scheme", conn.Spec.BaseURL)
		r.logger.Error(err, "Invalid baseURL")
		return err
	}
	return nil
}

// checkNonAuthConnectivity performs a connectivity check using a non-authorized endpoint.
func (r *HarborConnectionReconciler) checkNonAuthConnectivity(ctx context.Context, conn *harborv1alpha1.HarborConnection) (ctrl.Result, error) {
	pingURL := fmt.Sprintf("%s/api/v2.0/ping", conn.Spec.BaseURL)
	r.logger.Info("No credentials provided; checking connectivity using non-authorized endpoint", "url", pingURL)

	req, err := http.NewRequest("GET", pingURL, nil)
	if err != nil {
		r.logger.Error(err, "Failed to create HTTP request for connectivity check")
		return ctrl.Result{}, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		r.logger.Error(err, "Failed to perform connectivity check on Harbor API")
		return ctrl.Result{}, err
	}
	defer resp.Body.Close()

	// Both 200 (OK) and 401 (Unauthorized) indicate connectivity.
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusUnauthorized {
		body, _ := io.ReadAll(resp.Body)
		err = fmt.Errorf("connectivity check failed: unexpected status code %d, body: %s", resp.StatusCode, string(body))
		r.logger.Error(err, "Harbor connectivity check failed")
		return ctrl.Result{}, err
	}

	r.logger.Info("Successfully checked connectivity on Harbor API using non-authorized endpoint")
	return ctrl.Result{}, nil
}

// checkAuthenticatedConnection verifies Harbor API credentials by fetching the secret and authenticating via the /users/current endpoint.
func (r *HarborConnectionReconciler) checkAuthenticatedConnection(ctx context.Context, conn *harborv1alpha1.HarborConnection) (ctrl.Result, error) {
	username := conn.Spec.Credentials.AccessKey

	// Retrieve the secret containing the access secret.
	secret, err := r.getSecret(ctx, conn)
	if err != nil {
		r.logger.Error(err, "Failed to get secret")
		return ctrl.Result{}, err
	}

	accessSecretBytes, ok := secret.Data["access_secret"]
	if !ok {
		err = fmt.Errorf("access_secret not found in secret %s/%s", conn.Namespace, conn.Spec.Credentials.AccessSecretRef)
		r.logger.Error(err, "Secret data missing access_secret")
		return ctrl.Result{}, err
	}
	password := string(accessSecretBytes)

	// Build the Harbor API URL for verifying credentials via /users/current.
	authURL := fmt.Sprintf("%s/api/v2.0/users/current", conn.Spec.BaseURL)
	r.logger.Info("Verifying Harbor API credentials", "url", authURL)

	authReq, err := http.NewRequest("GET", authURL, nil)
	if err != nil {
		r.logger.Error(err, "Failed to create HTTP request for credential check")
		return ctrl.Result{}, err
	}
	authReq.SetBasicAuth(username, password)

	resp, err := http.DefaultClient.Do(authReq)
	if err != nil {
		r.logger.Error(err, "Failed to perform HTTP request for credential check")
		return ctrl.Result{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		err = fmt.Errorf("harbor API credential check failed with status %d, body: %s", resp.StatusCode, string(body))
		r.logger.Error(err, "Harbor API authentication failed")
		return ctrl.Result{}, err
	}

	r.logger.Info("Successfully authenticated with Harbor API")
	return ctrl.Result{}, nil
}

// getSecret fetches the secret specified in the HarborConnection credentials.
func (r *HarborConnectionReconciler) getSecret(ctx context.Context, conn *harborv1alpha1.HarborConnection) (*corev1.Secret, error) {
	secretKey := types.NamespacedName{
		Namespace: conn.Namespace,
		Name:      conn.Spec.Credentials.AccessSecretRef,
	}
	var secret corev1.Secret
	if err := r.Get(ctx, secretKey, &secret); err != nil {
		return nil, err
	}
	return &secret, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *HarborConnectionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&harborv1alpha1.HarborConnection{}).
		Named("harborconnection").
		Complete(r)
}
