package controller

import (
	"context"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
	"github.com/rkthtrifork/harbor-operator/internal/harborclient"
)

type WebhookPolicyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	logger logr.Logger
}

// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=webhookpolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=webhookpolicies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=webhookpolicies/finalizers,verbs=update
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=projects,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections;clusterharborconnections,verbs=get;list;watch

func (r *WebhookPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.FromContext(ctx).WithName(fmt.Sprintf("[WebhookPolicy:%s]", req.NamespacedName))

	var cr harborv1alpha1.WebhookPolicy
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

	if done, err := finalizeIfDeleting(ctx, r.Client, &cr, cr.Spec.GetDeletionPolicy(), func() error {
		if cr.Status.HarborWebhookPolicyID == 0 {
			return nil
		}
		projectKey, _, resolveErr := resolveProject(ctx, r.Client, cr.Namespace, cr.Spec.ProjectRef)
		if resolveErr != nil {
			return resolveErr
		}
		return hc.DeleteWebhookPolicy(ctx, projectKey, cr.Status.HarborWebhookPolicyID)
	}); done {
		return ctrl.Result{}, err
	}

	if err := ensureFinalizer(ctx, r.Client, &cr); err != nil {
		return ctrl.Result{}, err
	}

	return r.reconcileWebhookPolicy(ctx, hc, &cr)
}

func (r *WebhookPolicyReconciler) reconcileWebhookPolicy(ctx context.Context, hc *harborclient.Client, cr *harborv1alpha1.WebhookPolicy) (ctrl.Result, error) {
	projectKey, projectID, err := resolveProject(ctx, r.Client, cr.Namespace, cr.Spec.ProjectRef)
	if err != nil {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}

	targets, targetsHash, err := r.buildTargets(ctx, cr)
	if err != nil {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}

	policy := harborclient.WebhookPolicy{
		Name:        cr.Name,
		Description: cr.Spec.Description,
		Targets:     targets,
		EventTypes:  cr.Spec.EventTypes,
		Enabled:     cr.Spec.Enabled != nil && *cr.Spec.Enabled,
	}
	if projectID != 0 {
		policy.ProjectID = projectID
	}

	if cr.Status.HarborWebhookPolicyID == 0 && cr.Spec.AllowTakeover {
		adopted, err := r.adoptExisting(ctx, hc, projectKey, cr)
		if err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		}
		if adopted {
			r.logger.Info("Adopted existing webhook policy", "ID", cr.Status.HarborWebhookPolicyID)
			return ctrl.Result{Requeue: true}, nil
		}
	}

	if cr.Status.HarborWebhookPolicyID == 0 {
		id, err := hc.CreateWebhookPolicy(ctx, projectKey, policy)
		if err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		}
		cr.Status.HarborWebhookPolicyID = id
		cr.Status.TargetsHash = targetsHash
		if err := setReadyStatus(ctx, r.Client, cr, &cr.Status.HarborStatusBase, cr.Generation, "Created", "Webhook policy created"); err != nil {
			return ctrl.Result{}, err
		}
		return returnWithDriftDetection(&cr.Spec.HarborSpecBase)
	}

	current, err := hc.GetWebhookPolicy(ctx, projectKey, cr.Status.HarborWebhookPolicyID)
	if err != nil {
		if harborclient.IsNotFound(err) {
			return requeueOnRemoteNotFound(ctx, r.Client, cr, &cr.Status.HarborStatusBase, cr.Generation, func() {
				cr.Status.HarborWebhookPolicyID = 0
			}, "Webhook policy not found in Harbor")
		}
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}

	statusChanged := false
	if webhookPolicyNeedsUpdate(policy, current) || (targetsHash != "" && targetsHash != cr.Status.TargetsHash) {
		if err := hc.UpdateWebhookPolicy(ctx, projectKey, cr.Status.HarborWebhookPolicyID, policy); err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		}
		r.logger.Info("Updated webhook policy", "ID", cr.Status.HarborWebhookPolicyID)
		if targetsHash != "" && targetsHash != cr.Status.TargetsHash {
			cr.Status.TargetsHash = targetsHash
			statusChanged = true
		}
	}

	condChanged := markReady(&cr.Status.HarborStatusBase, cr.Generation, "Reconciled", "Webhook policy reconciled")
	if statusChanged || condChanged {
		if err := r.Status().Update(ctx, cr); err != nil {
			return ctrl.Result{}, err
		}
	}
	return returnWithDriftDetection(&cr.Spec.HarborSpecBase)
}

func (r *WebhookPolicyReconciler) buildTargets(ctx context.Context, cr *harborv1alpha1.WebhookPolicy) ([]harborclient.WebhookTarget, string, error) {
	if len(cr.Spec.Targets) == 0 {
		return nil, "", fmt.Errorf("spec.targets must not be empty")
	}
	out := make([]harborclient.WebhookTarget, 0, len(cr.Spec.Targets))
	hashPartsList := make([]string, 0, len(cr.Spec.Targets))
	for i, t := range cr.Spec.Targets {
		authHeader := t.AuthHeader
		if t.AuthHeaderSecretRef != nil {
			if t.AuthHeader != "" {
				return nil, "", fmt.Errorf("spec.targets[%d]: authHeader and authHeaderSecretRef are mutually exclusive", i)
			}
			value, err := readSecretValue(ctx, r.Client, *t.AuthHeaderSecretRef, cr.Namespace, "authHeader")
			if err != nil {
				return nil, "", fmt.Errorf("spec.targets[%d]: failed to read authHeaderSecretRef: %w", i, err)
			}
			authHeader = value
		}
		out = append(out, harborclient.WebhookTarget{
			Type:           t.Type,
			Address:        t.Address,
			AuthHeader:     authHeader,
			PayloadFormat:  t.PayloadFormat,
			SkipCertVerify: t.SkipCertVerify,
		})
		hashPartsList = append(hashPartsList, fmt.Sprintf("type=%s|addr=%s|auth=%s|format=%s|skip=%t", t.Type, t.Address, authHeader, t.PayloadFormat, t.SkipCertVerify))
	}
	return out, hashParts(hashPartsList...), nil
}

func (r *WebhookPolicyReconciler) adoptExisting(ctx context.Context, hc *harborclient.Client, projectKey string, cr *harborv1alpha1.WebhookPolicy) (bool, error) {
	policies, err := hc.ListWebhookPolicies(ctx, projectKey)
	if err != nil {
		return false, err
	}
	for _, p := range policies {
		if strings.EqualFold(p.Name, cr.Name) {
			cr.Status.HarborWebhookPolicyID = p.ID
			return true, r.Status().Update(ctx, cr)
		}
	}
	return false, nil
}

func (r *WebhookPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	builder, err := setupHarborBackedController(
		mgr,
		&harborv1alpha1.WebhookPolicy{},
		func() client.ObjectList { return &harborv1alpha1.WebhookPolicyList{} },
		func(obj client.Object) *harborv1alpha1.HarborConnectionReference {
			return obj.(*harborv1alpha1.WebhookPolicy).Spec.HarborConnectionRef
		},
		"webhookpolicy",
	)
	if err != nil {
		return err
	}
	return builder.Complete(r)
}

func webhookPolicyNeedsUpdate(desired harborclient.WebhookPolicy, current *harborclient.WebhookPolicy) bool {
	if current == nil {
		return true
	}
	nd := normalizeWebhookPolicy(desired)
	nc := normalizeWebhookPolicy(*current)
	return !reflect.DeepEqual(nd, nc)
}

func normalizeWebhookPolicy(in harborclient.WebhookPolicy) harborclient.WebhookPolicy {
	in.ID = 0
	in.ProjectID = 0
	in.Creator = ""
	in.CreationTime = ""
	in.UpdateTime = ""
	for i := range in.Targets {
		in.Targets[i].AuthHeader = ""
	}
	sort.Strings(in.EventTypes)
	sort.SliceStable(in.Targets, func(i, j int) bool {
		return webhookTargetKey(in.Targets[i]) < webhookTargetKey(in.Targets[j])
	})
	return in
}

func webhookTargetKey(t harborclient.WebhookTarget) string {
	return fmt.Sprintf("%s|%s|%s|%t", t.Type, t.Address, t.PayloadFormat, t.SkipCertVerify)
}
