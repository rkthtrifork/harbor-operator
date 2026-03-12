package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"

	"github.com/go-logr/logr"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
	"github.com/rkthtrifork/harbor-operator/internal/harborclient"
)

type ImmutableTagRuleReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	logger logr.Logger
}

// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=immutabletagrules,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=immutabletagrules/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=immutabletagrules/finalizers,verbs=update
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=projects,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections;clusterharborconnections,verbs=get;list;watch

func (r *ImmutableTagRuleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.FromContext(ctx).WithName(fmt.Sprintf("[ImmutableTagRule:%s]", req.NamespacedName))

	var cr harborv1alpha1.ImmutableTagRule
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
		if done, finalErr := finalizeWithoutHarborConnection(ctx, r.Client, &cr, cr.Spec.GetDeletionPolicy(), true, err); done {
			return ctrl.Result{}, finalErr
		}
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}

	if done, err := finalizeIfDeleting(ctx, r.Client, &cr, func() error {
		if cr.Status.HarborImmutableRuleID == 0 {
			return nil
		}
		projectKey, _, resolveErr := resolveProject(ctx, r.Client, hc, cr.Namespace, cr.Spec.ProjectRef, cr.Spec.ProjectNameOrID)
		if resolveErr != nil {
			return resolveErr
		}
		return hc.DeleteImmutableRule(ctx, projectKey, cr.Status.HarborImmutableRuleID)
	}); done {
		return ctrl.Result{}, err
	}

	if err := ensureFinalizer(ctx, r.Client, &cr); err != nil {
		return ctrl.Result{}, err
	}

	projectKey, _, err := resolveProject(ctx, r.Client, hc, cr.Namespace, cr.Spec.ProjectRef, cr.Spec.ProjectNameOrID)
	if err != nil {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}

	params, err := jsonMapToAnyImmutable(cr.Spec.Params)
	if err != nil {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}
	desired := harborclient.ImmutableRule{
		Priority:       cr.Spec.Priority,
		Disabled:       cr.Spec.Disabled,
		Action:         cr.Spec.Action,
		Template:       cr.Spec.Template,
		Params:         params,
		TagSelectors:   toImmutableSelectors(cr.Spec.TagSelectors),
		ScopeSelectors: toImmutableScopeSelectors(cr.Spec.ScopeSelectors),
	}

	if cr.Status.HarborImmutableRuleID == 0 && cr.Spec.AllowTakeover {
		adopted, err := r.adoptExisting(ctx, hc, projectKey, &cr, desired)
		if err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		}
		if adopted {
			r.logger.Info("Adopted existing immutable tag rule", "ID", cr.Status.HarborImmutableRuleID)
			return ctrl.Result{Requeue: true}, nil
		}
	}

	if cr.Status.HarborImmutableRuleID == 0 {
		if err := hc.CreateImmutableRule(ctx, projectKey, desired); err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		}
		id, err := r.findMatchingRuleID(ctx, hc, projectKey, desired)
		if err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		}
		cr.Status.HarborImmutableRuleID = id
		if err := setReadyStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, "Created", "Immutable tag rule created"); err != nil {
			return ctrl.Result{}, err
		}
		return returnWithDriftDetection(&cr.Spec.HarborSpecBase)
	}

	current, err := r.getImmutableRule(ctx, hc, projectKey, cr.Status.HarborImmutableRuleID)
	if err != nil {
		if harborclient.IsNotFound(err) {
			return requeueOnRemoteNotFound(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, func() {
				cr.Status.HarborImmutableRuleID = 0
			}, "Immutable tag rule not found in Harbor")
		}
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}

	if immutableRuleNeedsUpdate(desired, current) {
		if err := hc.UpdateImmutableRule(ctx, projectKey, cr.Status.HarborImmutableRuleID, desired); err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		}
		r.logger.Info("Updated immutable tag rule", "ID", cr.Status.HarborImmutableRuleID)
	}

	if err := setReadyStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, "Reconciled", "Immutable tag rule reconciled"); err != nil {
		return ctrl.Result{}, err
	}
	return returnWithDriftDetection(&cr.Spec.HarborSpecBase)
}

func (r *ImmutableTagRuleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	builder, err := setupHarborBackedController(
		mgr,
		&harborv1alpha1.ImmutableTagRule{},
		func() client.ObjectList { return &harborv1alpha1.ImmutableTagRuleList{} },
		func(obj client.Object) harborv1alpha1.HarborConnectionReference {
			return obj.(*harborv1alpha1.ImmutableTagRule).Spec.HarborConnectionRef
		},
		"immutabletagrule",
	)
	if err != nil {
		return err
	}
	return builder.Complete(r)
}

func (r *ImmutableTagRuleReconciler) adoptExisting(ctx context.Context, hc *harborclient.Client, projectKey string, cr *harborv1alpha1.ImmutableTagRule, desired harborclient.ImmutableRule) (bool, error) {
	ruleID, err := r.findMatchingRuleID(ctx, hc, projectKey, desired)
	if err != nil {
		return false, err
	}
	if ruleID == 0 {
		return false, nil
	}
	cr.Status.HarborImmutableRuleID = ruleID
	return true, r.Status().Update(ctx, cr)
}

func (r *ImmutableTagRuleReconciler) findMatchingRuleID(ctx context.Context, hc *harborclient.Client, projectKey string, desired harborclient.ImmutableRule) (int, error) {
	rules, err := hc.ListImmutableRules(ctx, projectKey)
	if err != nil {
		return 0, err
	}
	for _, rule := range rules {
		if immutableRuleMatches(desired, rule) {
			return rule.ID, nil
		}
	}
	return 0, nil
}

func (r *ImmutableTagRuleReconciler) getImmutableRule(ctx context.Context, hc *harborclient.Client, projectKey string, id int) (*harborclient.ImmutableRule, error) {
	rules, err := hc.ListImmutableRules(ctx, projectKey)
	if err != nil {
		return nil, err
	}
	for i := range rules {
		if rules[i].ID == id {
			return &rules[i], nil
		}
	}
	return nil, &harborclient.HTTPError{StatusCode: 404, Message: "rule not found"}
}

func immutableRuleMatches(desired harborclient.ImmutableRule, current harborclient.ImmutableRule) bool {
	d := normalizeImmutableRule(desired)
	c := normalizeImmutableRule(current)
	return reflect.DeepEqual(d, c)
}

func immutableRuleNeedsUpdate(desired harborclient.ImmutableRule, current *harborclient.ImmutableRule) bool {
	if current == nil {
		return true
	}
	return !immutableRuleMatches(desired, *current)
}

func normalizeImmutableRule(in harborclient.ImmutableRule) harborclient.ImmutableRule {
	in.ID = 0
	in.Params = normalizeAnyMap(in.Params)
	sort.SliceStable(in.TagSelectors, func(i, j int) bool {
		return immutableSelectorKey(in.TagSelectors[i]) < immutableSelectorKey(in.TagSelectors[j])
	})
	if len(in.TagSelectors) == 0 {
		in.TagSelectors = nil
	}
	if in.ScopeSelectors != nil {
		keys := make([]string, 0, len(in.ScopeSelectors))
		for key := range in.ScopeSelectors {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		sorted := make(map[string][]harborclient.ImmutableSelector, len(in.ScopeSelectors))
		for _, key := range keys {
			selectors := in.ScopeSelectors[key]
			sort.SliceStable(selectors, func(i, j int) bool {
				return immutableSelectorKey(selectors[i]) < immutableSelectorKey(selectors[j])
			})
			sorted[key] = selectors
		}
		in.ScopeSelectors = sorted
	}
	if len(in.ScopeSelectors) == 0 {
		in.ScopeSelectors = nil
	}
	return in
}

func immutableSelectorKey(in harborclient.ImmutableSelector) string {
	return fmt.Sprintf("%s|%s|%s|%s", in.Kind, in.Decoration, in.Pattern, in.Extras)
}

func toImmutableSelectors(in []harborv1alpha1.ImmutableSelector) []harborclient.ImmutableSelector {
	out := make([]harborclient.ImmutableSelector, 0, len(in))
	for _, sel := range in {
		out = append(out, harborclient.ImmutableSelector{
			Kind:       sel.Kind,
			Decoration: sel.Decoration,
			Pattern:    sel.Pattern,
			Extras:     sel.Extras,
		})
	}
	return out
}

func toImmutableScopeSelectors(in map[string][]harborv1alpha1.ImmutableSelector) map[string][]harborclient.ImmutableSelector {
	if in == nil {
		return nil
	}
	out := map[string][]harborclient.ImmutableSelector{}
	for key, selectors := range in {
		out[key] = toImmutableSelectors(selectors)
	}
	return out
}

func jsonMapToAnyImmutable(in map[string]apiextensionsv1.JSON) (map[string]any, error) {
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
			return nil, fmt.Errorf("invalid params for %s: %w", key, err)
		}
		out[key] = value
	}
	return out, nil
}

func normalizeAnyMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := map[string]any{}
	for key, value := range in {
		out[key] = normalizeAny(value)
	}
	return out
}
