package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/go-logr/logr"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
	"github.com/rkthtrifork/harbor-operator/internal/harborclient"
)

type GCScheduleReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	logger logr.Logger
}

// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=gcschedules,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=gcschedules/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=gcschedules/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections;clusterharborconnections,verbs=get;list;watch

func (r *GCScheduleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.FromContext(ctx).WithName(fmt.Sprintf("[GCSchedule:%s]", req.NamespacedName))

	var cr harborv1alpha1.GCSchedule
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
	if err := ensureGCScheduleSingletonOwner(ctx, r.Client, &cr); err != nil {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}

	params, paramsHash, err := gcParameters(cr.Spec.Parameters)
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
	hash := hashParts(
		fmt.Sprintf("type=%s", cr.Spec.Schedule.Type),
		fmt.Sprintf("cron=%s", cr.Spec.Schedule.Cron),
		fmt.Sprintf("params=%s", paramsHash),
	)

	if cr.Status.LastAppliedScheduleHash == "" {
		if err := hc.CreateGCSchedule(ctx, sched); err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		}
		cr.Status.LastAppliedScheduleHash = hash
		if err := setReadyStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, "Created", "GC schedule created"); err != nil {
			return ctrl.Result{}, err
		}
		return returnWithDriftDetection(&cr.Spec.HarborSpecBase)
	}

	statusChanged := false
	if cr.Status.LastAppliedScheduleHash != hash {
		if err := hc.UpdateGCSchedule(ctx, sched); err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		}
		cr.Status.LastAppliedScheduleHash = hash
		statusChanged = true
		r.logger.Info("Updated GC schedule")
	}

	condChanged := markReady(&cr.Status.HarborStatusBase, cr.Generation, "Reconciled", "GC schedule reconciled")
	if statusChanged || condChanged {
		if err := r.Status().Update(ctx, &cr); err != nil {
			return ctrl.Result{}, err
		}
	}
	return returnWithDriftDetection(&cr.Spec.HarborSpecBase)
}

func (r *GCScheduleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	builder, err := setupHarborBackedController(
		mgr,
		&harborv1alpha1.GCSchedule{},
		func() client.ObjectList { return &harborv1alpha1.GCScheduleList{} },
		func(obj client.Object) harborv1alpha1.HarborConnectionReference {
			return obj.(*harborv1alpha1.GCSchedule).Spec.HarborConnectionRef
		},
		"gcschedule",
	)
	if err != nil {
		return err
	}
	return builder.Complete(r)
}

func gcParameters(in map[string]apiextensionsv1.JSON) (map[string]any, string, error) {
	if in == nil {
		return nil, "", nil
	}
	out := map[string]any{}
	keys := make([]string, 0, len(in))
	for key, raw := range in {
		if len(raw.Raw) == 0 {
			continue
		}
		var value any
		if err := json.Unmarshal(raw.Raw, &value); err != nil {
			return nil, "", fmt.Errorf("invalid gc parameters for %s: %w", key, err)
		}
		out[key] = value
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		raw := in[key]
		if len(raw.Raw) == 0 {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%s", key, strings.TrimSpace(string(raw.Raw))))
	}
	return out, strings.Join(parts, "&"), nil
}
