package controller

import (
	"context"
	"fmt"
	"reflect"
	"strconv"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
	"github.com/rkthtrifork/harbor-operator/internal/harborclient"
)

type QuotaReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	logger logr.Logger
}

// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=quotas,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=quotas/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=quotas/finalizers,verbs=update
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=projects,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections;clusterharborconnections,verbs=get;list;watch

func (r *QuotaReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.FromContext(ctx).WithName(fmt.Sprintf("[Quota:%s]", req.NamespacedName))

	var cr harborv1alpha1.Quota
	if found, err := loadResource(ctx, r.Client, req.NamespacedName, &cr, r.logger); err != nil {
		return ctrl.Result{}, err
	} else if !found {
		return ctrl.Result{}, nil
	}

	if err := markReconcilingIfNeeded(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation); err != nil {
		return ctrl.Result{}, err
	}

	hc, err := getHarborClient(ctx, r.Client, cr.Namespace, cr.Spec.HarborConnectionRef)
	if err != nil {
		if done, finalErr := finalizeWithoutHarborConnection(ctx, r.Client, &cr, cr.Spec.GetDeletionPolicy(), false, err); done {
			return ctrl.Result{}, finalErr
		}
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}

	if done, err := finalizeIfDeleting(ctx, r.Client, &cr, nil); done {
		return ctrl.Result{}, err
	}

	if err := ensureFinalizer(ctx, r.Client, &cr); err != nil {
		return ctrl.Result{}, err
	}

	_, projectID, err := resolveProject(ctx, r.Client, hc, cr.Namespace, cr.Spec.ProjectRef, cr.Spec.ProjectNameOrID)
	if err != nil {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}
	if projectID == 0 {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, fmt.Errorf("unable to resolve project ID for quota"))
	}

	if cr.Status.HarborQuotaID == 0 {
		quotaID, err := r.findQuotaID(ctx, hc, projectID)
		if err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		}
		cr.Status.HarborQuotaID = quotaID
		if err := r.Status().Update(ctx, &cr); err != nil {
			return ctrl.Result{}, err
		}
	}

	current, err := hc.GetQuota(ctx, cr.Status.HarborQuotaID)
	if err != nil {
		if harborclient.IsNotFound(err) {
			return requeueOnRemoteNotFound(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, func() {
				cr.Status.HarborQuotaID = 0
			}, "Quota not found in Harbor")
		}
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}

	if quotaNeedsUpdate(cr.Spec.Hard, current) {
		if err := hc.UpdateQuota(ctx, cr.Status.HarborQuotaID, cr.Spec.Hard); err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		}
		r.logger.Info("Updated quota", "ID", cr.Status.HarborQuotaID)
	}

	if err := setReadyStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, "Reconciled", "Quota reconciled"); err != nil {
		return ctrl.Result{}, err
	}
	return returnWithDriftDetection(&cr.Spec.HarborSpecBase)
}

func (r *QuotaReconciler) SetupWithManager(mgr ctrl.Manager) error {
	builder, err := setupHarborBackedController(
		mgr,
		&harborv1alpha1.Quota{},
		func() client.ObjectList { return &harborv1alpha1.QuotaList{} },
		func(obj client.Object) harborv1alpha1.HarborConnectionReference {
			return obj.(*harborv1alpha1.Quota).Spec.HarborConnectionRef
		},
		"quota",
	)
	if err != nil {
		return err
	}
	return builder.Complete(r)
}

func (r *QuotaReconciler) findQuotaID(ctx context.Context, hc *harborclient.Client, projectID int) (int, error) {
	quotas, err := hc.ListQuotas(ctx, "project", strconv.Itoa(projectID))
	if err != nil {
		return 0, err
	}
	if len(quotas) == 0 {
		return 0, fmt.Errorf("no quota found for project %d", projectID)
	}
	return quotas[0].ID, nil
}

func quotaNeedsUpdate(desired map[string]int64, current *harborclient.Quota) bool {
	if current == nil {
		return true
	}
	if desired == nil {
		return false
	}
	return !reflect.DeepEqual(desired, current.Hard)
}
