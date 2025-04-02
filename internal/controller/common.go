package controller

import (
	"context"
	"fmt"
	"net/url"

	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const finalizerName = "harbor.operator/finalizer"

// getHarborConnection fetches the HarborConnection referenced in the Registry.
func (r *RegistryReconciler) getHarborConnection(ctx context.Context, namespace, name string) (*harborv1alpha1.HarborConnection, error) {
	var harborConn harborv1alpha1.HarborConnection
	if err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &harborConn); err != nil {
		return nil, err
	}

	parsed, err := url.Parse(harborConn.Spec.BaseURL)
	if err != nil {
		return nil, err
	}
	if parsed.Scheme == "" {
		return nil, fmt.Errorf("baseURL %s is missing a protocol scheme", harborConn.Spec.BaseURL)
	}

	return &harborConn, nil
}

// DriftDetectable is an interface for objects that have a DriftDetectionInterval.
type DriftDetectable interface {
	GetDriftDetectionInterval() metav1.Duration
}

// returnWithDriftDetection now accepts any type that satisfies DriftDetectable.
func returnWithDriftDetection(obj DriftDetectable) (reconcile.Result, error) {
	if obj.GetDriftDetectionInterval().Duration > 0 {
		return reconcile.Result{RequeueAfter: obj.GetDriftDetectionInterval().Duration}, nil
	}
	return reconcile.Result{}, nil
}
