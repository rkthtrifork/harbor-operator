package controller

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
	"github.com/rkthtrifork/harbor-operator/internal/harborclient"
)

type ScanAllScheduleReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	logger logr.Logger
}

// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=scanallschedules,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=scanallschedules/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=scanallschedules/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections;clusterharborconnections,verbs=get;list;watch

func (r *ScanAllScheduleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.FromContext(ctx).WithName(fmt.Sprintf("[ScanAllSchedule:%s]", req.NamespacedName))

	var cr harborv1alpha1.ScanAllSchedule
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
	if err := ensureScanAllScheduleSingletonOwner(ctx, r.Client, &cr); err != nil {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}

	params, paramsHash, err := scheduleParameters(cr.Spec.Parameters, "scan all")
	if err != nil {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
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
	// Harbor treats Manual as a trigger. We still send parameters/hash.
	hash := hashParts(
		fmt.Sprintf("type=%s", cr.Spec.Schedule.Type),
		fmt.Sprintf("cron=%s", cr.Spec.Schedule.Cron),
		fmt.Sprintf("params=%s", paramsHash),
	)

	current, err := hc.GetScanAllSchedule(ctx)
	if err != nil && !harborclient.IsNotFound(err) {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}

	statusChanged := false
	reason, message := "Reconciled", "Scan all schedule reconciled"
	schedulesMatch := scanAllSchedulesEqual(current, &sched)
	switch {
	case harborclient.IsNotFound(err):
		if err := hc.CreateScanAllSchedule(ctx, sched); err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		}
		cr.Status.LastAppliedScheduleHash = hash
		statusChanged = true
		reason, message = "Created", "Scan all schedule created"
	case !schedulesMatch:
		if err := hc.UpdateScanAllSchedule(ctx, sched); err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		}
		cr.Status.LastAppliedScheduleHash = hash
		statusChanged = true
		r.logger.Info("Updated scan all schedule")
	case cr.Status.LastAppliedScheduleHash != hash:
		cr.Status.LastAppliedScheduleHash = hash
		statusChanged = true
	}

	condChanged := markReady(&cr.Status.HarborStatusBase, cr.Generation, reason, message)
	if statusChanged || condChanged {
		if err := r.Status().Update(ctx, &cr); err != nil {
			return ctrl.Result{}, err
		}
	}
	return returnWithDriftDetection(&cr.Spec.HarborSpecBase)
}

func (r *ScanAllScheduleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	builder, err := setupHarborBackedController(
		mgr,
		&harborv1alpha1.ScanAllSchedule{},
		func() client.ObjectList { return &harborv1alpha1.ScanAllScheduleList{} },
		func(obj client.Object) *harborv1alpha1.HarborConnectionReference {
			return obj.(*harborv1alpha1.ScanAllSchedule).Spec.HarborConnectionRef
		},
		"scanallschedule",
	)
	if err != nil {
		return err
	}
	return builder.Complete(r)
}

func scanAllSchedulesEqual(current, desired *harborclient.Schedule) bool {
	if current == nil || desired == nil {
		return false
	}
	if current.Schedule.Type != desired.Schedule.Type || current.Schedule.Cron != desired.Schedule.Cron {
		return false
	}
	currentJSON, err := json.Marshal(current.Parameters)
	if err != nil {
		return false
	}
	desiredJSON, err := json.Marshal(desired.Parameters)
	if err != nil {
		return false
	}
	return string(currentJSON) == string(desiredJSON)
}
