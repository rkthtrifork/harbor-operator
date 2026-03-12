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

	"github.com/go-logr/logr"
	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
	"github.com/rkthtrifork/harbor-operator/internal/harborclient"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	finalizerName = "harbor.harbor-operator.io/finalizer"
	adminName     = "admin"

	harborConnectionRefNamespacedIndex = "harbor.harbor-operator.io/harborConnectionRefNamespaced"
	harborConnectionRefClusterIndex    = "harbor.harbor-operator.io/harborConnectionRefCluster"
)

type connectionConfig struct {
	baseURL           string
	namespace         string
	credentials       *harborv1alpha1.Credentials
	caBundle          string
	caBundleSecretRef *harborv1alpha1.SecretReference
	displayName       string
}

type harborConnectionRefAccessor func(client.Object) harborv1alpha1.HarborConnectionReference

func setupHarborBackedController(
	mgr ctrl.Manager,
	obj client.Object,
	newList func() client.ObjectList,
	getRef harborConnectionRefAccessor,
	name string,
) (*builder.TypedBuilder[reconcile.Request], error) {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), obj, harborConnectionRefNamespacedIndex, func(raw client.Object) []string {
		ref := normalizedHarborConnectionRef(getRef(raw))
		if ref.Name == "" || ref.Kind != harborv1alpha1.HarborConnectionReferenceKindNamespaced {
			return nil
		}
		return []string{types.NamespacedName{Namespace: raw.GetNamespace(), Name: ref.Name}.String()}
	}); err != nil {
		return nil, err
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), obj, harborConnectionRefClusterIndex, func(raw client.Object) []string {
		ref := normalizedHarborConnectionRef(getRef(raw))
		if ref.Name == "" || ref.Kind != harborv1alpha1.HarborConnectionReferenceKindCluster {
			return nil
		}
		return []string{ref.Name}
	}); err != nil {
		return nil, err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(obj).
		Watches(
			&harborv1alpha1.HarborConnection{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []reconcile.Request {
				return requestsForIndexedHarborConnection(ctx, mgr, newList, harborConnectionRefNamespacedIndex, client.ObjectKeyFromObject(object).String())
			}),
		).
		Watches(
			&harborv1alpha1.ClusterHarborConnection{},
			handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []reconcile.Request {
				return requestsForIndexedHarborConnection(ctx, mgr, newList, harborConnectionRefClusterIndex, object.GetName())
			}),
		).
		Named(name), nil
}

func requestsForIndexedHarborConnection(
	ctx context.Context,
	mgr ctrl.Manager,
	newList func() client.ObjectList,
	indexName, indexValue string,
) []reconcile.Request {
	list := newList()
	if err := mgr.GetClient().List(ctx, list, client.MatchingFields{indexName: indexValue}); err != nil {
		ctrl.Log.WithName("harbor-connection-watch").Error(err, "Failed to list Harbor dependents", "index", indexName, "value", indexValue)
		return nil
	}

	requests := []reconcile.Request{}
	if err := apimeta.EachListItem(list, func(item runtime.Object) error {
		obj, ok := item.(client.Object)
		if !ok {
			return nil
		}
		requests = append(requests, reconcile.Request{NamespacedName: client.ObjectKeyFromObject(obj)})
		return nil
	}); err != nil {
		ctrl.Log.WithName("harbor-connection-watch").Error(err, "Failed to walk Harbor dependents", "index", indexName, "value", indexValue)
		return nil
	}
	return requests
}

func resolveHarborConnection(ctx context.Context, c client.Client, namespace string, ref harborv1alpha1.HarborConnectionReference) (*connectionConfig, error) {
	kind := ref.Kind
	if kind == "" {
		kind = harborv1alpha1.HarborConnectionReferenceKindNamespaced
	}

	switch kind {
	case harborv1alpha1.HarborConnectionReferenceKindNamespaced:
		var harborConn harborv1alpha1.HarborConnection
		key := types.NamespacedName{Namespace: namespace, Name: ref.Name}
		if err := c.Get(ctx, key, &harborConn); err != nil {
			return nil, err
		}
		return &connectionConfig{
			baseURL:           harborConn.Spec.BaseURL,
			namespace:         harborConn.Namespace,
			credentials:       harborConn.Spec.Credentials,
			caBundle:          harborConn.Spec.CABundle,
			caBundleSecretRef: harborConn.Spec.CABundleSecretRef,
			displayName:       fmt.Sprintf("HarborConnection %s/%s", harborConn.Namespace, harborConn.Name),
		}, nil
	case harborv1alpha1.HarborConnectionReferenceKindCluster:
		var harborConn harborv1alpha1.ClusterHarborConnection
		key := types.NamespacedName{Name: ref.Name}
		if err := c.Get(ctx, key, &harborConn); err != nil {
			return nil, err
		}
		return &connectionConfig{
			baseURL:           harborConn.Spec.BaseURL,
			namespace:         "",
			credentials:       harborConn.Spec.Credentials,
			caBundle:          harborConn.Spec.CABundle,
			caBundleSecretRef: harborConn.Spec.CABundleSecretRef,
			displayName:       fmt.Sprintf("ClusterHarborConnection %s", harborConn.Name),
		}, nil
	default:
		return nil, fmt.Errorf("unsupported harborConnectionRef.kind %q", ref.Kind)
	}
}

func getHarborAuth(ctx context.Context, c client.Client, conn *connectionConfig) (string, string, error) {
	if conn.credentials == nil {
		return "", "", fmt.Errorf("%s has no credentials configured", conn.displayName)
	}
	pass, err := readSecretValue(ctx, c, harborv1alpha1.SecretReference{
		Name:      conn.credentials.PasswordSecretRef.Name,
		Key:       conn.credentials.PasswordSecretRef.Key,
		Namespace: conn.credentials.PasswordSecretRef.Namespace,
	}, conn.namespace, "access_secret")
	if err != nil {
		return "", "", err
	}
	return conn.credentials.Username, pass, nil
}

func getHarborClient(ctx context.Context, c client.Client, namespace string, ref harborv1alpha1.HarborConnectionReference) (*harborclient.Client, error) {
	conn, err := resolveHarborConnection(ctx, c, namespace, ref)
	if err != nil {
		return nil, err
	}
	return buildHarborClient(ctx, c, conn, true)
}

func buildHarborClient(ctx context.Context, c client.Client, conn *connectionConfig, requireCredentials bool) (*harborclient.Client, error) {
	user, pass := "", ""
	if conn.credentials != nil {
		var err error
		user, pass, err = getHarborAuth(ctx, c, conn)
		if err != nil {
			return nil, err
		}
	} else if requireCredentials {
		return nil, fmt.Errorf("%s has no credentials configured", conn.displayName)
	}

	caBundle, err := resolveConnectionCABundle(ctx, c, conn)
	if err != nil {
		return nil, err
	}
	return newHarborClient(conn.baseURL, user, pass, caBundle)
}

func resolveConnectionCABundle(ctx context.Context, c client.Client, conn *connectionConfig) (string, error) {
	caBundle := conn.caBundle
	if conn.caBundleSecretRef != nil {
		if caBundle != "" {
			return "", fmt.Errorf("caBundle and caBundleSecretRef are mutually exclusive")
		}
		value, err := readSecretValue(ctx, c, *conn.caBundleSecretRef, conn.namespace, "ca.crt")
		if err != nil {
			return "", fmt.Errorf("failed to read caBundleSecretRef: %w", err)
		}
		caBundle = value
	}
	return caBundle, nil
}

func newHarborClient(baseURL, user, pass, caBundle string) (*harborclient.Client, error) {
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
		return harborclient.NewWithHTTPClient(baseURL, user, pass, httpClient), nil
	}

	return harborclient.New(baseURL, user, pass), nil
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

type HarborDeletionPolicyAware interface {
	GetDeletionPolicy() harborv1alpha1.DeletionPolicy
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

func loadResource(ctx context.Context, c client.Client, key types.NamespacedName, obj client.Object, logger logr.Logger) (bool, error) {
	if err := c.Get(ctx, key, obj); err != nil {
		if client.IgnoreNotFound(err) == nil {
			logger.V(1).Info("Resource disappeared")
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func markReconcilingIfNeeded(ctx context.Context, c client.Client, obj client.Object, base *harborv1alpha1.HarborStatusBase, generation int64) error {
	if base.ObservedGeneration == generation {
		return nil
	}
	return setReconcilingStatus(ctx, c, obj, base, generation, "", "")
}

func requeueOnRemoteNotFound(ctx context.Context, c client.Client, obj client.Object, base *harborv1alpha1.HarborStatusBase, generation int64, reset func(), message string) (reconcile.Result, error) {
	if reset != nil {
		reset()
	}
	if err := setReconcilingStatus(ctx, c, obj, base, generation, "NotFound", message); err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{Requeue: true}, nil
}

func finalizeWithoutHarborConnection(ctx context.Context, c client.Client, obj client.Object, deletionPolicy harborv1alpha1.DeletionPolicy, requiresRemoteCleanup bool, err error) (bool, error) {
	if obj.GetDeletionTimestamp().IsZero() || !apierrors.IsNotFound(err) {
		return false, nil
	}
	if !requiresRemoteCleanup || deletionPolicy == harborv1alpha1.DeletionPolicyOrphan {
		return true, removeFinalizer(ctx, c, obj)
	}
	return false, nil
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
