package controller

import (
	"context"
	"fmt"
	"strings"

	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
	"github.com/rkthtrifork/harbor-operator/internal/harborclient"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const finalizerName = "harbor.harbor-operator.io/finalizer"

// getHarborConnection fetches the HarborConnection referenced in the Registry.
func getHarborConnection(ctx context.Context, c client.Client, namespace, name string) (*harborv1alpha1.HarborConnection, error) {
	var harborConn harborv1alpha1.HarborConnection
	key := types.NamespacedName{Namespace: namespace, Name: name}
	if err := c.Get(ctx, key, &harborConn); err != nil {
		return nil, err
	}
	return &harborConn, nil
}

// getHarborAuth is a helper function that retrieves Harbor authentication credentials.
// It can be called from any reconciler that has access to a client.Client.
func getHarborAuth(ctx context.Context, c client.Client, harborConn *harborv1alpha1.HarborConnection) (string, string, error) {
	secretKey := types.NamespacedName{
		Namespace: harborConn.Namespace,
		Name:      harborConn.Spec.Credentials.PasswordSecretRef.Name,
	}
	var secret corev1.Secret
	if err := c.Get(ctx, secretKey, &secret); err != nil {
		return "", "", err
	}

	accessSecretBytes, ok := secret.Data[harborConn.Spec.Credentials.PasswordSecretRef.Key]
	if !ok {
		return "", "", fmt.Errorf("access_secret not found in secret %s/%s", secretKey.Namespace, secretKey.Name)
	}
	return harborConn.Spec.Credentials.Username, string(accessSecretBytes), nil
}

func getHarborClient(ctx context.Context, c client.Client, namespace, name string) (*harborclient.Client, error) {
	conn, err := getHarborConnection(ctx, c, namespace, name)
	if err != nil {
		return nil, err
	}
	if conn.Spec.Credentials == nil {
		return nil, fmt.Errorf("HarborConnection %s/%s has no credentials configured", conn.Namespace, conn.Name)
	}
	user, pass, err := getHarborAuth(ctx, c, conn)
	if err != nil {
		return nil, err
	}
	return harborclient.New(conn.Spec.BaseURL, user, pass), nil
}

func ensureFinalizer(ctx context.Context, c client.Client, obj client.Object) error {
	if controllerutil.ContainsFinalizer(obj, finalizerName) {
		return nil
	}
	controllerutil.AddFinalizer(obj, finalizerName)
	return c.Update(ctx, obj)
}

func removeFinalizer(ctx context.Context, c client.Client, obj client.Object) error {
	if !controllerutil.ContainsFinalizer(obj, finalizerName) {
		return nil
	}
	controllerutil.RemoveFinalizer(obj, finalizerName)
	return c.Update(ctx, obj)
}

// DriftDetectable is an interface for objects that have a DriftDetectionInterval.
type DriftDetectable interface {
	GetDriftDetectionInterval() *metav1.Duration
}

func returnWithDriftDetection(obj DriftDetectable) (reconcile.Result, error) {
	if obj.GetDriftDetectionInterval() == nil || obj.GetDriftDetectionInterval().Duration == 0 {
		return reconcile.Result{}, nil
	}
	if obj.GetDriftDetectionInterval().Duration < 0 {
		return reconcile.Result{}, fmt.Errorf("drift detection interval must be greater than 0")
	}
	return reconcile.Result{RequeueAfter: obj.GetDriftDetectionInterval().Duration}, nil
}

func hashParts(parts ...string) string {
	return hashSecret(strings.Join(parts, "\n"))
}
