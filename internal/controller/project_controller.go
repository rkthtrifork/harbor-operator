package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

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

// -----------------------------------------------------------------------------
// Reconciler
// -----------------------------------------------------------------------------

type ProjectReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	logger logr.Logger
}

// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=projects,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=projects/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections,verbs=get;list;watch

func (r *ProjectReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.FromContext(ctx).WithName(fmt.Sprintf("[Project:%s]", req.NamespacedName))

	//---------------------------------------------------------------------------
	// 1. Load CR
	//---------------------------------------------------------------------------
	var projCR harborv1alpha1.Project
	if err := r.Get(ctx, req.NamespacedName, &projCR); err != nil {
		if errors.IsNotFound(err) {
			r.logger.V(1).Info("Resource disappeared")
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	//---------------------------------------------------------------------------
	// 2. Resolve Harbor connection + typed client
	//---------------------------------------------------------------------------
	conn, err := getHarborConnection(ctx, r.Client, projCR.Namespace, projCR.Spec.HarborConnectionRef)
	if err != nil {
		return ctrl.Result{}, err
	}
	user, pass, err := getHarborAuth(ctx, r.Client, conn)
	if err != nil {
		return ctrl.Result{}, err
	}
	hc := harborclient.New(conn.Spec.BaseURL, user, pass)

	//---------------------------------------------------------------------------
	// 3. Handle deletion
	//---------------------------------------------------------------------------
	if !projCR.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&projCR, finalizerName) {
			if err := r.deleteProject(ctx, hc, &projCR); err != nil {
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(&projCR, finalizerName)
			if err := r.Update(ctx, &projCR); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	//---------------------------------------------------------------------------
	// 4. Ensure finalizer
	//---------------------------------------------------------------------------
	if !controllerutil.ContainsFinalizer(&projCR, finalizerName) {
		controllerutil.AddFinalizer(&projCR, finalizerName)
		if err := r.Update(ctx, &projCR); err != nil {
			return ctrl.Result{}, err
		}
	}

	//---------------------------------------------------------------------------
	// 5. Defaults & adoption
	//---------------------------------------------------------------------------
	if projCR.Spec.Name == "" {
		projCR.Spec.Name = projCR.Name
	}

	if projCR.Status.HarborProjectID == 0 && projCR.Spec.AllowTakeover {
		if adopted, err := r.adoptExisting(ctx, hc, &projCR); err != nil {
			return ctrl.Result{}, err
		} else if adopted {
			r.logger.Info("Adopted existing project",
				"Name", projCR.Spec.Name, "ID", projCR.Status.HarborProjectID)
		}
	}

	//---------------------------------------------------------------------------
	// 6. Desired payload
	//---------------------------------------------------------------------------
	desired, err := r.buildCreateReq(ctx, hc, &projCR)
	if err != nil {
		return ctrl.Result{}, err
	}

	//---------------------------------------------------------------------------
	// 7. Create / Update path
	//---------------------------------------------------------------------------
	if projCR.Status.HarborProjectID == 0 {
		// create
		newID, err := hc.CreateProject(ctx, desired)
		if err != nil {
			return ctrl.Result{}, err
		}
		projCR.Status.HarborProjectID = newID
		if err := r.Status().Update(ctx, &projCR); err != nil {
			return ctrl.Result{}, err
		}
		r.logger.Info("Created project", "ID", newID)
		return returnWithDriftDetection(&projCR.Spec.HarborSpecBase)
	}

	// update
	current, err := hc.GetProjectByID(ctx, projCR.Status.HarborProjectID)
	if err != nil {
		if harborclient.IsNotFound(err) {
			// It was deleted out-of-band → clear status and requeue immediately
			projCR.Status.HarborProjectID = 0
			_ = r.Status().Update(ctx, &projCR)
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	}

	if projectNeedsUpdate(desired, *current) {
		if err := hc.UpdateProject(ctx, current.ProjectID, desired); err != nil {
			return ctrl.Result{}, err
		}
		r.logger.Info("Updated project", "ID", current.ProjectID)
	}
	return returnWithDriftDetection(&projCR.Spec.HarborSpecBase)
}

// -----------------------------------------------------------------------------
// Helpers
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

// -----------------------------------------------------------------------------
// Build desired payload
// -----------------------------------------------------------------------------

func (r *ProjectReconciler) buildCreateReq(ctx context.Context,
	hc *harborclient.Client, cr *harborv1alpha1.Project) (harborclient.CreateProjectRequest, error) {

	// Metadata ----------------------------------
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

	// CVE allow-list ----------------------------
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

	// Storage limit -----------------------------
	var storageLimit *int
	if cr.Spec.StorageLimit != 0 {
		storageLimit = &cr.Spec.StorageLimit
	}

	// Registry lookup (optional) ----------------
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

// -----------------------------------------------------------------------------
// Update comparison
// -----------------------------------------------------------------------------

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

// -----------------------------------------------------------------------------
// Setup
// -----------------------------------------------------------------------------

func (r *ProjectReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&harborv1alpha1.Project{}).
		Named("project").
		Complete(r)
}
