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

type ScannerRegistrationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	logger logr.Logger
}

// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=scannerregistrations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=scannerregistrations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=scannerregistrations/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections,verbs=get;list;watch

func (r *ScannerRegistrationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.FromContext(ctx).WithName(fmt.Sprintf("[ScannerRegistration:%s]", req.NamespacedName))

	var cr harborv1alpha1.ScannerRegistration
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

	hc, err := getHarborClient(ctx, r.Client, cr.Namespace, cr.Spec.HarborConnectionRef)
	if err != nil {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}

	if done, err := finalizeIfDeleting(ctx, r.Client, &cr, func() error {
		if cr.Status.HarborScannerID == "" {
			return nil
		}
		return hc.DeleteScanner(ctx, cr.Status.HarborScannerID)
	}); done {
		return ctrl.Result{}, err
	}

	if err := ensureFinalizer(ctx, r.Client, &cr); err != nil {
		return ctrl.Result{}, err
	}

	cr.Spec.Name = defaultString(cr.Spec.Name, cr.Name)

	credential, credentialHash, err := r.resolveCredential(ctx, &cr)
	if err != nil {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}

	reqBody := harborclient.ScannerRegistrationReq{
		Name:             cr.Spec.Name,
		Description:      cr.Spec.Description,
		URL:              cr.Spec.URL,
		Auth:             cr.Spec.Auth,
		AccessCredential: credential,
		SkipCertVerify:   cr.Spec.SkipCertVerify,
		UseInternalAddr:  cr.Spec.UseInternalAddr,
		Disabled:         cr.Spec.Disabled,
	}

	if cr.Status.HarborScannerID == "" && cr.Spec.AllowTakeover {
		adopted, err := r.adoptExisting(ctx, hc, &cr)
		if err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		}
		if adopted {
			r.logger.Info("Adopted existing scanner registration", "ID", cr.Status.HarborScannerID)
			return ctrl.Result{Requeue: true}, nil
		}
	}

	if cr.Status.HarborScannerID == "" {
		id, err := hc.CreateScanner(ctx, reqBody)
		if err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		}
		cr.Status.HarborScannerID = id
		cr.Status.CredentialHash = credentialHash
		if err := setReadyStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, "Created", "Scanner registration created"); err != nil {
			return ctrl.Result{}, err
		}
		return returnWithDriftDetection(&cr.Spec.HarborSpecBase)
	}

	current, err := hc.GetScanner(ctx, cr.Status.HarborScannerID)
	if err != nil {
		if harborclient.IsNotFound(err) {
			cr.Status.HarborScannerID = ""
			if err := setReconcilingStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, "NotFound", "Scanner registration not found in Harbor"); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}

	statusChanged := false
	if scannerNeedsUpdate(reqBody, current) || (credentialHash != "" && credentialHash != cr.Status.CredentialHash) {
		if err := hc.UpdateScanner(ctx, cr.Status.HarborScannerID, reqBody); err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		}
		r.logger.Info("Updated scanner registration", "ID", cr.Status.HarborScannerID)
		if credentialHash != "" && credentialHash != cr.Status.CredentialHash {
			cr.Status.CredentialHash = credentialHash
			statusChanged = true
		}
	}

	if cr.Spec.Default && !current.IsDefault {
		if err := hc.SetDefaultScanner(ctx, cr.Status.HarborScannerID, true); err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		}
		r.logger.Info("Set scanner registration as default", "ID", cr.Status.HarborScannerID)
	}

	condChanged := markReady(&cr.Status.HarborStatusBase, cr.Generation, "Reconciled", "Scanner registration reconciled")
	if statusChanged || condChanged {
		if err := r.Status().Update(ctx, &cr); err != nil {
			return ctrl.Result{}, err
		}
	}
	return returnWithDriftDetection(&cr.Spec.HarborSpecBase)
}

func (r *ScannerRegistrationReconciler) resolveCredential(ctx context.Context, cr *harborv1alpha1.ScannerRegistration) (string, string, error) {
	if cr.Spec.AccessCredential != "" && cr.Spec.AccessCredentialSecretRef != nil {
		return "", "", fmt.Errorf("spec.accessCredential and spec.accessCredentialSecretRef are mutually exclusive")
	}
	credential := cr.Spec.AccessCredential
	if cr.Spec.AccessCredentialSecretRef != nil {
		value, err := readSecretValue(ctx, r.Client, *cr.Spec.AccessCredentialSecretRef, cr.Namespace, "accessCredential")
		if err != nil {
			return "", "", fmt.Errorf("failed to read accessCredentialSecretRef: %w", err)
		}
		credential = value
	}
	if credential == "" {
		return "", "", nil
	}
	return credential, hashParts(credential), nil
}

func (r *ScannerRegistrationReconciler) adoptExisting(ctx context.Context, hc *harborclient.Client, cr *harborv1alpha1.ScannerRegistration) (bool, error) {
	registrations, err := hc.ListScanners(ctx)
	if err != nil {
		return false, err
	}
	for _, reg := range registrations {
		if strings.EqualFold(reg.Name, cr.Spec.Name) {
			cr.Status.HarborScannerID = reg.UUID
			return true, r.Status().Update(ctx, cr)
		}
	}
	return false, nil
}

func (r *ScannerRegistrationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&harborv1alpha1.ScannerRegistration{}).
		Named("scannerregistration").
		Complete(r)
}

func scannerNeedsUpdate(desired harborclient.ScannerRegistrationReq, current *harborclient.ScannerRegistration) bool {
	if current == nil {
		return true
	}
	return desired.Name != current.Name ||
		desired.Description != current.Description ||
		desired.URL != current.URL ||
		desired.Auth != current.Auth ||
		desired.SkipCertVerify != current.SkipCertVerify ||
		desired.UseInternalAddr != current.UseInternalAddr ||
		desired.Disabled != current.Disabled
}
