// Copyright 2025 The Harbor-Operator Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
	"github.com/rkthtrifork/harbor-operator/internal/harborclient"
)

// -----------------------------------------------------------------------------
// RegistryReconciler
// -----------------------------------------------------------------------------

type RegistryReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	logger   logr.Logger
	recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=registries,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=registries/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections,verbs=get;list;watch

func (r *RegistryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.FromContext(ctx).WithName(fmt.Sprintf("[Registry:%s]", req.NamespacedName))

	//---------------------------------------------------------------------
	// Load CR
	//---------------------------------------------------------------------
	var cr harborv1alpha1.Registry
	if err := r.Get(ctx, req.NamespacedName, &cr); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	//---------------------------------------------------------------------
	// Mark Reconciling = True
	//---------------------------------------------------------------------
	r.markReconciling(&cr)
	_ = r.Status().Update(ctx, &cr)

	//---------------------------------------------------------------------
	// Harbor client
	//---------------------------------------------------------------------
	conn, err := getHarborConnection(ctx, r.Client, cr.Namespace, cr.Spec.HarborConnectionRef)
	if err != nil {
		r.fail(&cr, "NoConnection", err.Error())
		return ctrl.Result{}, err
	}
	user, pass, err := getHarborAuth(ctx, r.Client, conn)
	if err != nil {
		r.fail(&cr, "SecretError", err.Error())
		return ctrl.Result{}, err
	}
	hc := harborclient.New(conn.Spec.BaseURL, user, pass)

	//---------------------------------------------------------------------
	// Deletion flow
	//---------------------------------------------------------------------
	if !cr.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&cr, finalizerName) {
			if err := r.deleteRegistry(ctx, hc, &cr); err != nil {
				r.fail(&cr, "DeleteError", err.Error())
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(&cr, finalizerName)
			_ = r.Update(ctx, &cr)
			r.recorder.Event(&cr, corev1.EventTypeNormal, "Deleted",
				"Registry deleted from Harbor")
		}
		return ctrl.Result{}, nil
	}

	//---------------------------------------------------------------------
	// Ensure finalizer
	//---------------------------------------------------------------------
	if !controllerutil.ContainsFinalizer(&cr, finalizerName) {
		controllerutil.AddFinalizer(&cr, finalizerName)
		_ = r.Update(ctx, &cr)
	}

	//---------------------------------------------------------------------
	// Defaults & adoption
	//---------------------------------------------------------------------
	if cr.Spec.Name == "" {
		cr.Spec.Name = cr.Name // in-memory default; do not mutate Spec
	}
	if cr.Status.HarborRegistryID == 0 && cr.Spec.AllowTakeover {
		if ok, err := r.adoptExisting(ctx, hc, &cr); err != nil {
			r.fail(&cr, "AdoptionError", err.Error())
			return ctrl.Result{}, err
		} else if ok {
			r.logger.Info("Adopted registry", "ID", cr.Status.HarborRegistryID)
			r.recorder.Event(&cr, corev1.EventTypeNormal, "Adopted",
				fmt.Sprintf("Existing registry adopted (ID=%d)", cr.Status.HarborRegistryID))
		}
	}

	//---------------------------------------------------------------------
	// Desired payload
	//---------------------------------------------------------------------
	createReq := r.buildCreateReq(cr)

	//---------------------------------------------------------------------
	// Create path
	//---------------------------------------------------------------------
	if cr.Status.HarborRegistryID == 0 {
		id, err := hc.CreateRegistry(ctx, createReq)
		if err != nil {
			r.fail(&cr, "CreateError", err.Error())
			return ctrl.Result{}, err
		}
		cr.Status.HarborRegistryID = id
		_ = r.Status().Update(ctx, &cr)
		r.logger.Info("Created registry", "ID", id)
		r.recorder.Event(&cr, corev1.EventTypeNormal, "Created",
			fmt.Sprintf("Registry created in Harbor (ID=%d)", id))

		r.markReady(&cr)
		_ = r.Status().Update(ctx, &cr)
		return returnWithDriftDetection(&cr.Spec.HarborSpecBase)
	}

	//---------------------------------------------------------------------
	// Read current state
	//---------------------------------------------------------------------
	current, err := hc.GetRegistryByID(ctx, cr.Status.HarborRegistryID)
	if err != nil {
		if harborclient.IsNotFound(err) {
			cr.Status.HarborRegistryID = 0
			_ = r.Status().Update(ctx, &cr)
			return ctrl.Result{Requeue: true}, nil
		}
		r.fail(&cr, "GetError", err.Error())
		return ctrl.Result{}, err
	}

	//---------------------------------------------------------------------
	// Update path
	//---------------------------------------------------------------------
	if registryNeedsUpdate(createReq, *current) {
		if err := hc.UpdateRegistry(ctx, current.ID, createReq); err != nil {
			r.fail(&cr, "UpdateError", err.Error())
			return ctrl.Result{}, err
		}
		r.logger.Info("Updated registry", "ID", current.ID)
		r.recorder.Event(&cr, corev1.EventTypeNormal, "Updated",
			fmt.Sprintf("Registry updated in Harbor (ID=%d)", current.ID))
	}

	//---------------------------------------------------------------------
	// Success
	//---------------------------------------------------------------------
	r.markReady(&cr)
	_ = r.Status().Update(ctx, &cr)
	return returnWithDriftDetection(&cr.Spec.HarborSpecBase)
}

// -----------------------------------------------------------------------------
// Status / condition helpers
// -----------------------------------------------------------------------------

func (r *RegistryReconciler) markReconciling(cr *harborv1alpha1.Registry) {
	harborv1alpha1.SetStatusCondition(&cr.Status.Conditions, metav1.Condition{
		Type:    harborv1alpha1.ConditionReconciling,
		Status:  metav1.ConditionTrue,
		Reason:  "Reconciling",
		Message: "Reconciling resource",
	})
	harborv1alpha1.RemoveCondition(&cr.Status.Conditions, harborv1alpha1.ConditionStalled)
	harborv1alpha1.RemoveCondition(&cr.Status.Conditions, harborv1alpha1.ConditionReady)
	cr.Status.ObservedGeneration = cr.Generation
}

func (r *RegistryReconciler) markReady(cr *harborv1alpha1.Registry) {
	harborv1alpha1.SetStatusCondition(&cr.Status.Conditions, metav1.Condition{
		Type:    harborv1alpha1.ConditionReady,
		Status:  metav1.ConditionTrue,
		Reason:  "Reconciled",
		Message: "Resource is ready",
	})
	harborv1alpha1.RemoveCondition(&cr.Status.Conditions, harborv1alpha1.ConditionReconciling)
	harborv1alpha1.RemoveCondition(&cr.Status.Conditions, harborv1alpha1.ConditionStalled)
	cr.Status.ObservedGeneration = cr.Generation
}

func (r *RegistryReconciler) fail(cr *harborv1alpha1.Registry, reason, msg string) {
	harborv1alpha1.SetStatusCondition(&cr.Status.Conditions, metav1.Condition{
		Type:    harborv1alpha1.ConditionStalled,
		Status:  metav1.ConditionTrue,
		Reason:  reason,
		Message: msg,
	})
	harborv1alpha1.RemoveCondition(&cr.Status.Conditions, harborv1alpha1.ConditionReconciling)
	harborv1alpha1.RemoveCondition(&cr.Status.Conditions, harborv1alpha1.ConditionReady)
	cr.Status.ObservedGeneration = cr.Generation
	_ = r.Status().Update(context.TODO(), cr)
	r.recorder.Event(cr, corev1.EventTypeWarning, reason, msg)
}

// -----------------------------------------------------------------------------
// CRUD helpers (logic unchanged from your original implementation)
// -----------------------------------------------------------------------------

func (r *RegistryReconciler) deleteRegistry(ctx context.Context, hc *harborclient.Client,
	cr *harborv1alpha1.Registry) error {

	if cr.Status.HarborRegistryID == 0 {
		return nil
	}
	err := hc.DeleteRegistry(ctx, cr.Status.HarborRegistryID)
	if harborclient.IsNotFound(err) {
		return nil
	}
	return err
}

func (r *RegistryReconciler) adoptExisting(ctx context.Context, hc *harborclient.Client,
	cr *harborv1alpha1.Registry) (bool, error) {

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
	return harborclient.CreateRegistryRequest{
		URL:         cr.Spec.URL,
		Name:        cr.Spec.Name,
		Description: cr.Spec.Description,
		Type:        cr.Spec.Type,
		Insecure:    cr.Spec.Insecure,
	}
}

func registryNeedsUpdate(desired harborclient.CreateRegistryRequest, current harborclient.Registry) bool {
	return desired.URL != current.URL ||
		desired.Description != current.Description ||
		!strings.EqualFold(desired.Type, current.Type) ||
		desired.Insecure != current.Insecure
}

// -----------------------------------------------------------------------------
// Setup
// -----------------------------------------------------------------------------

func (r *RegistryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("harbor-operator")
	return ctrl.NewControllerManagedBy(mgr).
		For(&harborv1alpha1.Registry{}).
		Named("registry").
		Complete(r)
}
