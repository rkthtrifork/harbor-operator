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

// MemberReconciler reconciles a Member object.
type MemberReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	logger logr.Logger
}

// RBAC permissions.
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=members,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=members/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections,verbs=get;list;watch

// Reconcile implements the reconciliation loop for the Member resource.
func (r *MemberReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.FromContext(ctx).WithName(fmt.Sprintf("[Member:%s]", req.NamespacedName))

	// Fetch the Member instance.
	var member harborv1alpha1.Member
	if err := r.Get(ctx, req.NamespacedName, &member); err != nil {
		if errors.IsNotFound(err) {
			r.logger.Info("Member resource not found; it may have been deleted")
			return ctrl.Result{}, nil
		}
		r.logger.Error(err, "Failed to get Member")
		return ctrl.Result{}, err
	}

	// Retrieve the HarborConnection referenced by the Member.
	harborConn, err := r.getHarborConnection(ctx, member.Namespace, member.Spec.HarborConnectionRef)
	if err != nil {
		r.logger.Error(err, "Failed to get HarborConnection", "HarborConnectionRef", member.Spec.HarborConnectionRef)
		return ctrl.Result{}, err
	}

	// Validate the Harbor BaseURL.
	if err := r.validateBaseURL(harborConn.Spec.BaseURL); err != nil {
		r.logger.Error(err, "Invalid Harbor BaseURL", "BaseURL", harborConn.Spec.BaseURL)
		return ctrl.Result{}, err
	}

	// Convert role name to role id using the provided mapping.
	roleID, err := r.convertRoleNameToID(member.Spec.Role)
	if err != nil {
		r.logger.Error(err, "Invalid role", "Role", member.Spec.Role)
		return ctrl.Result{}, err
	}

	// Build the member creation payload.
	memberRequest := r.buildMemberRequest(&member, roleID)

	// Build the Harbor API URL for creating a project member.
	// We use ProjectRef in the path; if it is a name, we indicate that with a header.
	membersURL := fmt.Sprintf("%s/api/v2.0/projects/%s/members", harborConn.Spec.BaseURL, member.Spec.ProjectRef)
	r.logger.Info("Sending member creation request", "url", membersURL)

	// Marshal the payload to JSON.
	payloadBytes, err := json.Marshal(memberRequest)
	if err != nil {
		r.logger.Error(err, "Failed to marshal member payload")
		return ctrl.Result{}, err
	}

	// Create the HTTP POST request.
	reqHTTP, err := http.NewRequest("POST", membersURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		r.logger.Error(err, "Failed to create HTTP request for member creation")
		return ctrl.Result{}, err
	}
	reqHTTP.Header.Set("Content-Type", "application/json")

	// Generate a unique request ID for the header.
	reqHTTP.Header.Set("X-Request-Id", fmt.Sprintf("%d", time.Now().UnixNano()))
	// Indicate that the project reference in the URL is a name.
	reqHTTP.Header.Set("X-Is-Resource-Name", "true")

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
		r.logger.Error(err, "Failed to perform HTTP request for member creation")
		return ctrl.Result{}, err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			r.logger.Error(err, "Failed to close response body")
		}
	}()

	// Check for a successful status code (e.g., 201 Created).
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		err := fmt.Errorf("failed to create member: status %d, body: %s", resp.StatusCode, string(body))
		r.logger.Error(err, "Harbor member creation failed")
		return ctrl.Result{}, err
	}

	r.logger.Info("Successfully created member on Harbor", "Project", member.Spec.ProjectRef)
	// Optionally update Member status here if needed.
	return ctrl.Result{}, nil
}

// convertRoleNameToID converts a human-readable role name into the corresponding Harbor role ID.
// Mapping: "admin": 1, "developer": 2, "guest": 3, "maintainer": 4. For "guest", you may also use 5.
func (r *MemberReconciler) convertRoleNameToID(role string) (int, error) {
	switch strings.ToLower(role) {
	case "admin":
		return 1, nil
	case "developer":
		return 2, nil
	case "guest":
		// Defaulting to 3 for guest; adjust if needed.
		return 3, nil
	case "maintainer":
		return 4, nil
	default:
		return 0, fmt.Errorf("unsupported role: %s", role)
	}
}

// createMemberRequest represents the payload sent to Harbor to create a project member.
type createMemberRequest struct {
	RoleID      int                         `json:"role_id"`
	MemberUser  *harborv1alpha1.MemberUser  `json:"member_user,omitempty"`
	MemberGroup *harborv1alpha1.MemberGroup `json:"member_group,omitempty"`
}

// buildMemberRequest constructs the JSON payload for the member creation request.
func (r *MemberReconciler) buildMemberRequest(member *harborv1alpha1.Member, roleID int) createMemberRequest {
	return createMemberRequest{
		RoleID:      roleID,
		MemberUser:  member.Spec.MemberUser,
		MemberGroup: member.Spec.MemberGroup,
	}
}

// getHarborConnection retrieves the HarborConnection referenced in the Member.
func (r *MemberReconciler) getHarborConnection(ctx context.Context, namespace, name string) (*harborv1alpha1.HarborConnection, error) {
	var harborConn harborv1alpha1.HarborConnection
	if err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &harborConn); err != nil {
		return nil, err
	}
	return &harborConn, nil
}

// validateBaseURL verifies that the provided URL is valid and contains a scheme.
func (r *MemberReconciler) validateBaseURL(baseURL string) error {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return err
	}
	if parsedURL.Scheme == "" {
		return fmt.Errorf("baseURL %s is missing a protocol scheme", baseURL)
	}
	return nil
}

// getHarborAuth retrieves the Harbor authentication credentials from the HarborConnection.
func (r *MemberReconciler) getHarborAuth(ctx context.Context, harborConn *harborv1alpha1.HarborConnection) (string, string, error) {
	secretKey := types.NamespacedName{
		Namespace: harborConn.Namespace,
		Name:      "temp",
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
func (r *MemberReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&harborv1alpha1.Member{}).
		Named("member").
		Complete(r)
}
