package controller

import (
	"context"
	"fmt"
	"net/url"

	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
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
