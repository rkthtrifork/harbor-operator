package controller

import (
	"context"
	"fmt"
	"net/url"

	"k8s.io/apimachinery/pkg/runtime"
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
	if found, err := loadResource(ctx, r.Client, req.NamespacedName, &conn, r.logger); err != nil {
		r.logger.Error(err, "Failed to get HarborConnection")
		return ctrl.Result{}, err
	} else if !found {
		return ctrl.Result{}, nil
	}

	if err := markReconcilingIfNeeded(ctx, r.Client, &conn, &conn.Status.HarborStatusBase, conn.Generation); err != nil {
		return ctrl.Result{}, err
	}

	// Validate the BaseURL.
	if err := r.validateBaseURL(&conn); err != nil {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &conn, &conn.Status.HarborStatusBase, conn.Generation, err)
	}

	// If no credentials are provided, perform a non-authenticated connectivity check.
	if conn.Spec.Credentials == nil {
		hc, err := buildHarborClient(ctx, r.Client, &connectionConfig{
			baseURL:           conn.Spec.BaseURL,
			namespace:         conn.Namespace,
			credentials:       conn.Spec.Credentials,
			caBundle:          conn.Spec.CABundle,
			caBundleSecretRef: conn.Spec.CABundleSecretRef,
			displayName:       fmt.Sprintf("HarborConnection %s/%s", conn.Namespace, conn.Name),
		}, false)
		if err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &conn, &conn.Status.HarborStatusBase, conn.Generation, err)
		}
		if err := hc.Ping(ctx); err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &conn, &conn.Status.HarborStatusBase, conn.Generation, err)
		}
		conn.Status.Authenticated = false
		if err := setReadyStatus(ctx, r.Client, &conn, &conn.Status.HarborStatusBase, conn.Generation, "Reachable", "Harbor reachable without credentials"); err != nil {
			return ctrl.Result{}, err
		}
		r.logger.Info("Harbor reachable without credentials")
		return ctrl.Result{}, nil
	}

	// Otherwise, perform an authenticated check.
	hc, err := buildHarborClient(ctx, r.Client, &connectionConfig{
		baseURL:           conn.Spec.BaseURL,
		namespace:         conn.Namespace,
		credentials:       conn.Spec.Credentials,
		caBundle:          conn.Spec.CABundle,
		caBundleSecretRef: conn.Spec.CABundleSecretRef,
		displayName:       fmt.Sprintf("HarborConnection %s/%s", conn.Namespace, conn.Name),
	}, true)
	if err != nil {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &conn, &conn.Status.HarborStatusBase, conn.Generation, err)
	}

	if _, err := hc.GetCurrentUser(ctx); err != nil {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &conn, &conn.Status.HarborStatusBase, conn.Generation, err)
	}

	conn.Status.Authenticated = true
	if err := setReadyStatus(ctx, r.Client, &conn, &conn.Status.HarborStatusBase, conn.Generation, "Authenticated", "Successfully authenticated with Harbor API"); err != nil {
		return ctrl.Result{}, err
	}

	r.logger.Info("Successfully authenticated with Harbor API")
	return ctrl.Result{}, nil
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

// SetupWithManager sets up the controller with the Manager.
func (r *HarborConnectionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&harborv1alpha1.HarborConnection{}).
		Named("harborconnection").
		Complete(r)
}
