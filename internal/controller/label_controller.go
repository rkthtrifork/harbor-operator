package controller

import (
	"context"
	"fmt"
	"reflect"
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

type LabelReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	logger logr.Logger
}

// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=labels,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=labels/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=labels/finalizers,verbs=update
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=projects,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections,verbs=get;list;watch

func (r *LabelReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.FromContext(ctx).WithName(fmt.Sprintf("[Label:%s]", req.NamespacedName))

	var cr harborv1alpha1.Label
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
		if cr.Status.HarborLabelID == 0 {
			return nil
		}
		return hc.DeleteLabel(ctx, cr.Status.HarborLabelID)
	}); done {
		return ctrl.Result{}, err
	}

	if err := ensureFinalizer(ctx, r.Client, &cr); err != nil {
		return ctrl.Result{}, err
	}

	cr.Spec.Name = defaultString(cr.Spec.Name, cr.Name)

	scope := cr.Spec.Scope
	var projectID int
	if cr.Spec.ProjectRef != nil {
		if scope == "" {
			scope = "p"
		} else if scope != "p" {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, fmt.Errorf("spec.scope must be 'p' when projectRef is set"))
		}
		_, pid, err := resolveProject(ctx, r.Client, hc, cr.Namespace, cr.Spec.ProjectRef, "")
		if err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		}
		projectID = pid
	} else if scope == "" {
		scope = "g"
	}
	if scope == "p" && projectID == 0 {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, fmt.Errorf("project-scoped labels require projectRef"))
	}

	desired := harborclient.Label{
		Name:        cr.Spec.Name,
		Description: cr.Spec.Description,
		Color:       cr.Spec.Color,
		Scope:       scope,
		ProjectID:   projectID,
	}

	if cr.Status.HarborLabelID == 0 && cr.Spec.AllowTakeover {
		adopted, err := r.adoptExisting(ctx, hc, &cr, desired)
		if err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		}
		if adopted {
			r.logger.Info("Adopted existing label", "ID", cr.Status.HarborLabelID)
			return ctrl.Result{Requeue: true}, nil
		}
	}

	if cr.Status.HarborLabelID == 0 {
		id, err := hc.CreateLabel(ctx, desired)
		if err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		}
		cr.Status.HarborLabelID = id
		if err := setReadyStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, "Created", "Label created"); err != nil {
			return ctrl.Result{}, err
		}
		return returnWithDriftDetection(&cr.Spec.HarborSpecBase)
	}

	current, err := hc.GetLabel(ctx, cr.Status.HarborLabelID)
	if err != nil {
		if harborclient.IsNotFound(err) {
			cr.Status.HarborLabelID = 0
			if err := setReconcilingStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, "NotFound", "Label not found in Harbor"); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}

	if labelNeedsUpdate(desired, current) {
		if err := hc.UpdateLabel(ctx, cr.Status.HarborLabelID, desired); err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		}
		r.logger.Info("Updated label", "ID", cr.Status.HarborLabelID)
	}

	if err := setReadyStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, "Reconciled", "Label reconciled"); err != nil {
		return ctrl.Result{}, err
	}
	return returnWithDriftDetection(&cr.Spec.HarborSpecBase)
}

func (r *LabelReconciler) adoptExisting(ctx context.Context, hc *harborclient.Client, cr *harborv1alpha1.Label, desired harborclient.Label) (bool, error) {
	var projectID *int
	if desired.Scope == "p" {
		projectID = &desired.ProjectID
	}
	labels, err := hc.ListLabels(ctx, desired.Name, desired.Scope, projectID)
	if err != nil {
		return false, err
	}
	for _, label := range labels {
		if strings.EqualFold(label.Name, desired.Name) && label.Scope == desired.Scope {
			if desired.Scope != "p" || label.ProjectID == desired.ProjectID {
				cr.Status.HarborLabelID = label.ID
				return true, r.Status().Update(ctx, cr)
			}
		}
	}
	return false, nil
}

func (r *LabelReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&harborv1alpha1.Label{}).
		Named("label").
		Complete(r)
}

func labelNeedsUpdate(desired harborclient.Label, current *harborclient.Label) bool {
	if current == nil {
		return true
	}
	nd := normalizeLabel(desired)
	nc := normalizeLabel(*current)
	return !reflect.DeepEqual(nd, nc)
}

func normalizeLabel(in harborclient.Label) harborclient.Label {
	in.ID = 0
	in.CreationTime = ""
	in.UpdateTime = ""
	return in
}
