package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/go-logr/logr"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections,verbs=get;list;watch

func (r *GCScheduleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.FromContext(ctx).WithName(fmt.Sprintf("[GCSchedule:%s]", req.NamespacedName))

	var cr harborv1alpha1.GCSchedule
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

	conn, err := getHarborConnection(ctx, r.Client, cr.Namespace, cr.Spec.HarborConnectionRef)
	if err != nil {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}
	if conn.Spec.Credentials == nil {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, fmt.Errorf("HarborConnection %s/%s has no credentials configured", conn.Namespace, conn.Name))
	}
	user, pass, err := getHarborAuth(ctx, r.Client, conn)
	if err != nil {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}
	hc := harborclient.New(conn.Spec.BaseURL, user, pass)

	if !cr.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&cr, finalizerName) {
			controllerutil.RemoveFinalizer(&cr, finalizerName)
			_ = r.Update(ctx, &cr)
		}
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(&cr, finalizerName) {
		controllerutil.AddFinalizer(&cr, finalizerName)
		_ = r.Update(ctx, &cr)
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
	if sched.Schedule.Type != "Manual" && sched.Schedule.Type != "None" && sched.Schedule.Cron == "" {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, fmt.Errorf("schedule.cron is required for schedule type %q", sched.Schedule.Type))
	}
	hashInput := fmt.Sprintf("type=%s\ncron=%s\nparams=%s", cr.Spec.Schedule.Type, cr.Spec.Schedule.Cron, paramsHash)
	hash := hashSecret(hashInput)

	if cr.Status.LastAppliedScheduleHash == "" {
		if err := hc.CreateGCSchedule(ctx, sched); err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		}
		cr.Status.LastAppliedScheduleHash = hash
		markReady(&cr.Status.HarborStatusBase, cr.Generation, "Created", "GC schedule created")
		if err := r.Status().Update(ctx, &cr); err != nil {
			return ctrl.Result{}, err
		}
		return returnWithDriftDetection(&cr.Spec.HarborSpecBase)
	}

	if cr.Status.LastAppliedScheduleHash != hash {
		if err := hc.UpdateGCSchedule(ctx, sched); err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		}
		cr.Status.LastAppliedScheduleHash = hash
		r.logger.Info("Updated GC schedule")
	}

	if changed := markReady(&cr.Status.HarborStatusBase, cr.Generation, "Reconciled", "GC schedule reconciled"); changed {
		if err := r.Status().Update(ctx, &cr); err != nil {
			return ctrl.Result{}, err
		}
	}
	return returnWithDriftDetection(&cr.Spec.HarborSpecBase)
}

func (r *GCScheduleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&harborv1alpha1.GCSchedule{}).
		Named("gcschedule").
		Complete(r)
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
