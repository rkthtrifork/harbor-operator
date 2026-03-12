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

type ClusterHarborConnectionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	logger logr.Logger
}

// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=clusterharborconnections,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=clusterharborconnections/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=clusterharborconnections/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *ClusterHarborConnectionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.FromContext(ctx).WithName(fmt.Sprintf("[ClusterHarborConnection:%s]", req.Name))

	var conn harborv1alpha1.ClusterHarborConnection
	if err := r.Get(ctx, req.NamespacedName, &conn); err != nil {
		if client.IgnoreNotFound(err) == nil {
			r.logger.V(1).Info("ClusterHarborConnection resource not found")
			return ctrl.Result{}, nil
		}
		r.logger.Error(err, "Failed to get ClusterHarborConnection")
		return ctrl.Result{}, err
	}

	if err := markReconcilingIfNeeded(ctx, r.Client, &conn, &conn.Status.HarborStatusBase, conn.Generation); err != nil {
		return ctrl.Result{}, err
	}

	if err := r.validateBaseURL(conn.Spec.BaseURL); err != nil {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &conn, &conn.Status.HarborStatusBase, conn.Generation, err)
	}

	cfg := &connectionConfig{
		baseURL:           conn.Spec.BaseURL,
		namespace:         "",
		credentials:       conn.Spec.Credentials,
		caBundle:          conn.Spec.CABundle,
		caBundleSecretRef: conn.Spec.CABundleSecretRef,
		displayName:       fmt.Sprintf("ClusterHarborConnection %s", conn.Name),
	}

	if conn.Spec.Credentials == nil {
		hc, err := buildHarborClient(ctx, r.Client, cfg, false)
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
		return ctrl.Result{}, nil
	}

	hc, err := buildHarborClient(ctx, r.Client, cfg, true)
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
	return ctrl.Result{}, nil
}

func (r *ClusterHarborConnectionReconciler) validateBaseURL(baseURL string) error {
	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		r.logger.Error(err, "Invalid baseURL format")
		return err
	}
	if parsedURL.Scheme == "" {
		err := fmt.Errorf("baseURL %s is missing a protocol scheme", baseURL)
		r.logger.Error(err, "Invalid baseURL")
		return err
	}
	return nil
}

func (r *ClusterHarborConnectionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&harborv1alpha1.ClusterHarborConnection{}).
		Named("clusterharborconnection").
		Complete(r)
}
