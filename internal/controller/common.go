package controller

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
	"github.com/rkthtrifork/harbor-operator/internal/harborclient"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	finalizerName = "harbor.harbor-operator.io/finalizer"
	adminName     = "admin"
)

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

	caBundle := conn.Spec.CABundle
	if conn.Spec.CABundleSecretRef != nil {
		if caBundle != "" {
			return nil, fmt.Errorf("caBundle and caBundleSecretRef are mutually exclusive")
		}
		value, err := readSecretValue(ctx, c, *conn.Spec.CABundleSecretRef, conn.Namespace, "ca.crt")
		if err != nil {
			return nil, fmt.Errorf("failed to read caBundleSecretRef: %w", err)
		}
		caBundle = value
	}

	if caBundle != "" {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM([]byte(caBundle)) {
			return nil, fmt.Errorf("invalid caBundle: no certificates found")
		}
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.TLSClientConfig = &tls.Config{RootCAs: pool}
		httpClient := &http.Client{
			Timeout:   30 * time.Second,
			Transport: transport,
		}
		return harborclient.NewWithHTTPClient(conn.Spec.BaseURL, user, pass, httpClient), nil
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

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func finalizeIfDeleting(ctx context.Context, c client.Client, obj client.Object, deleteFn func() error) (bool, error) {
	if obj.GetDeletionTimestamp().IsZero() {
		return false, nil
	}
	if deleteFn != nil {
		if err := deleteFn(); err != nil {
			return true, err
		}
	}
	if err := removeFinalizer(ctx, c, obj); err != nil {
		return true, err
	}
	return true, nil
}

func resolveRegistryID(ctx context.Context, c client.Client, namespace string, ref *harborv1alpha1.RegistryReference, id *int) (int, error) {
	if ref != nil && id != nil {
		return 0, fmt.Errorf("registryRef and registryID are mutually exclusive")
	}
	if ref != nil {
		if ref.Name == "" {
			return 0, fmt.Errorf("registryRef.name must not be empty")
		}
		ns := ref.Namespace
		if ns == "" {
			ns = namespace
		}
		var registry harborv1alpha1.Registry
		if err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: ref.Name}, &registry); err != nil {
			return 0, err
		}
		if registry.Status.HarborRegistryID == 0 {
			return 0, fmt.Errorf("referenced Registry %s/%s does not have harborRegistryID yet", ns, ref.Name)
		}
		return registry.Status.HarborRegistryID, nil
	}
	if id != nil {
		if *id <= 0 {
			return 0, fmt.Errorf("registryID must be greater than 0")
		}
		return *id, nil
	}
	return 0, fmt.Errorf("registryRef or registryID must be set")
}

func resolveProject(ctx context.Context, c client.Client, hc *harborclient.Client, namespace string, ref *harborv1alpha1.ProjectReference, nameOrID string) (string, int, error) {
	if ref != nil && nameOrID != "" {
		return "", 0, fmt.Errorf("projectRef and projectNameOrID are mutually exclusive")
	}
	if ref != nil {
		if ref.Name == "" {
			return "", 0, fmt.Errorf("projectRef.name must not be empty")
		}
		ns := ref.Namespace
		if ns == "" {
			ns = namespace
		}
		var project harborv1alpha1.Project
		if err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: ref.Name}, &project); err != nil {
			return "", 0, err
		}
		if project.Status.HarborProjectID == 0 {
			return "", 0, fmt.Errorf("referenced Project %s/%s does not have harborProjectID yet", ns, ref.Name)
		}
		return strconv.Itoa(project.Status.HarborProjectID), project.Status.HarborProjectID, nil
	}
	if nameOrID == "" {
		return "", 0, fmt.Errorf("projectRef or projectNameOrID must be set")
	}
	if id, err := strconv.Atoi(nameOrID); err == nil && id > 0 {
		return nameOrID, id, nil
	}
	if hc == nil {
		return nameOrID, 0, nil
	}
	project, err := hc.FindProjectByName(ctx, nameOrID)
	if err != nil {
		return "", 0, err
	}
	if project == nil || project.ProjectID == 0 {
		return "", 0, fmt.Errorf("project %q not found in Harbor", nameOrID)
	}
	return nameOrID, project.ProjectID, nil
}
