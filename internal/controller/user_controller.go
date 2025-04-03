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
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
	harborv1alpha1 "github.com/your-org/harbor-operator/api/v1alpha1"
)

// harborUserResponse represents a user as returned by Harbor.
type harborUserResponse struct {
	UserID   int    `json:"user_id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	RealName string `json:"real_name"`
	Comment  string `json:"comment"`
	SysAdmin bool   `json:"sysadmin"`
	// additional fields can be added if needed
}

// createUserRequest is the payload sent to Harbor when creating or updating a user.
type createUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	RealName string `json:"real_name,omitempty"`
	Comment  string `json:"comment,omitempty"`
	Password string `json:"password,omitempty"`
	// SysAdmin is not part of the profile update endpoint.
	// For sysadmin changes, a separate endpoint is used.
}

// UserReconciler reconciles a User object.
type UserReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	logger logr.Logger
}

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

	// Handle deletion.
	if !user.GetDeletionTimestamp().IsZero() {
		if controllerutil.ContainsFinalizer(&user, finalizerName) {
			if err := r.deleteHarborUser(ctx, &user); err != nil {
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

	// Retrieve the HarborConnection referenced by the User.
	harborConn, err := r.getHarborConnection(ctx, user.Namespace, user.Spec.HarborConnectionRef)
	if err != nil {
		r.logger.Error(err, "Failed to get HarborConnection", "HarborConnectionRef", user.Spec.HarborConnectionRef)
		return ctrl.Result{}, err
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

	// Retrieve the existing user from Harbor using HarborUserID if available.
	var existing *harborUserResponse
	if user.Status.HarborUserID != 0 {
		existing, err = r.getHarborUserByID(ctx, harborConn, user.Status.HarborUserID)
		if err != nil {
			r.logger.Error(err, "Failed to get user by ID from Harbor", "HarborUserID", user.Status.HarborUserID)
		}
	}

	// Build the desired user payload from the CR.
	desired := r.buildUserRequest(&user)

	// If a user exists and its profile differs from the desired state, update it.
	if existing != nil {
		if userNeedsUpdate(desired, *existing) {
			r.logger.Info("User in Harbor differs from desired state, updating", "Username", user.Spec.Username)
			if err := r.updateHarborUser(ctx, harborConn, existing.UserID, desired); err != nil {
				return ctrl.Result{}, err
			}
			r.logger.Info("Successfully updated user profile on Harbor", "Username", user.Spec.Username)
		} else {
			r.logger.V(1).Info("User profile is already in sync with desired state", "Username", user.Spec.Username)
		}

		// If the sysadmin flag is different, call the dedicated endpoint.
		if existing.SysAdmin != user.Spec.SysAdmin {
			if err := r.updateHarborUserSysAdmin(ctx, harborConn, existing.UserID, user.Spec.SysAdmin); err != nil {
				r.logger.Error(err, "Failed to update user sysadmin flag", "Username", user.Spec.Username)
				return ctrl.Result{}, err
			}
			r.logger.Info("Successfully updated user sysadmin flag", "Username", user.Spec.Username)
		}

		return returnWithDriftDetection(&user)
	}

	// If the user is not found, create a new one.
	if user.Status.HarborUserID != 0 {
		r.logger.Info("User with stored ID not found. Assuming it was removed externally. Creating new user", "Username", user.Spec.Username)
	} else {
		r.logger.Info("Creating new user", "Username", user.Spec.Username)
	}
	newID, err := r.createHarborUser(ctx, harborConn, desired)
	if err != nil {
		return ctrl.Result{}, err
	}
	user.Status.HarborUserID = newID
	if err := r.Status().Update(ctx, &user); err != nil {
		r.logger.Error(err, "Failed to update User status with Harbor user ID", "HarborUserID", newID)
		return ctrl.Result{}, err
	}
	r.logger.Info("Successfully created user on Harbor", "Username", user.Spec.Username, "HarborUserID", newID)

	return returnWithDriftDetection(&user)
}

// adoptExistingUser attempts to adopt an existing user from Harbor by username.
// If found, updates the CR status with the Harbor user ID.
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

// buildUserRequest constructs the JSON request for the user creation/update.
func (r *UserReconciler) buildUserRequest(user *harborv1alpha1.User) createUserRequest {
	return createUserRequest{
		Username: user.Spec.Username,
		Email:    user.Spec.Email,
		RealName: user.Spec.RealName,
		Comment:  user.Spec.Comment,
		Password: user.Spec.Password,
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
	defer resp.Body.Close()

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

// updateHarborUser sends a PUT request to Harbor to update an existing user's profile.
func (r *UserReconciler) updateHarborUser(ctx context.Context, harborConn *harborv1alpha1.HarborConnection, id int, payload createUserRequest) error {
	updateURL := fmt.Sprintf("%s/api/v2.0/users/%d", harborConn.Spec.BaseURL, id)
	r.logger.V(1).Info("Sending user profile update request", "url", updateURL)

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
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update user: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// updateHarborUserSysAdmin updates the sysadmin flag for a user using the dedicated endpoint.
func (r *UserReconciler) updateHarborUserSysAdmin(ctx context.Context, harborConn *harborv1alpha1.HarborConnection, id int, sysadmin bool) error {
	sysAdminURL := fmt.Sprintf("%s/api/v2.0/users/%d/sysadmin", harborConn.Spec.BaseURL, id)
	r.logger.V(1).Info("Sending user sysadmin update request", "url", sysAdminURL)

	// Create a payload that toggles the sysadmin flag.
	payload := map[string]bool{"sysadmin_flag": sysadmin}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal sysadmin payload: %w", err)
	}

	reqHTTP, err := http.NewRequest("PUT", sysAdminURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request for sysadmin update: %w", err)
	}
	reqHTTP.Header.Set("Content-Type", "application/json")

	username, password, err := getHarborAuth(ctx, r.Client, harborConn)
	if err != nil {
		return fmt.Errorf("failed to get Harbor auth credentials: %w", err)
	}
	reqHTTP.SetBasicAuth(username, password)

	resp, err := http.DefaultClient.Do(reqHTTP)
	if err != nil {
		return fmt.Errorf("failed to perform HTTP request for sysadmin update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to update sysadmin flag: status %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// getHarborUser retrieves the user from Harbor by listing users and searching by username.
func (r *UserReconciler) getHarborUser(ctx context.Context, harborConn *harborv1alpha1.HarborConnection, username string) (*harborUserResponse, error) {
	usersURL := fmt.Sprintf("%s/api/v2.0/users", harborConn.Spec.BaseURL)
	reqHTTP, err := http.NewRequest("GET", usersURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request for users: %w", err)
	}
	reqHTTP.Header.Set("Content-Type", "application/json")

	u, p, err := getHarborAuth(ctx, r.Client, harborConn)
	if err != nil {
		return nil, fmt.Errorf("failed to get Harbor auth credentials: %w", err)
	}
	reqHTTP.SetBasicAuth(u, p)

	resp, err := http.DefaultClient.Do(reqHTTP)
	if err != nil {
		return nil, fmt.Errorf("failed to perform GET request for users: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list users: status %d, body: %s", resp.StatusCode, string(body))
	}

	var users []harborUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, fmt.Errorf("failed to decode users response: %w", err)
	}

	for _, usr := range users {
		if strings.EqualFold(usr.Username, username) {
			return &usr, nil
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

	u, p, err := getHarborAuth(ctx, r.Client, harborConn)
	if err != nil {
		return nil, fmt.Errorf("failed to get Harbor auth credentials: %w", err)
	}
	reqHTTP.SetBasicAuth(u, p)

	resp, err := http.DefaultClient.Do(reqHTTP)
	if err != nil {
		return nil, fmt.Errorf("failed to perform GET request for user by ID: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get user by ID: status %d, body: %s", resp.StatusCode, string(body))
	}

	var usr harborUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&usr); err != nil {
		return nil, fmt.Errorf("failed to decode user response by ID: %w", err)
	}

	return &usr, nil
}

// userNeedsUpdate compares the desired user profile with the existing user.
func userNeedsUpdate(desired createUserRequest, current harborUserResponse) bool {
	return desired.Email != current.Email ||
		desired.RealName != current.RealName ||
		desired.Comment != current.Comment
}

// deleteHarborUser implements the deletion logic for a user in Harbor.
func (r *UserReconciler) deleteHarborUser(ctx context.Context, user *harborv1alpha1.User) error {
	harborConn, err := r.getHarborConnection(ctx, user.Namespace, user.Spec.HarborConnectionRef)
	if err != nil {
		return err
	}

	// If no HarborUserID is set, nothing to delete.
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

	u, p, err := getHarborAuth(ctx, r.Client, harborConn)
	if err != nil {
		return err
	}
	reqHTTP.SetBasicAuth(u, p)

	resp, err := http.DefaultClient.Do(reqHTTP)
	if err != nil {
		return fmt.Errorf("failed to perform DELETE request: %w", err)
	}
	defer resp.Body.Close()

	// If the user is already removed, assume deletion.
	if resp.StatusCode == http.StatusNotFound {
		r.logger.V(1).Info("User not found during deletion; assuming it was already removed", "HarborUserID", user.Status.HarborUserID)
		return nil
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete user: status %d, body: %s", resp.StatusCode, string(body))
	}

	r.logger.Info("Successfully removed user from Harbor", "Username", user.Spec.Username)
	return nil
}

// getHarborConnection is a helper to retrieve the HarborConnection resource.
// Adjust this implementation as needed.
func (r *UserReconciler) getHarborConnection(ctx context.Context, namespace, name string) (*harborv1alpha1.HarborConnection, error) {
	var harborConn harborv1alpha1.HarborConnection
	key := types.NamespacedName{Namespace: namespace, Name: name}
	if err := r.Get(ctx, key, &harborConn); err != nil {
		return nil, err
	}
	return &harborConn, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *UserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&harborv1alpha1.User{}).
		Named("user").
		Complete(r)
}
