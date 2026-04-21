package controller

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
	"github.com/rkthtrifork/harbor-operator/internal/harborclient"
)

type PurgeAuditScheduleReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	logger logr.Logger
}

// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=purgeauditschedules,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=purgeauditschedules/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=purgeauditschedules/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections;clusterharborconnections,verbs=get;list;watch

func (r *PurgeAuditScheduleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.FromContext(ctx).WithName(fmt.Sprintf("[PurgeAuditSchedule:%s]", req.NamespacedName))

	var cr harborv1alpha1.PurgeAuditSchedule
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

	if done, err := finalizeIfDeleting(ctx, r.Client, &cr, cr.Spec.GetDeletionPolicy(), nil); done {
		return ctrl.Result{}, err
	}

	if err := ensureFinalizer(ctx, r.Client, &cr); err != nil {
		return ctrl.Result{}, err
	}
	if err := ensurePurgeAuditScheduleSingletonOwner(ctx, r.Client, &cr); err != nil {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}

	params := map[string]any{}
	if cr.Spec.Parameters.AuditRetentionHour != 0 {
		params["audit_retention_hour"] = cr.Spec.Parameters.AuditRetentionHour
	}
	if cr.Spec.Parameters.IncludeEventTypes != "" {
		params["include_event_types"] = cr.Spec.Parameters.IncludeEventTypes
	}
	if cr.Spec.Parameters.DryRun {
		params["dry_run"] = cr.Spec.Parameters.DryRun
	}

	sched := harborclient.Schedule{
		Schedule: harborclient.ScheduleObj{
			Type: cr.Spec.Schedule.Type,
			Cron: cr.Spec.Schedule.Cron,
		},
		Parameters: params,
	}
	if sched.Schedule.Type != harborv1alpha1.ScheduleTypeManual && sched.Schedule.Type != harborv1alpha1.ScheduleTypeNone && sched.Schedule.Cron == "" {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, fmt.Errorf("schedule.cron is required for schedule type %q", sched.Schedule.Type))
	}
	hash := hashParts(
		fmt.Sprintf("type=%s", cr.Spec.Schedule.Type),
		fmt.Sprintf("cron=%s", cr.Spec.Schedule.Cron),
		fmt.Sprintf("retention=%d", cr.Spec.Parameters.AuditRetentionHour),
		fmt.Sprintf("include=%s", cr.Spec.Parameters.IncludeEventTypes),
		fmt.Sprintf("dry=%t", cr.Spec.Parameters.DryRun),
	)

	if cr.Status.LastAppliedScheduleHash == "" {
		if err := hc.CreatePurgeSchedule(ctx, sched); err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		}
		cr.Status.LastAppliedScheduleHash = hash
		if err := setReadyStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, "Created", "Purge audit schedule created"); err != nil {
			return ctrl.Result{}, err
		}
		return returnWithDriftDetection(&cr.Spec.HarborSpecBase)
	}

	statusChanged := false
	if cr.Status.LastAppliedScheduleHash != hash {
		if err := hc.UpdatePurgeSchedule(ctx, sched); err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		}
		cr.Status.LastAppliedScheduleHash = hash
		statusChanged = true
		r.logger.Info("Updated purge audit schedule")
	}

	condChanged := markReady(&cr.Status.HarborStatusBase, cr.Generation, "Reconciled", "Purge audit schedule reconciled")
	if statusChanged || condChanged {
		if err := r.Status().Update(ctx, &cr); err != nil {
			return ctrl.Result{}, err
		}
	}
	return returnWithDriftDetection(&cr.Spec.HarborSpecBase)
}

func (r *PurgeAuditScheduleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	builder, err := setupHarborBackedController(
		mgr,
		&harborv1alpha1.PurgeAuditSchedule{},
		func() client.ObjectList { return &harborv1alpha1.PurgeAuditScheduleList{} },
		func(obj client.Object) harborv1alpha1.HarborConnectionReference {
			return obj.(*harborv1alpha1.PurgeAuditSchedule).Spec.HarborConnectionRef
		},
		"purgeaudit",
	)
	if err != nil {
		return err
	}
	return builder.Complete(r)
}
