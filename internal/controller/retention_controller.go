package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

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

type RetentionPolicyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	logger logr.Logger
}

// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=retentionpolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=retentionpolicies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=retentionpolicies/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections,verbs=get;list;watch

func (r *RetentionPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.FromContext(ctx).WithName(fmt.Sprintf("[RetentionPolicy:%s]", req.NamespacedName))

	var cr harborv1alpha1.RetentionPolicy
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
			if cr.Status.HarborRetentionID != 0 {
				if err := hc.DeleteRetention(ctx, cr.Status.HarborRetentionID); err != nil {
					return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
				}
			}
			controllerutil.RemoveFinalizer(&cr, finalizerName)
			_ = r.Update(ctx, &cr)
		}
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(&cr, finalizerName) {
		controllerutil.AddFinalizer(&cr, finalizerName)
		_ = r.Update(ctx, &cr)
	}

	if cr.Spec.Trigger == nil {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, fmt.Errorf("spec.trigger is required"))
	}
	if cr.Spec.Trigger.Kind == "Schedule" {
		if cron, ok := cr.Spec.Trigger.Settings["cron"]; !ok || len(cron.Raw) == 0 {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, fmt.Errorf("spec.trigger.settings.cron is required for Schedule trigger"))
		}
	}

	policy, err := toRetentionPolicy(cr)
	if err != nil {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}

	if cr.Status.HarborRetentionID == 0 {
		newID, err := hc.CreateRetention(ctx, policy)
		if err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		}
		cr.Status.HarborRetentionID = newID
		markReady(&cr.Status.HarborStatusBase, cr.Generation, "Created", "Retention policy created")
		if err := r.Status().Update(ctx, &cr); err != nil {
			return ctrl.Result{}, err
		}
		return returnWithDriftDetection(&cr.Spec.HarborSpecBase)
	}

	current, err := hc.GetRetentionByID(ctx, cr.Status.HarborRetentionID)
	if err != nil {
		if harborclient.IsNotFound(err) {
			cr.Status.HarborRetentionID = 0
			markReconciling(&cr.Status.HarborStatusBase, cr.Generation, "NotFound", "Retention policy not found in Harbor")
			_ = r.Status().Update(ctx, &cr)
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}

	if retentionNeedsUpdate(policy, current) {
		if err := hc.UpdateRetention(ctx, cr.Status.HarborRetentionID, policy); err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		}
		r.logger.Info("Updated retention policy", "ID", cr.Status.HarborRetentionID)
	}

	if changed := markReady(&cr.Status.HarborStatusBase, cr.Generation, "Reconciled", "Retention policy reconciled"); changed {
		if err := r.Status().Update(ctx, &cr); err != nil {
			return ctrl.Result{}, err
		}
	}
	return returnWithDriftDetection(&cr.Spec.HarborSpecBase)
}

func toRetentionPolicy(cr harborv1alpha1.RetentionPolicy) (harborclient.RetentionPolicy, error) {
	rules := make([]harborclient.RetentionRule, 0, len(cr.Spec.Rules))
	for i, rule := range cr.Spec.Rules {
		params, err := jsonMapToObjectMap(rule.Params)
		if err != nil {
			return harborclient.RetentionPolicy{}, fmt.Errorf("invalid params for rule %d: %w", i, err)
		}
		rules = append(rules, harborclient.RetentionRule{
			Priority:       i + 1,
			Disabled:       rule.Disabled,
			Action:         rule.Action,
			Template:       rule.Template,
			Params:         params,
			TagSelectors:   toRetentionSelectors(rule.TagSelectors),
			ScopeSelectors: toRetentionScopeSelectors(rule.ScopeSelectors),
		})
	}
	trigger, err := toRetentionTrigger(cr.Spec.Trigger)
	if err != nil {
		return harborclient.RetentionPolicy{}, err
	}
	return harborclient.RetentionPolicy{
		Algorithm: cr.Spec.Algorithm,
		Rules:     rules,
		Trigger:   trigger,
		Scope:     toRetentionScope(cr.Spec.Scope),
	}, nil
}

func toRetentionTrigger(trigger *harborv1alpha1.RetentionTrigger) (*harborclient.RetentionTrigger, error) {
	if trigger == nil {
		return nil, nil
	}
	settings, err := jsonMapToAny(trigger.Settings)
	if err != nil {
		return nil, fmt.Errorf("invalid trigger settings: %w", err)
	}
	references, err := jsonMapToAny(trigger.References)
	if err != nil {
		return nil, fmt.Errorf("invalid trigger references: %w", err)
	}
	return &harborclient.RetentionTrigger{
		Kind:       trigger.Kind,
		Settings:   settings,
		References: references,
	}, nil
}

func toRetentionScope(scope *harborv1alpha1.RetentionScope) *harborclient.RetentionScope {
	if scope == nil {
		return nil
	}
	return &harborclient.RetentionScope{
		Level: scope.Level,
		Ref:   scope.Ref,
	}
}

func toRetentionSelectors(in []harborv1alpha1.RetentionSelector) []harborclient.RetentionSelector {
	out := make([]harborclient.RetentionSelector, 0, len(in))
	for _, sel := range in {
		out = append(out, harborclient.RetentionSelector{
			Kind:       sel.Kind,
			Decoration: sel.Decoration,
			Pattern:    sel.Pattern,
			Extras:     sel.Extras,
		})
	}
	return out
}

func toRetentionScopeSelectors(in map[string][]harborv1alpha1.RetentionSelector) map[string][]harborclient.RetentionSelector {
	if in == nil {
		return nil
	}
	out := map[string][]harborclient.RetentionSelector{}
	for key, selectors := range in {
		out[key] = toRetentionSelectors(selectors)
	}
	return out
}

func jsonMapToAny(in map[string]apiextensionsv1.JSON) (map[string]any, error) {
	if in == nil {
		return nil, nil
	}
	out := map[string]any{}
	for key, raw := range in {
		if len(raw.Raw) == 0 {
			continue
		}
		var value any
		if err := json.Unmarshal(raw.Raw, &value); err != nil {
			return nil, fmt.Errorf("invalid json for %s: %w", key, err)
		}
		out[key] = value
	}
	return out, nil
}

func jsonMapToObjectMap(in map[string]apiextensionsv1.JSON) (map[string]map[string]any, error) {
	if in == nil {
		return nil, nil
	}
	out := map[string]map[string]any{}
	for key, raw := range in {
		if len(raw.Raw) == 0 {
			continue
		}
		var value map[string]any
		if err := json.Unmarshal(raw.Raw, &value); err != nil {
			return nil, fmt.Errorf("invalid json for %s: %w", key, err)
		}
		out[key] = value
	}
	return out, nil
}

func retentionNeedsUpdate(desired harborclient.RetentionPolicy, current *harborclient.RetentionPolicy) bool {
	if current == nil {
		return true
	}
	normalizedDesired := normalizeRetentionPolicy(desired)
	normalizedCurrent := normalizeRetentionPolicy(*current)
	return !reflect.DeepEqual(normalizedDesired, normalizedCurrent)
}

func normalizeRetentionPolicy(in harborclient.RetentionPolicy) harborclient.RetentionPolicy {
	in.ID = 0
	for i := range in.Rules {
		in.Rules[i] = normalizeRetentionRule(in.Rules[i])
	}
	return in
}

func normalizeRetentionRule(in harborclient.RetentionRule) harborclient.RetentionRule {
	in.ID = 0
	return in
}

func (r *RetentionPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&harborv1alpha1.RetentionPolicy{}).
		Named("retentionpolicy").
		Complete(r)
}
