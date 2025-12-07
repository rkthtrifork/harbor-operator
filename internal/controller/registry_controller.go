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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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

	// Harbor client
	conn, err := getHarborConnection(ctx, r.Client, cr.Namespace, cr.Spec.HarborConnectionRef)
	if err != nil {
		SetReadyCondition(&cr.Status.Conditions, false, ReasonConnectionFailed, fmt.Sprintf("Failed to get HarborConnection: %v", err))
		SetStalledCondition(&cr.Status.Conditions, true, ReasonConnectionFailed, err.Error())
		_ = r.Status().Update(ctx, &cr)
		return ctrl.Result{}, err
	}
	user, pass, err := getHarborAuth(ctx, r.Client, conn)
	if err != nil {
		SetReadyCondition(&cr.Status.Conditions, false, ReasonConnectionFailed, fmt.Sprintf("Failed to get Harbor credentials: %v", err))
		SetStalledCondition(&cr.Status.Conditions, true, ReasonConnectionFailed, err.Error())
		_ = r.Status().Update(ctx, &cr)
		return ctrl.Result{}, err
	}
	hc := harborclient.New(conn.Spec.BaseURL, user, pass)

	// Deletion
	if !cr.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&cr, finalizerName) {
			if err := r.deleteRegistry(ctx, hc, &cr); err != nil {
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(&cr, finalizerName)
			_ = r.Update(ctx, &cr)
		}
		return ctrl.Result{}, nil
	}

	// Finalizer
	if !controllerutil.ContainsFinalizer(&cr, finalizerName) {
		controllerutil.AddFinalizer(&cr, finalizerName)
		_ = r.Update(ctx, &cr)
	}

	// Defaults & adoption
	if cr.Spec.Name == "" {
		cr.Spec.Name = cr.Name
	}

	if cr.Status.HarborRegistryID == 0 && cr.Spec.AllowTakeover {
		SetReconcilingCondition(&cr.Status.Conditions, true, ReasonAdopting, "Attempting to adopt existing registry")
		_ = r.Status().Update(ctx, &cr)
		if ok, err := r.adoptExisting(ctx, hc, &cr); err != nil {
			SetReadyCondition(&cr.Status.Conditions, false, ReasonReconcileError, fmt.Sprintf("Failed to adopt registry: %v", err))
			SetStalledCondition(&cr.Status.Conditions, true, ReasonReconcileError, err.Error())
			SetReconcilingCondition(&cr.Status.Conditions, false, ReasonReconcileError, "Adoption failed")
			_ = r.Status().Update(ctx, &cr)
			return ctrl.Result{}, err
		} else if ok {
			r.logger.Info("Adopted registry", "ID", cr.Status.HarborRegistryID)
		}
	}

	// Desired payload
	createReq := r.buildCreateReq(cr)

	// Create / Update
	if cr.Status.HarborRegistryID == 0 {
		SetReconcilingCondition(&cr.Status.Conditions, true, ReasonCreating, "Creating registry in Harbor")
		_ = r.Status().Update(ctx, &cr)
		id, err := hc.CreateRegistry(ctx, createReq)
		if err != nil {
			SetReadyCondition(&cr.Status.Conditions, false, ReasonReconcileError, fmt.Sprintf("Failed to create registry: %v", err))
			SetStalledCondition(&cr.Status.Conditions, true, ReasonReconcileError, err.Error())
			SetReconcilingCondition(&cr.Status.Conditions, false, ReasonReconcileError, "Creation failed")
			_ = r.Status().Update(ctx, &cr)
			return ctrl.Result{}, err
		}
		cr.Status.HarborRegistryID = id
		SetReadyCondition(&cr.Status.Conditions, true, ReasonReconcileSuccess, "Registry created successfully")
		SetReconcilingCondition(&cr.Status.Conditions, false, ReasonReconcileSuccess, "Reconciliation complete")
		SetStalledCondition(&cr.Status.Conditions, false, ReasonReconcileSuccess, "")
		_ = r.Status().Update(ctx, &cr)
		r.logger.Info("Created registry", "ID", id)
		return returnWithDriftDetection(&cr.Spec.HarborSpecBase)
	}

	current, err := hc.GetRegistryByID(ctx, cr.Status.HarborRegistryID)
	if err != nil {
		if harborclient.IsNotFound(err) {
			cr.Status.HarborRegistryID = 0
			SetReadyCondition(&cr.Status.Conditions, false, ReasonReconcileError, "Registry was deleted out-of-band")
			SetReconcilingCondition(&cr.Status.Conditions, true, ReasonReconcileError, "Recreating registry")
			_ = r.Status().Update(ctx, &cr)
			return ctrl.Result{Requeue: true}, nil
		}
		SetReadyCondition(&cr.Status.Conditions, false, ReasonReconcileError, fmt.Sprintf("Failed to get registry: %v", err))
		SetStalledCondition(&cr.Status.Conditions, true, ReasonReconcileError, err.Error())
		_ = r.Status().Update(ctx, &cr)
		return ctrl.Result{}, err
	}

	if registryNeedsUpdate(createReq, *current) {
		SetReconcilingCondition(&cr.Status.Conditions, true, ReasonUpdating, "Updating registry in Harbor")
		_ = r.Status().Update(ctx, &cr)
		if err := hc.UpdateRegistry(ctx, current.ID, createReq); err != nil {
			SetReadyCondition(&cr.Status.Conditions, false, ReasonReconcileError, fmt.Sprintf("Failed to update registry: %v", err))
			SetStalledCondition(&cr.Status.Conditions, true, ReasonReconcileError, err.Error())
			SetReconcilingCondition(&cr.Status.Conditions, false, ReasonReconcileError, "Update failed")
			_ = r.Status().Update(ctx, &cr)
			return ctrl.Result{}, err
		}
		r.logger.Info("Updated registry", "ID", current.ID)
	}
	SetReadyCondition(&cr.Status.Conditions, true, ReasonReconcileSuccess, "Registry reconciled successfully")
	SetReconcilingCondition(&cr.Status.Conditions, false, ReasonReconcileSuccess, "Reconciliation complete")
	SetStalledCondition(&cr.Status.Conditions, false, ReasonReconcileSuccess, "")
	_ = r.Status().Update(ctx, &cr)
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
	regs, err := hc.ListRegistries(ctx)
	if err != nil {
		return false, err
	}
	for _, rg := range regs {
		if strings.EqualFold(rg.Name, cr.Spec.Name) {
			cr.Status.HarborRegistryID = rg.ID
			return true, r.Status().Update(ctx, cr)
		}
	}
	return false, nil
}

func (r *RegistryReconciler) buildCreateReq(cr harborv1alpha1.Registry) harborclient.CreateRegistryRequest {
	desired := harborclient.CreateRegistryRequest{
		URL:         cr.Spec.URL,
		Name:        cr.Spec.Name,
		Description: cr.Spec.Description,
		Type:        cr.Spec.Type,
		Insecure:    cr.Spec.Insecure,
	}
	return desired
}

func registryNeedsUpdate(desired harborclient.CreateRegistryRequest, current harborclient.Registry) bool {
	return desired.URL != current.URL ||
		desired.Description != current.Description ||
		!strings.EqualFold(desired.Type, current.Type) ||
		desired.Insecure != current.Insecure
}

func (r *RegistryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&harborv1alpha1.Registry{}).
		Named("registry").
		Complete(r)
}
