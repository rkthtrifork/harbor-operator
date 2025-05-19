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
// UserReconciler
// -----------------------------------------------------------------------------

type UserReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	logger   logr.Logger
	recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=users,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=users/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections,verbs=get;list;watch

func (r *UserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.FromContext(ctx).WithName(fmt.Sprintf("[User:%s]", req.NamespacedName))

	//---------------------------------------------------------------------
	// Load CR
	//---------------------------------------------------------------------
	var cr harborv1alpha1.User
	if err := r.Get(ctx, req.NamespacedName, &cr); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	//---------------------------------------------------------------------
	// Mark Reconciling=True
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
	// Deletion
	//---------------------------------------------------------------------
	if !cr.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&cr, finalizerName) {
			if err := r.deleteUser(ctx, hc, &cr); err != nil {
				r.fail(&cr, "DeleteError", err.Error())
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(&cr, finalizerName)
			_ = r.Update(ctx, &cr)
			r.recorder.Event(&cr, corev1.EventTypeNormal, "Deleted", "User deleted from Harbor")
		}
		return ctrl.Result{}, nil
	}

	//---------------------------------------------------------------------
	// Finalizer
	//---------------------------------------------------------------------
	if !controllerutil.ContainsFinalizer(&cr, finalizerName) {
		controllerutil.AddFinalizer(&cr, finalizerName)
		_ = r.Update(ctx, &cr)
	}

	//---------------------------------------------------------------------
	// Defaults & adoption
	//---------------------------------------------------------------------
	if cr.Spec.Username == "" {
		cr.Spec.Username = cr.Name // in-memory only
	}
	if cr.Status.HarborUserID == 0 && cr.Spec.AllowTakeover {
		if ok, err := r.adoptExisting(ctx, hc, &cr); err != nil {
			r.fail(&cr, "AdoptionError", err.Error())
			return ctrl.Result{}, err
		} else if ok {
			r.logger.Info("Adopted user", "ID", cr.Status.HarborUserID)
			r.recorder.Event(&cr, corev1.EventTypeNormal, "Adopted",
				fmt.Sprintf("Existing user %q adopted (ID=%d)", cr.Spec.Username, cr.Status.HarborUserID))
		}
	}

	//---------------------------------------------------------------------
	// Desired payload
	//---------------------------------------------------------------------
	createReq := r.buildCreateReq(cr)

	//---------------------------------------------------------------------
	// Create / Update
	//---------------------------------------------------------------------
	if cr.Status.HarborUserID == 0 {
		id, err := hc.CreateUser(ctx, createReq)
		if err != nil {
			r.fail(&cr, "CreateError", err.Error())
			return ctrl.Result{}, err
		}
		cr.Status.HarborUserID = id
		_ = r.Status().Update(ctx, &cr)
		r.recorder.Event(&cr, corev1.EventTypeNormal, "Created",
			fmt.Sprintf("User created in Harbor (ID=%d)", id))

		r.markReady(&cr)
		_ = r.Status().Update(ctx, &cr)
		return returnWithDriftDetection(&cr.Spec.HarborSpecBase)
	}

	current, err := hc.GetUserByID(ctx, cr.Status.HarborUserID)
	if err != nil {
		if harborclient.IsNotFound(err) {
			cr.Status.HarborUserID = 0
			_ = r.Status().Update(ctx, &cr)
			return ctrl.Result{Requeue: true}, nil
		}
		r.fail(&cr, "GetError", err.Error())
		return ctrl.Result{}, err
	}

	if userNeedsUpdate(createReq, *current) {
		if err := hc.UpdateUser(ctx, current.UserID, createReq); err != nil {
			r.fail(&cr, "UpdateError", err.Error())
			return ctrl.Result{}, err
		}
		r.recorder.Event(&cr, corev1.EventTypeNormal, "Updated",
			fmt.Sprintf("User updated in Harbor (ID=%d)", current.UserID))
	}

	//---------------------------------------------------------------------
	// Success
	//---------------------------------------------------------------------
	r.markReady(&cr)
	_ = r.Status().Update(ctx, &cr)
	return returnWithDriftDetection(&cr.Spec.HarborSpecBase)
}

// -----------------------------------------------------------------------------
// Status helpers
// -----------------------------------------------------------------------------

func (r *UserReconciler) markReconciling(cr *harborv1alpha1.User) {
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

func (r *UserReconciler) markReady(cr *harborv1alpha1.User) {
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

func (r *UserReconciler) fail(cr *harborv1alpha1.User, reason, msg string) {
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
// CRUD helpers (unchanged logic)
// -----------------------------------------------------------------------------

func (r *UserReconciler) buildCreateReq(cr harborv1alpha1.User) harborclient.CreateUserRequest {
	return harborclient.CreateUserRequest{
		Email:    cr.Spec.Email,
		Realname: cr.Spec.Realname,
		Comment:  cr.Spec.Comment,
		Password: cr.Spec.Password,
		Username: cr.Spec.Username,
	}
}

func (r *UserReconciler) deleteUser(ctx context.Context, hc *harborclient.Client, cr *harborv1alpha1.User) error {
	if cr.Status.HarborUserID == 0 {
		return nil
	}
	err := hc.DeleteUser(ctx, cr.Status.HarborUserID)
	if harborclient.IsNotFound(err) {
		return nil
	}
	return err
}

func (r *UserReconciler) adoptExisting(ctx context.Context, hc *harborclient.Client, cr *harborv1alpha1.User) (bool, error) {
	users, err := hc.ListUsers(ctx, "username="+cr.Spec.Username)
	if err != nil {
		return false, err
	}
	for _, u := range users {
		if strings.EqualFold(u.Username, cr.Spec.Username) {
			cr.Status.HarborUserID = u.UserID
			return true, r.Status().Update(ctx, cr)
		}
	}
	return false, nil
}

func userNeedsUpdate(desired harborclient.CreateUserRequest, current harborclient.User) bool {
	return desired.Email != current.Email ||
		desired.Realname != current.Realname ||
		desired.Comment != current.Comment
}

// -----------------------------------------------------------------------------
// Setup
// -----------------------------------------------------------------------------

func (r *UserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("harbor-operator")
	return ctrl.NewControllerManagedBy(mgr).
		For(&harborv1alpha1.User{}).
		Named("user").
		Complete(r)
}
