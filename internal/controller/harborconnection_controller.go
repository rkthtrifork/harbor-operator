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
	"net/url"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
	"github.com/rkthtrifork/harbor-operator/internal/harborclient"
)

// -----------------------------------------------------------------------------
// HarborConnectionReconciler
// -----------------------------------------------------------------------------

// HarborConnectionReconciler reconciles a HarborConnection object.
type HarborConnectionReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	logger   logr.Logger
	recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *HarborConnectionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.FromContext(ctx).WithName(fmt.Sprintf("[HarborConnection:%s]", req.NamespacedName))

	// ---------------------------------------------------------------------
	// Load CR
	// ---------------------------------------------------------------------
	var conn harborv1alpha1.HarborConnection
	if err := r.Get(ctx, req.NamespacedName, &conn); err != nil {
		if errors.IsNotFound(err) {
			// CR deleted – nothing to do.
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// ---------------------------------------------------------------------
	// Mark Reconciling=True
	// ---------------------------------------------------------------------
	r.markReconciling(&conn)
	_ = r.Status().Update(ctx, &conn)

	// ---------------------------------------------------------------------
	// Validate BaseURL
	// ---------------------------------------------------------------------
	if err := r.validateBaseURL(&conn); err != nil {
		r.markStalled(&conn, "InvalidBaseURL", err.Error())
		_ = r.Status().Update(ctx, &conn)
		r.recorder.Event(&conn, corev1.EventTypeWarning, "InvalidBaseURL", err.Error())
		return ctrl.Result{}, err
	}

	// ---------------------------------------------------------------------
	// Obtain credentials (may be empty for anonymous ping)
	// ---------------------------------------------------------------------
	user, pass, err := getHarborAuth(ctx, r.Client, &conn)
	if err != nil {
		r.markStalled(&conn, "SecretError", err.Error())
		_ = r.Status().Update(ctx, &conn)
		r.recorder.Event(&conn, corev1.EventTypeWarning, "SecretError", err.Error())
		return ctrl.Result{}, err
	}

	// ---------------------------------------------------------------------
	// Connectivity / authentication check
	// ---------------------------------------------------------------------
	if conn.Spec.Credentials == nil {
		err = r.checkNonAuthConnectivity(ctx, &conn)
	} else {
		err = r.checkAuthenticatedConnection(ctx, &conn, user, pass)
	}

	if err != nil {
		r.markStalled(&conn, "ConnectionError", err.Error())
		_ = r.Status().Update(ctx, &conn)
		r.recorder.Event(&conn, corev1.EventTypeWarning, "ConnectionError", err.Error())
		return ctrl.Result{}, err
	}

	// ---------------------------------------------------------------------
	// Success – Ready=True
	// ---------------------------------------------------------------------
	r.markReady(&conn)
	if err := r.Status().Update(ctx, &conn); err != nil {
		return ctrl.Result{}, err
	}
	r.recorder.Event(&conn, corev1.EventTypeNormal, "Reconciled", "Harbor endpoint verified")

	return ctrl.Result{}, nil
}

// -----------------------------------------------------------------------------
// Status helpers
// -----------------------------------------------------------------------------

func (r *HarborConnectionReconciler) markReconciling(cr *harborv1alpha1.HarborConnection) {
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

func (r *HarborConnectionReconciler) markStalled(cr *harborv1alpha1.HarborConnection, reason, msg string) {
	harborv1alpha1.SetStatusCondition(&cr.Status.Conditions, metav1.Condition{
		Type:    harborv1alpha1.ConditionStalled,
		Status:  metav1.ConditionTrue,
		Reason:  reason,
		Message: msg,
	})
	harborv1alpha1.RemoveCondition(&cr.Status.Conditions, harborv1alpha1.ConditionReconciling)
	harborv1alpha1.RemoveCondition(&cr.Status.Conditions, harborv1alpha1.ConditionReady)
	cr.Status.ObservedGeneration = cr.Generation
}

func (r *HarborConnectionReconciler) markReady(cr *harborv1alpha1.HarborConnection) {
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

// -----------------------------------------------------------------------------
// Validation & connectivity helpers
// -----------------------------------------------------------------------------

// validateBaseURL verifies that the BaseURL is a valid URL and includes a scheme.
func (r *HarborConnectionReconciler) validateBaseURL(conn *harborv1alpha1.HarborConnection) error {
	parsed, err := url.Parse(conn.Spec.BaseURL)
	if err != nil {
		return err
	}
	if parsed.Scheme == "" {
		return fmt.Errorf("baseURL %q is missing a protocol scheme", conn.Spec.BaseURL)
	}
	return nil
}

func (r *HarborConnectionReconciler) checkNonAuthConnectivity(ctx context.Context, conn *harborv1alpha1.HarborConnection) error {
	hc := harborclient.New(conn.Spec.BaseURL, "", "")
	return hc.Ping(ctx)
}

func (r *HarborConnectionReconciler) checkAuthenticatedConnection(
	ctx context.Context,
	conn *harborv1alpha1.HarborConnection,
	username, password string,
) error {
	hc := harborclient.New(conn.Spec.BaseURL, username, password)
	_, err := hc.GetCurrentUser(ctx)
	return err
}

// -----------------------------------------------------------------------------
// Setup
// -----------------------------------------------------------------------------

// SetupWithManager sets up the controller with the Manager.
func (r *HarborConnectionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("harbor-operator")
	return ctrl.NewControllerManagedBy(mgr).
		For(&harborv1alpha1.HarborConnection{}).
		Named("harborconnection").
		Complete(r)
}
