package controller

import (
	"bytes"
	"context"
	"encoding/json"
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

// UserReconciler reconciles a User object.
type UserReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	logger logr.Logger
}

// RBAC permissions.
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=users,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=users/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections,verbs=get;list;watch

// Reconcile implements the reconciliation loop for the User resource.
func (r *UserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.FromContext(ctx).WithName(fmt.Sprintf("[User:%s]", req.NamespacedName))

	// Fetch the User instance.
	var user harborv1alpha1.User
	if err := r.Get(ctx, req.NamespacedName, &user); err != nil {
		if errors.IsNotFound(err) {
			r.logger.Info("User resource not found; it may have been deleted")
			return ctrl.Result{}, nil
		}
		r.logger.Error(err, "Failed to get User")
		return ctrl.Result{}, err
	}

	// Retrieve the HarborConnection referenced by the User.
	harborConn, err := r.getHarborConnection(ctx, user.Namespace, user.Spec.HarborConnectionRef)
	if err != nil {
		r.logger.Error(err, "Failed to get HarborConnection", "HarborConnectionRef", user.Spec.HarborConnectionRef)
		return ctrl.Result{}, err
	}

	// Validate the Harbor BaseURL.
	if err := r.validateBaseURL(harborConn.Spec.BaseURL); err != nil {
		r.logger.Error(err, "Invalid Harbor BaseURL", "BaseURL", harborConn.Spec.BaseURL)
		return ctrl.Result{}, err
	}

	// Build the user creation payload.
	userRequest := r.buildUserRequest(&user)

	// Build the Harbor API URL for creating a user.
	usersURL := fmt.Sprintf("%s/api/v2.0/users", harborConn.Spec.BaseURL)
	r.logger.Info("Sending user creation request", "url", usersURL)

	// Marshal the payload to JSON.
	payloadBytes, err := json.Marshal(userRequest)
	if err != nil {
		r.logger.Error(err, "Failed to marshal user payload")
		return ctrl.Result{}, err
	}

	// Create the HTTP POST request.
	reqHTTP, err := http.NewRequest("POST", usersURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		r.logger.Error(err, "Failed to create HTTP request for user creation")
		return ctrl.Result{}, err
	}
	reqHTTP.Header.Set("Content-Type", "application/json")

	// Set authentication using HarborConnection credentials.
	authUser, authPass, err := r.getHarborAuth(ctx, harborConn)
	if err != nil {
		r.logger.Error(err, "Failed to get Harbor authentication credentials")
		return ctrl.Result{}, err
	}
	reqHTTP.SetBasicAuth(authUser, authPass)

	// Perform the HTTP request.
	resp, err := http.DefaultClient.Do(reqHTTP)
	if err != nil {
		r.logger.Error(err, "Failed to perform HTTP request for user creation")
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
		err := fmt.Errorf("failed to create user: status %d, body: %s", resp.StatusCode, string(body))
		r.logger.Error(err, "Harbor user creation failed")
		return ctrl.Result{}, err
	}

	r.logger.Info("Successfully created user on Harbor", "Username", user.Spec.Username)
	// Optionally update User status here if needed.
	return ctrl.Result{}, nil
}

// getHarborConnection retrieves the HarborConnection referenced in the User.
func (r *UserReconciler) getHarborConnection(ctx context.Context, namespace, name string) (*harborv1alpha1.HarborConnection, error) {
	var harborConn harborv1alpha1.HarborConnection
	if err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &harborConn); err != nil {
		return nil, err
	}
	return &harborConn, nil
}

// validateBaseURL verifies that the provided URL is valid and contains a scheme.
func (r *UserReconciler) validateBaseURL(baseURL string) error {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return err
	}
	if parsedURL.Scheme == "" {
		return fmt.Errorf("baseURL %s is missing a protocol scheme", baseURL)
	}
	return nil
}

// createUserRequest represents the payload sent to Harbor to create a user.
type createUserRequest struct {
	Email    string `json:"email"`
	RealName string `json:"realname"`
	Comment  string `json:"comment,omitempty"`
	Password string `json:"password"`
	Username string `json:"username"`
}

// buildUserRequest constructs the JSON payload for the user creation request.
func (r *UserReconciler) buildUserRequest(user *harborv1alpha1.User) createUserRequest {
	return createUserRequest{
		Email:    user.Spec.Email,
		RealName: user.Spec.RealName,
		Comment:  user.Spec.Comment,
		Password: user.Spec.Password,
		Username: user.Spec.Username,
	}
}

// getHarborAuth retrieves the Harbor authentication credentials from the HarborConnection.
func (r *UserReconciler) getHarborAuth(ctx context.Context, harborConn *harborv1alpha1.HarborConnection) (string, string, error) {
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
func (r *UserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&harborv1alpha1.User{}).
		Named("user").
		Complete(r)
}
