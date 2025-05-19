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
	"time"

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
// ProjectReconciler
// -----------------------------------------------------------------------------

type ProjectReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	logger   logr.Logger
	recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=projects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=projects/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections,verbs=get;list;watch

func (r *ProjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.FromContext(ctx).WithName(fmt.Sprintf("[Project:%s]", req.NamespacedName))

	//---------------------------------------------------------------------
	// Load CR
	//---------------------------------------------------------------------
	var cr harborv1alpha1.Project
	if err := r.Get(ctx, req.NamespacedName, &cr); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	//------------------------------------------------------------------------
	// Mark Reconciling=True
	//------------------------------------------------------------------------
	r.markReconciling(&cr)
	_ = r.Status().Update(ctx, &cr)

	//---------------------------------------------------------------------
	// Resolve Harbor connection + typed client
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
	// Handle deletion
	//---------------------------------------------------------------------
	if !cr.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&cr, finalizerName) {
			if err := r.deleteProject(ctx, hc, &cr); err != nil {
				r.fail(&cr, "DeleteError", err.Error())
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(&cr, finalizerName)
			_ = r.Update(ctx, &cr)
			r.recorder.Event(&cr, corev1.EventTypeNormal, "Deleted",
				"Project deleted from Harbor")
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
		cr.Spec.Name = cr.Name // in-memory only, do NOT mutate on the API server
	}
	if cr.Status.HarborProjectID == 0 && cr.Spec.AllowTakeover {
		if adopted, err := r.adoptExisting(ctx, hc, &cr); err != nil {
			r.fail(&cr, "AdoptionError", err.Error())
			return ctrl.Result{}, err
		} else if adopted {
			r.logger.Info("Adopted existing project",
				"Name", cr.Spec.Name, "ID", cr.Status.HarborProjectID)
			r.recorder.Event(&cr, corev1.EventTypeNormal, "Adopted",
				fmt.Sprintf("Existing project %q adopted (ID=%d)",
					cr.Spec.Name, cr.Status.HarborProjectID))
		}
	}

	//---------------------------------------------------------------------
	// Desired payload
	//---------------------------------------------------------------------
	createReq, err := r.buildCreateReq(ctx, hc, &cr)
	if err != nil {
		r.fail(&cr, "BuildRequestError", err.Error())
		return ctrl.Result{}, err
	}

	//---------------------------------------------------------------------
	// Create path
	//---------------------------------------------------------------------
	if cr.Status.HarborProjectID == 0 {
		newID, err := hc.CreateProject(ctx, createReq)
		if err != nil {
			r.fail(&cr, "CreateError", err.Error())
			return ctrl.Result{}, err
		}
		cr.Status.HarborProjectID = newID
		_ = r.Status().Update(ctx, &cr)
		r.logger.Info("Created project", "ID", newID)
		r.recorder.Event(&cr, corev1.EventTypeNormal, "Created",
			fmt.Sprintf("Project created in Harbor (ID=%d)", newID))

		r.markReady(&cr)
		_ = r.Status().Update(ctx, &cr)
		return returnWithDriftDetection(&cr.Spec.HarborSpecBase)
	}

	//---------------------------------------------------------------------
	// Read current state
	//---------------------------------------------------------------------
	current, err := hc.GetProjectByID(ctx, cr.Status.HarborProjectID)
	if err != nil {
		if harborclient.IsNotFound(err) {
			// Deleted out-of-band → clear status and requeue.
			cr.Status.HarborProjectID = 0
			_ = r.Status().Update(ctx, &cr)
			return ctrl.Result{Requeue: true}, nil
		}
		r.fail(&cr, "GetError", err.Error())
		return ctrl.Result{}, err
	}

	//---------------------------------------------------------------------
	// Update path (if needed)
	//---------------------------------------------------------------------
	if projectNeedsUpdate(createReq, *current) {
		if err := hc.UpdateProject(ctx, current.ProjectID, createReq); err != nil {
			r.fail(&cr, "UpdateError", err.Error())
			return ctrl.Result{}, err
		}
		r.logger.Info("Updated project", "ID", current.ProjectID)
		r.recorder.Event(&cr, corev1.EventTypeNormal, "Updated",
			fmt.Sprintf("Project updated in Harbor (ID=%d)", current.ProjectID))
	}

	//---------------------------------------------------------------------
	// Success!
	//---------------------------------------------------------------------
	r.markReady(&cr)
	_ = r.Status().Update(ctx, &cr)
	return returnWithDriftDetection(&cr.Spec.HarborSpecBase)
}

// -----------------------------------------------------------------------------
// Condition / Status helpers
// -----------------------------------------------------------------------------

func (r *ProjectReconciler) markReconciling(cr *harborv1alpha1.Project) {
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

func (r *ProjectReconciler) markReady(cr *harborv1alpha1.Project) {
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

func (r *ProjectReconciler) fail(cr *harborv1alpha1.Project, reason, msg string) {
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
// Deletion helpers (unchanged behaviour)
// -----------------------------------------------------------------------------

func (r *ProjectReconciler) deleteProject(ctx context.Context, hc *harborclient.Client,
	cr *harborv1alpha1.Project) error {

	if cr.Status.HarborProjectID == 0 {
		return nil
	}
	err := hc.DeleteProject(ctx, cr.Status.HarborProjectID)
	if harborclient.IsNotFound(err) {
		r.logger.V(1).Info("Project already gone", "ID", cr.Status.HarborProjectID)
		return nil
	}
	return err
}

// adoption by name
func (r *ProjectReconciler) adoptExisting(ctx context.Context, hc *harborclient.Client,
	cr *harborv1alpha1.Project) (bool, error) {

	projects, err := hc.ListProjects(ctx)
	if err != nil {
		return false, err
	}
	for _, p := range projects {
		if strings.EqualFold(p.Name, cr.Spec.Name) {
			cr.Status.HarborProjectID = p.ProjectID
			return true, r.Status().Update(ctx, cr)
		}
	}
	return false, nil
}

func (r *ProjectReconciler) buildCreateReq(ctx context.Context, hc *harborclient.Client, cr *harborv1alpha1.Project) (harborclient.CreateProjectRequest, error) {
	var meta harborclient.ProjectMetadata
	if m := cr.Spec.Metadata; m != nil {
		meta = harborclient.ProjectMetadata{
			Public:                   m.Public,
			EnableContentTrust:       m.EnableContentTrust,
			EnableContentTrustCosign: m.EnableContentTrustCosign,
			PreventVul:               m.PreventVul,
			Severity:                 m.Severity,
			AutoScan:                 m.AutoScan,
			AutoSBOMGeneration:       m.AutoSBOMGeneration,
			ReuseSysCVEAllowlist:     m.ReuseSysCVEAllowlist,
			RetentionID:              m.RetentionID,
			ProxySpeedKB:             m.ProxySpeedKB,
		}
	}

	var allow harborclient.CVEAllowlist
	if a := cr.Spec.CVEAllowlist; a != nil {
		allow.ID = a.ID
		allow.ProjectID = a.ProjectID
		allow.ExpiresAt = a.ExpiresAt
		allow.CreationTime = a.CreationTime.UTC().Format(time.RFC3339)
		allow.UpdateTime = a.UpdateTime.UTC().Format(time.RFC3339)
		allow.Items = make([]harborclient.CVEAllowlistItem, len(a.Items))
		for i, item := range a.Items {
			allow.Items[i].CveID = item.CveID
		}
	}

	var storageLimit *int64
	if cr.Spec.StorageLimit != nil {
		storageLimit = cr.Spec.StorageLimit
	}

	var registryID *int
	if rn := cr.Spec.RegistryName; rn != "" {
		regs, err := hc.ListRegistries(ctx)
		if err != nil {
			return harborclient.CreateProjectRequest{}, err
		}
		for _, reg := range regs {
			if strings.EqualFold(reg.Name, rn) {
				registryID = &reg.ID
				break
			}
		}
		if registryID == nil {
			return harborclient.CreateProjectRequest{},
				fmt.Errorf("registry %q not found in Harbor", rn)
		}
	}

	return harborclient.CreateProjectRequest{
		ProjectName:  cr.Spec.Name,
		Public:       cr.Spec.Public,
		Owner:        cr.Spec.Owner,
		Metadata:     meta,
		CVEAllowlist: allow,
		StorageLimit: storageLimit,
		RegistryID:   registryID,
	}, nil
}

func projectNeedsUpdate(desired harborclient.CreateProjectRequest,
	current harborclient.Project) bool {

	if desired.ProjectName != current.Name {
		return true
	}

	wantPub := "false"
	if desired.Public {
		wantPub = "true"
	}
	if wantPub != current.Metadata.Public {
		return true
	}

	if !strings.EqualFold(desired.Owner, current.OwnerName) {
		return true
	}

	if desired.RegistryID == nil && current.RegistryID != 0 {
		return true
	}
	if desired.RegistryID != nil && *desired.RegistryID != current.RegistryID {
		return true
	}

	mw := desired.Metadata
	mc := current.Metadata
	switch {
	case mw.EnableContentTrust != mc.EnableContentTrust,
		mw.EnableContentTrustCosign != mc.EnableContentTrustCosign,
		mw.PreventVul != mc.PreventVul,
		mw.Severity != mc.Severity,
		mw.AutoScan != mc.AutoScan,
		mw.AutoSBOMGeneration != mc.AutoSBOMGeneration,
		mw.ReuseSysCVEAllowlist != mc.ReuseSysCVEAllowlist,
		mw.RetentionID != mc.RetentionID,
		mw.ProxySpeedKB != mc.ProxySpeedKB:
		return true
	}

	aw := desired.CVEAllowlist
	ac := current.CVEAllowlist
	if aw.ID != ac.ID ||
		aw.ProjectID != ac.ProjectID ||
		aw.ExpiresAt != ac.ExpiresAt ||
		len(aw.Items) != len(ac.Items) {
		return true
	}
	for i := range aw.Items {
		if aw.Items[i].CveID != ac.Items[i].CveID {
			return true
		}
	}
	return false
}

func (r *ProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("harbor-operator")
	return ctrl.NewControllerManagedBy(mgr).
		For(&harborv1alpha1.Project{}).
		Named("project").
		Complete(r)
}
