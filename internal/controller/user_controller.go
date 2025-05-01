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

// UserReconciler reconciles a User object.
type UserReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	logger logr.Logger
}

// harborUserResponse represents a user as returned by Harbor.
type harborUserResponse struct {
	Email           string `json:"email"`
	Realname        string `json:"realname"`
	Comment         string `json:"comment"`
	UserID          int    `json:"user_id"`
	Username        string `json:"username"`
	SysadminFlag    bool   `json:"sysadmin_flag"`
	AdminRoleInAuth bool   `json:"admin_role_in_auth"`
	CreationTime    string `json:"creation_time"`
	UpdateTime      string `json:"update_time"`
}

// createUserRequest is the payload sent to Harbor when creating a user.
type createUserRequest struct {
	Email    string `json:"email,omitempty"`
	Realname string `json:"realname,omitempty"`
	Comment  string `json:"comment,omitempty"`
	Password string `json:"password,omitempty"`
	Username string `json:"username,omitempty"`
}

// updateUserRequest is the payload sent to Harbor when updating a user.
type updateUserRequest struct {
	Email    string `json:"email,omitempty"`
	Realname string `json:"realname,omitempty"`
	Comment  string `json:"comment,omitempty"`
}

// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=users,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=users/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections,verbs=get;list;watch

// Reconcile is the reconciliation loop for the User resource.
func (r *UserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.FromContext(ctx).WithName(fmt.Sprintf("[User:%s]", req.NamespacedName))

	// Fetch the User instance.
	var user harborv1alpha1.User
	if err := r.Get(ctx, req.NamespacedName, &user); err != nil {
		if errors.IsNotFound(err) {
			r.logger.V(1).Info("User resource not found; it may have been deleted")
			return ctrl.Result{}, nil
		}
		r.logger.Error(err, "Failed to get User")
		return ctrl.Result{}, err
	}

	// Retrieve the HarborConnection referenced by the User.
	harborConn, err := getHarborConnection(ctx, r.Client, user.Namespace, user.Spec.HarborConnectionRef)
	if err != nil {
		r.logger.Error(err, "Failed to get HarborConnection", "HarborConnectionRef", user.Spec.HarborConnectionRef)
		return ctrl.Result{}, err
	}

	// Handle deletion.
	if !user.GetDeletionTimestamp().IsZero() {
		if controllerutil.ContainsFinalizer(&user, finalizerName) {
			if err := r.deleteHarborUser(ctx, harborConn, &user); err != nil {
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(&user, finalizerName)
			if err := r.Update(ctx, &user); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Ensure the finalizer is present.
	if !controllerutil.ContainsFinalizer(&user, finalizerName) {
		controllerutil.AddFinalizer(&user, finalizerName)
		if err := r.Update(ctx, &user); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Default username to metadata name if not specified.
	if user.Spec.Username == "" {
		user.Spec.Username = user.ObjectMeta.Name
		r.logger.V(1).Info("No username specified; using metadata name", "Username", user.Spec.Username)
	}

	// Adoption logic: if no HarborUserID is set and AllowTakeover is enabled,
	// try to adopt an existing user by username.
	if user.Status.HarborUserID == 0 && user.Spec.AllowTakeover {
		adopted, adoptErr := r.adoptExistingUser(ctx, harborConn, &user)
		if adoptErr != nil {
			r.logger.Error(adoptErr, "Failed to adopt existing user", "Username", user.Spec.Username)
			return ctrl.Result{}, adoptErr
		}
		if adopted != nil {
			r.logger.Info("Successfully adopted existing user", "Username", user.Spec.Username, "HarborUserID", adopted.UserID)
		}
	}

	// Retrieve the existing user using HarborUserID if available.
	var existing *harborUserResponse
	if user.Status.HarborUserID != 0 {
		existing, err = r.getHarborUserByID(ctx, harborConn, user.Status.HarborUserID)
		if err != nil {
			r.logger.Error(err, "Failed to get user by ID from Harbor", "HarborUserID", user.Status.HarborUserID)
		}
	}

	// Build the desired user payload from the CR.
	desiredCreate := r.buildCreateUserRequest(&user)
	desiredUpdate := r.buildUpdateUserRequest(&user)

	// If a user exists and its configuration differs from the desired state, update it.
	if existing != nil {
		if userNeedsUpdate(desiredUpdate, *existing) {
			r.logger.Info("User in Harbor differs from desired state, updating", "Username", user.Spec.Username)
			if err := r.updateHarborUser(ctx, harborConn, existing.UserID, desiredUpdate); err != nil {
				return ctrl.Result{}, err
			}
			r.logger.Info("Successfully updated user on Harbor", "Username", user.Spec.Username)
		} else {
			r.logger.V(1).Info("User is already in sync with desired state", "Username", user.Spec.Username)
		}
		return returnWithDriftDetection(&user.Spec.HarborSpecBase)
	}

	// If the user is not found, create a new one.
	if user.Status.HarborUserID != 0 {
		r.logger.Info("User with stored ID not found. Assuming it was deleted externally. Creating new user", "Username", user.Spec.Username)
	} else {
		r.logger.Info("Creating new user", "Username", user.Spec.Username)
	}
	newID, err := r.createHarborUser(ctx, harborConn, desiredCreate)
	if err != nil {
		return ctrl.Result{}, err
	}
	user.Status.HarborUserID = newID
	if err := r.Status().Update(ctx, &user); err != nil {
		r.logger.Error(err, "Failed to update User status with Harbor user ID", "HarborUserID", newID)
		return ctrl.Result{}, err
	}
	r.logger.Info("Successfully created user on Harbor", "Username", user.Spec.Username, "HarborUserID", newID)

	return returnWithDriftDetection(&user.Spec.HarborSpecBase)
}

// adoptExistingUser attempts to adopt an existing user from Harbor by username.
// If a user is found, it updates the CR's status with the Harbor user ID.
func (r *UserReconciler) adoptExistingUser(ctx context.Context, harborConn *harborv1alpha1.HarborConnection, user *harborv1alpha1.User) (*harborUserResponse, error) {
	existing, err := r.getHarborUser(ctx, harborConn, user.Spec.Username)
	if err != nil {
		return nil, fmt.Errorf("failed to lookup user for adoption: %w", err)
	}
	if existing != nil {
		user.Status.HarborUserID = existing.UserID
		if err := r.Status().Update(ctx, user); err != nil {
			return nil, fmt.Errorf("failed to update user status during adoption: %w", err)
		}
	}
	return existing, nil
}

// buildCreateUserRequest constructs the JSON request for user creation.
func (r *UserReconciler) buildCreateUserRequest(user *harborv1alpha1.User) createUserRequest {
	return createUserRequest{
		Email:    user.Spec.Email,
		Realname: user.Spec.Realname,
		Comment:  user.Spec.Comment,
		Password: user.Spec.Password,
		Username: user.Spec.Username,
	}
}

// buildUpdateUserRequest constructs the JSON request for user update.
func (r *UserReconciler) buildUpdateUserRequest(user *harborv1alpha1.User) updateUserRequest {
	return updateUserRequest{
		Email:    user.Spec.Email,
		Realname: user.Spec.Realname,
		Comment:  user.Spec.Comment,
	}
}

// createHarborUser sends a POST request to Harbor to create a new user.
func (r *UserReconciler) createHarborUser(ctx context.Context, harborConn *harborv1alpha1.HarborConnection, payload createUserRequest) (int, error) {
	usersURL := fmt.Sprintf("%s/api/v2.0/users", harborConn.Spec.BaseURL)
	r.logger.V(1).Info("Sending user creation request", "url", usersURL)

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("failed to marshal user payload: %w", err)
	}

	reqHTTP, err := http.NewRequest("POST", usersURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return 0, fmt.Errorf("failed to create HTTP request for user creation: %w", err)
	}
	reqHTTP.Header.Set("Content-Type", "application/json")

	username, password, err := getHarborAuth(ctx, r.Client, harborConn)
	if err != nil {
		return 0, fmt.Errorf("failed to get Harbor auth credentials: %w", err)
	}
	reqHTTP.SetBasicAuth(username, password)

	resp, err := http.DefaultClient.Do(reqHTTP)
	if err != nil {
		return 0, fmt.Errorf("failed to perform HTTP request for user creation: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("failed to create user: status %d, body: %s", resp.StatusCode, string(body))
	}

	// Extract the user ID from the Location header.
	location := resp.Header.Get("location")
	if location == "" {
		return 0, fmt.Errorf("no location header received")
	}
	// Assuming the location header is like "/api/v2.0/users/1"
	idStr := path.Base(location)
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, fmt.Errorf("failed to parse user id from location header %s: %w", location, err)
	}

	return id, nil
}

// updateHarborUser sends a PUT request to Harbor to update an existing user.
func (r *UserReconciler) updateHarborUser(ctx context.Context, harborConn *harborv1alpha1.HarborConnection, id int, payload updateUserRequest) error {
	updateURL := fmt.Sprintf("%s/api/v2.0/users/%d", harborConn.Spec.BaseURL, id)
	r.logger.V(1).Info("Sending user update request", "url", updateURL)

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal user payload for update: %w", err)
	}

	reqHTTP, err := http.NewRequest("PUT", updateURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request for user update: %w", err)
	}
	reqHTTP.Header.Set("Content-Type", "application/json")

	username, password, err := getHarborAuth(ctx, r.Client, harborConn)
	if err != nil {
		return fmt.Errorf("failed to get Harbor auth credentials: %w", err)
	}
	reqHTTP.SetBasicAuth(username, password)

	resp, err := http.DefaultClient.Do(reqHTTP)
	if err != nil {
		return fmt.Errorf("failed to perform HTTP request for user update: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update user: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// getHarborUser retrieves the user from Harbor by listing users and searching by username.
func (r *UserReconciler) getHarborUser(ctx context.Context, harborConn *harborv1alpha1.HarborConnection, usernameQuery string) (*harborUserResponse, error) {
	usersURL := fmt.Sprintf("%s/api/v2.0/users?q=username=%s", harborConn.Spec.BaseURL, usernameQuery)
	reqHTTP, err := http.NewRequest("GET", usersURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request for users: %w", err)
	}
	reqHTTP.Header.Set("Content-Type", "application/json")

	username, password, err := getHarborAuth(ctx, r.Client, harborConn)
	if err != nil {
		return nil, fmt.Errorf("failed to get Harbor auth credentials: %w", err)
	}
	reqHTTP.SetBasicAuth(username, password)

	resp, err := http.DefaultClient.Do(reqHTTP)
	if err != nil {
		return nil, fmt.Errorf("failed to perform GET request for users: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list users: status %d, body: %s", resp.StatusCode, string(body))
	}

	var users []harborUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, fmt.Errorf("failed to decode users response: %w", err)
	}

	for _, u := range users {
		if strings.EqualFold(u.Username, usernameQuery) {
			return &u, nil
		}
	}
	return nil, nil
}

// getHarborUserByID retrieves the user from Harbor using its ID.
func (r *UserReconciler) getHarborUserByID(ctx context.Context, harborConn *harborv1alpha1.HarborConnection, id int) (*harborUserResponse, error) {
	getURL := fmt.Sprintf("%s/api/v2.0/users/%d", harborConn.Spec.BaseURL, id)
	reqHTTP, err := http.NewRequest("GET", getURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request for user by ID: %w", err)
	}
	reqHTTP.Header.Set("Content-Type", "application/json")

	username, password, err := getHarborAuth(ctx, r.Client, harborConn)
	if err != nil {
		return nil, fmt.Errorf("failed to get Harbor auth credentials: %w", err)
	}
	reqHTTP.SetBasicAuth(username, password)

	resp, err := http.DefaultClient.Do(reqHTTP)
	if err != nil {
		return nil, fmt.Errorf("failed to perform GET request for user by ID: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get user by ID: status %d, body: %s", resp.StatusCode, string(body))
	}

	var u harborUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return nil, fmt.Errorf("failed to decode user response by ID: %w", err)
	}

	return &u, nil
}

// userNeedsUpdate compares the desired user configuration with the existing user.
func userNeedsUpdate(desired updateUserRequest, current harborUserResponse) bool {
	if desired.Email != current.Email {
		return true
	}
	if desired.Realname != current.Realname {
		return true
	}
	if desired.Comment != current.Comment {
		return true
	}
	return false
}

// deleteHarborUser implements the deletion logic for a user in Harbor.
func (r *UserReconciler) deleteHarborUser(ctx context.Context, harborConn *harborv1alpha1.HarborConnection, user *harborv1alpha1.User) error {
	// If no HarborUserID is set, there's nothing to delete.
	if user.Status.HarborUserID == 0 {
		r.logger.V(1).Info("No HarborUserID present, nothing to delete")
		return nil
	}

	deleteURL := fmt.Sprintf("%s/api/v2.0/users/%d", harborConn.Spec.BaseURL, user.Status.HarborUserID)
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

	// If the user is already deleted, log at debug verbosity.
	if resp.StatusCode == http.StatusNotFound {
		r.logger.V(1).Info("User not found during deletion; assuming it was already deleted", "HarborUserID", user.Status.HarborUserID)
		return nil
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete user: status %d, body: %s", resp.StatusCode, string(body))
	}

	r.logger.Info("Successfully deleted user from Harbor", "Username", user.Spec.Username)
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *UserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&harborv1alpha1.User{}).
		Named("user").
		Complete(r)
}
