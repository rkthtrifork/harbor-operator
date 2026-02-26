package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
	"github.com/rkthtrifork/harbor-operator/internal/harborclient"
)

type RegistryReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	logger logr.Logger
}

// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=registries,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=registries/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=registries/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections,verbs=get;list;watch

func (r *RegistryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.FromContext(ctx).WithName(fmt.Sprintf("[Registry:%s]", req.NamespacedName))

	// Load CR
	var cr harborv1alpha1.Registry
	if err := r.Get(ctx, req.NamespacedName, &cr); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if cr.Status.ObservedGeneration != cr.Generation {
		if err := setReconcilingStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, "", ""); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Harbor client
	hc, err := getHarborClient(ctx, r.Client, cr.Namespace, cr.Spec.HarborConnectionRef)
	if err != nil {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}

	// Deletion
	if done, err := finalizeIfDeleting(ctx, r.Client, &cr, func() error {
		return r.deleteRegistry(ctx, hc, &cr)
	}); done {
		return ctrl.Result{}, err
	}

	// Finalizer
	if err := ensureFinalizer(ctx, r.Client, &cr); err != nil {
		return ctrl.Result{}, err
	}

	// Defaults & adoption
	cr.Spec.Name = defaultString(cr.Spec.Name, cr.Name)

	if cr.Status.HarborRegistryID == 0 && cr.Spec.AllowTakeover {
		if ok, err := r.adoptExisting(ctx, hc, &cr); err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		} else if ok {
			r.logger.Info("Adopted registry", "ID", cr.Status.HarborRegistryID)
		}
	}

	credential, credHash, caCert, err := r.buildRegistryCredential(ctx, &cr)
	if err != nil {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}

	// Desired payloads
	createReq := r.buildCreateReq(cr, credential, caCert)
	updateReq := r.buildUpdateReq(cr, credential, caCert)

	// Create / Update
	if cr.Status.HarborRegistryID == 0 {
		id, err := hc.CreateRegistry(ctx, createReq)
		if err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		}
		cr.Status.HarborRegistryID = id
		cr.Status.CredentialHash = credHash
		if err := setReadyStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, "Created", "Registry created"); err != nil {
			return ctrl.Result{}, err
		}
		r.logger.Info("Created registry", "ID", id)
		return returnWithDriftDetection(&cr.Spec.HarborSpecBase)
	}

	current, err := hc.GetRegistryByID(ctx, cr.Status.HarborRegistryID)
	if err != nil {
		if harborclient.IsNotFound(err) {
			cr.Status.HarborRegistryID = 0
			if err := setReconcilingStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, "NotFound", "Registry not found in Harbor"); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}

	statusChanged := false
	if registryNeedsUpdate(cr, *current, credHash, caCert) {
		if err := hc.UpdateRegistry(ctx, current.ID, updateReq); err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		}
		if credHash != "" && credHash != cr.Status.CredentialHash {
			cr.Status.CredentialHash = credHash
			statusChanged = true
		}
		r.logger.Info("Updated registry", "ID", current.ID)
	}
	condChanged := markReady(&cr.Status.HarborStatusBase, cr.Generation, "Reconciled", "Registry reconciled")
	if statusChanged || condChanged {
		if err := r.Status().Update(ctx, &cr); err != nil {
			return ctrl.Result{}, err
		}
	}
	return returnWithDriftDetection(&cr.Spec.HarborSpecBase)
}

func (r *RegistryReconciler) deleteRegistry(ctx context.Context, hc *harborclient.Client, cr *harborv1alpha1.Registry) error {
	if cr.Status.HarborRegistryID == 0 {
		return nil
	}
	err := hc.DeleteRegistry(ctx, cr.Status.HarborRegistryID)
	if harborclient.IsNotFound(err) {
		return nil
	}
	return err
}

func (r *RegistryReconciler) adoptExisting(ctx context.Context, hc *harborclient.Client, cr *harborv1alpha1.Registry) (bool, error) {
	reg, err := hc.FindRegistryByName(ctx, cr.Spec.Name)
	if err != nil {
		return false, err
	}
	if reg != nil {
		cr.Status.HarborRegistryID = reg.ID
		return true, r.Status().Update(ctx, cr)
	}
	return false, nil
}

func (r *RegistryReconciler) buildCreateReq(cr harborv1alpha1.Registry, credential *harborclient.RegistryCredential, caCert string) harborclient.CreateRegistryRequest {
	desired := harborclient.CreateRegistryRequest{
		URL:           cr.Spec.URL,
		Name:          cr.Spec.Name,
		Description:   cr.Spec.Description,
		Type:          cr.Spec.Type,
		Insecure:      cr.Spec.Insecure,
		CACertificate: caCert,
		Credential:    credential,
	}
	return desired
}

func (r *RegistryReconciler) buildUpdateReq(cr harborv1alpha1.Registry, credential *harborclient.RegistryCredential, caCert string) harborclient.UpdateRegistryRequest {
	req := harborclient.UpdateRegistryRequest{
		Name:          cr.Spec.Name,
		Description:   cr.Spec.Description,
		URL:           cr.Spec.URL,
		Insecure:      cr.Spec.Insecure,
		CACertificate: caCert,
	}
	if credential != nil {
		req.CredentialType = credential.Type
		req.AccessKey = credential.AccessKey
		req.AccessSecret = credential.AccessSecret
	}
	return req
}

func registryNeedsUpdate(cr harborv1alpha1.Registry, current harborclient.Registry, desiredCredHash, desiredCACert string) bool {
	if cr.Spec.Name != "" && cr.Spec.Name != current.Name {
		return true
	}
	if cr.Spec.URL != current.URL {
		return true
	}
	if cr.Spec.Description != current.Description {
		return true
	}
	if !strings.EqualFold(cr.Spec.Type, current.Type) {
		return true
	}
	if cr.Spec.Insecure != current.Insecure {
		return true
	}
	if desiredCACert != "" && desiredCACert != current.CACertificate {
		return true
	}
	if desiredCredHash != "" && desiredCredHash != cr.Status.CredentialHash {
		return true
	}
	return false
}

func (r *RegistryReconciler) buildRegistryCredential(ctx context.Context, cr *harborv1alpha1.Registry) (*harborclient.RegistryCredential, string, string, error) {
	var caCert string
	if cr.Spec.CACertificateRef != nil {
		if cr.Spec.CACertificate != "" {
			return nil, "", "", fmt.Errorf("spec.caCertificate and spec.caCertificateRef are mutually exclusive")
		}
		secretValue, err := readSecretValue(ctx, r.Client, *cr.Spec.CACertificateRef, cr.Namespace, "ca.crt")
		if err != nil {
			return nil, "", "", fmt.Errorf("failed to read caCertificateRef: %w", err)
		}
		caCert = secretValue
	} else {
		caCert = cr.Spec.CACertificate
	}

	if cr.Spec.Credential == nil {
		if caCert == "" {
			return nil, "", "", nil
		}
		return nil, hashSecret("ca:" + caCert), caCert, nil
	}

	cred := cr.Spec.Credential
	if cred.AccessKeySecretRef.Name == "" {
		return nil, "", "", fmt.Errorf("spec.credential.accessKeySecretRef.name is required")
	}
	if cred.AccessSecretSecretRef.Name == "" {
		return nil, "", "", fmt.Errorf("spec.credential.accessSecretSecretRef.name is required")
	}
	accessKey, err := readSecretValue(ctx, r.Client, cred.AccessKeySecretRef, cr.Namespace, "access_key")
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to read access key secret: %w", err)
	}
	accessSecret, err := readSecretValue(ctx, r.Client, cred.AccessSecretSecretRef, cr.Namespace, "access_secret")
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to read access secret: %w", err)
	}
	credType := cred.Type
	if credType == "" {
		credType = "basic"
	}
	out := &harborclient.RegistryCredential{
		Type:         credType,
		AccessKey:    accessKey,
		AccessSecret: accessSecret,
	}
	hashInput := fmt.Sprintf("type=%s\nkey=%s\nsecret=%s\nca=%s", credType, accessKey, accessSecret, caCert)
	return out, hashSecret(hashInput), caCert, nil
}

func (r *RegistryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&harborv1alpha1.Registry{}).
		Named("registry").
		Complete(r)
}
