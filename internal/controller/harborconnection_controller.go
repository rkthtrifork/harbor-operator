package controller

import (
	"context"
	"fmt"
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
	"github.com/rkthtrifork/harbor-operator/internal/harborclient"
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
	if err := r.validateBaseURL(&conn); err != nil {
		SetReadyCondition(&conn.Status.Conditions, false, ReasonInvalidSpec, err.Error())
		SetStalledCondition(&conn.Status.Conditions, true, ReasonInvalidSpec, err.Error())
		_ = r.Status().Update(ctx, &conn)
		return ctrl.Result{}, err
	}

	// Set reconciling condition
	SetReconcilingCondition(&conn.Status.Conditions, true, ReasonReconcileSuccess, "Checking Harbor connectivity")
	_ = r.Status().Update(ctx, &conn)

	// If no credentials are provided, perform a non-authenticated connectivity check.
	if conn.Spec.Credentials == nil {
		return r.checkNonAuthConnectivity(ctx, &conn)
	}

	// Otherwise, perform an authenticated check.
	return r.checkAuthenticatedConnection(ctx, &conn)
}

// validateBaseURL verifies that the BaseURL is a valid URL and includes a protocol scheme.
func (r *HarborConnectionReconciler) validateBaseURL(conn *harborv1alpha1.HarborConnection) error {
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

func (r *HarborConnectionReconciler) checkNonAuthConnectivity(
	ctx context.Context, conn *harborv1alpha1.HarborConnection) (ctrl.Result, error) {

	hc := harborclient.New(conn.Spec.BaseURL, "", "") // no creds
	if err := hc.Ping(ctx); err != nil {
		SetReadyCondition(&conn.Status.Conditions, false, ReasonConnectionFailed, fmt.Sprintf("Failed to connect to Harbor: %v", err))
		SetStalledCondition(&conn.Status.Conditions, true, ReasonConnectionFailed, err.Error())
		SetReconcilingCondition(&conn.Status.Conditions, false, ReasonReconcileError, "Connection failed")
		_ = r.Status().Update(ctx, conn)
		return ctrl.Result{}, err
	}
	r.logger.Info("Harbor reachable without credentials")
	SetReadyCondition(&conn.Status.Conditions, true, ReasonReconcileSuccess, "Harbor is reachable")
	SetReconcilingCondition(&conn.Status.Conditions, false, ReasonReconcileSuccess, "Reconciliation complete")
	SetStalledCondition(&conn.Status.Conditions, false, ReasonReconcileSuccess, "")
	_ = r.Status().Update(ctx, conn)
	return ctrl.Result{}, nil
}

func (r *HarborConnectionReconciler) checkAuthenticatedConnection(
	ctx context.Context, conn *harborv1alpha1.HarborConnection) (ctrl.Result, error) {

	user := conn.Spec.Credentials.Username
	pass, err := r.getPassword(ctx, r.Client, conn) // unchanged helper
	if err != nil {
		SetReadyCondition(&conn.Status.Conditions, false, ReasonConnectionFailed, fmt.Sprintf("Failed to get credentials: %v", err))
		SetStalledCondition(&conn.Status.Conditions, true, ReasonConnectionFailed, err.Error())
		SetReconcilingCondition(&conn.Status.Conditions, false, ReasonReconcileError, "Failed to get credentials")
		_ = r.Status().Update(ctx, conn)
		return ctrl.Result{}, err
	}

	hc := harborclient.New(conn.Spec.BaseURL, user, pass)
	if _, err := hc.GetCurrentUser(ctx); err != nil {
		SetReadyCondition(&conn.Status.Conditions, false, ReasonConnectionFailed, fmt.Sprintf("Failed to authenticate with Harbor: %v", err))
		SetStalledCondition(&conn.Status.Conditions, true, ReasonConnectionFailed, err.Error())
		SetReconcilingCondition(&conn.Status.Conditions, false, ReasonReconcileError, "Authentication failed")
		_ = r.Status().Update(ctx, conn)
		return ctrl.Result{}, err
	}

	r.logger.Info("Successfully authenticated with Harbor API")
	SetReadyCondition(&conn.Status.Conditions, true, ReasonReconcileSuccess, "Successfully authenticated with Harbor")
	SetReconcilingCondition(&conn.Status.Conditions, false, ReasonReconcileSuccess, "Reconciliation complete")
	SetStalledCondition(&conn.Status.Conditions, false, ReasonReconcileSuccess, "")
	_ = r.Status().Update(ctx, conn)
	return ctrl.Result{}, nil
}

// Retrieve the secret containing the access secret.
func (r *HarborConnectionReconciler) getPassword(ctx context.Context, client client.Client, conn *harborv1alpha1.HarborConnection) (string, error) {
	secret, err := r.getSecret(ctx, conn)
	if err != nil {
		return "", err
	}

	secretKey := conn.Spec.Credentials.PasswordSecretRef.Key
	if secretKey == "" {
		secretKey = "access_secret"
	}
	accessSecretBytes, ok := secret.Data[secretKey]
	if !ok {
		return "", fmt.Errorf("key %q not found in secret %s/%s", secretKey, secret.Namespace, secret.Name)
	}
	return string(accessSecretBytes), nil
}

// getSecret fetches the secret specified in the HarborConnection credentials.
func (r *HarborConnectionReconciler) getSecret(ctx context.Context, conn *harborv1alpha1.HarborConnection) (*corev1.Secret, error) {
	secretKey := types.NamespacedName{
		Namespace: conn.Namespace,
		Name:      conn.Spec.Credentials.PasswordSecretRef.Name,
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
