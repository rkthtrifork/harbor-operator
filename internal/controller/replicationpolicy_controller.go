package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
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

type ReplicationPolicyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	logger logr.Logger
}

// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=replicationpolicies,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=replicationpolicies/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=replicationpolicies/finalizers,verbs=update
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=registries,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections,verbs=get;list;watch

func (r *ReplicationPolicyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.FromContext(ctx).WithName(fmt.Sprintf("[ReplicationPolicy:%s]", req.NamespacedName))

	var cr harborv1alpha1.ReplicationPolicy
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
		if cr.Status.HarborReplicationPolicyID == 0 {
			return nil
		}
		return hc.DeleteReplicationPolicy(ctx, cr.Status.HarborReplicationPolicyID)
	}); done {
		return ctrl.Result{}, err
	}

	if err := ensureFinalizer(ctx, r.Client, &cr); err != nil {
		return ctrl.Result{}, err
	}

	cr.Spec.Name = defaultString(cr.Spec.Name, cr.Name)

	srcID, err := resolveRegistryID(ctx, r.Client, cr.Namespace, cr.Spec.SourceRegistryRef, cr.Spec.SourceRegistryID)
	if err != nil {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}
	destID, err := resolveRegistryID(ctx, r.Client, cr.Namespace, cr.Spec.DestinationRegistryRef, cr.Spec.DestinationRegistryID)
	if err != nil {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}

	filters, err := replicationFiltersFromSpec(cr.Spec.Filters)
	if err != nil {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}
	policy := harborclient.ReplicationPolicy{
		Name:                      cr.Spec.Name,
		Description:               cr.Spec.Description,
		SrcRegistry:               &harborclient.Registry{ID: srcID},
		DestRegistry:              &harborclient.Registry{ID: destID},
		DestNamespace:             cr.Spec.DestNamespace,
		DestNamespaceReplaceCount: cr.Spec.DestNamespaceReplaceCount,
		Trigger:                   replicationTriggerFromSpec(cr.Spec.Trigger),
		Filters:                   filters,
		ReplicateDeletion:         cr.Spec.ReplicateDeletion,
		Override:                  cr.Spec.Override,
		Enabled:                   cr.Spec.Enabled,
		Speed:                     cr.Spec.Speed,
		CopyByChunk:               cr.Spec.CopyByChunk,
		SingleActiveReplication:   cr.Spec.SingleActiveReplication,
	}

	if cr.Status.HarborReplicationPolicyID == 0 && cr.Spec.AllowTakeover {
		adopted, err := r.adoptExisting(ctx, hc, &cr)
		if err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		}
		if adopted {
			r.logger.Info("Adopted existing replication policy", "ID", cr.Status.HarborReplicationPolicyID)
			return ctrl.Result{Requeue: true}, nil
		}
	}

	if cr.Status.HarborReplicationPolicyID == 0 {
		id, err := hc.CreateReplicationPolicy(ctx, policy)
		if err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		}
		cr.Status.HarborReplicationPolicyID = id
		if err := setReadyStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, "Created", "Replication policy created"); err != nil {
			return ctrl.Result{}, err
		}
		return returnWithDriftDetection(&cr.Spec.HarborSpecBase)
	}

	current, err := hc.GetReplicationPolicy(ctx, cr.Status.HarborReplicationPolicyID)
	if err != nil {
		if harborclient.IsNotFound(err) {
			cr.Status.HarborReplicationPolicyID = 0
			if err := setReconcilingStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, "NotFound", "Replication policy not found in Harbor"); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}

	if replicationPolicyNeedsUpdate(policy, current) {
		if err := hc.UpdateReplicationPolicy(ctx, cr.Status.HarborReplicationPolicyID, policy); err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		}
		r.logger.Info("Updated replication policy", "ID", cr.Status.HarborReplicationPolicyID)
	}

	if err := setReadyStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, "Reconciled", "Replication policy reconciled"); err != nil {
		return ctrl.Result{}, err
	}
	return returnWithDriftDetection(&cr.Spec.HarborSpecBase)
}

func (r *ReplicationPolicyReconciler) adoptExisting(ctx context.Context, hc *harborclient.Client, cr *harborv1alpha1.ReplicationPolicy) (bool, error) {
	policies, err := hc.ListReplicationPolicies(ctx, cr.Spec.Name)
	if err != nil {
		return false, err
	}
	for _, p := range policies {
		if strings.EqualFold(p.Name, cr.Spec.Name) {
			cr.Status.HarborReplicationPolicyID = p.ID
			return true, r.Status().Update(ctx, cr)
		}
	}
	return false, nil
}

func (r *ReplicationPolicyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&harborv1alpha1.ReplicationPolicy{}).
		Named("replicationpolicy").
		Complete(r)
}

func replicationTriggerFromSpec(in *harborv1alpha1.ReplicationTriggerSpec) *harborclient.ReplicationTrigger {
	if in == nil {
		return nil
	}
	trigger := &harborclient.ReplicationTrigger{Type: in.Type}
	if in.Settings != nil {
		trigger.TriggerSettings = &harborclient.ReplicationTriggerSettings{Cron: in.Settings.Cron}
	}
	return trigger
}

func replicationFiltersFromSpec(in []harborv1alpha1.ReplicationFilterSpec) ([]harborclient.ReplicationFilter, error) {
	if len(in) == 0 {
		return nil, nil
	}
	out := make([]harborclient.ReplicationFilter, 0, len(in))
	for i, f := range in {
		var value any
		if len(f.Value.Raw) > 0 {
			if err := json.Unmarshal(f.Value.Raw, &value); err != nil {
				return nil, fmt.Errorf("invalid replication filter value at index %d: %w", i, err)
			}
		}
		out = append(out, harborclient.ReplicationFilter{
			Type:       f.Type,
			Value:      value,
			Decoration: f.Decoration,
		})
	}
	return out, nil
}

func replicationPolicyNeedsUpdate(desired harborclient.ReplicationPolicy, current *harborclient.ReplicationPolicy) bool {
	if current == nil {
		return true
	}
	nd := normalizeReplicationPolicy(desired)
	nc := normalizeReplicationPolicy(*current)
	return !reflect.DeepEqual(nd, nc)
}

func normalizeReplicationPolicy(in harborclient.ReplicationPolicy) harborclient.ReplicationPolicy {
	in.ID = 0
	in.CreationTime = ""
	in.UpdateTime = ""
	if in.SrcRegistry != nil {
		in.SrcRegistry = &harborclient.Registry{ID: in.SrcRegistry.ID}
	}
	if in.DestRegistry != nil {
		in.DestRegistry = &harborclient.Registry{ID: in.DestRegistry.ID}
	}
	for i := range in.Filters {
		in.Filters[i].Value = normalizeAny(in.Filters[i].Value)
	}
	sort.SliceStable(in.Filters, func(i, j int) bool {
		return replicationFilterKey(in.Filters[i]) < replicationFilterKey(in.Filters[j])
	})
	return in
}

func replicationFilterKey(f harborclient.ReplicationFilter) string {
	b, _ := json.Marshal(normalizeAny(f.Value))
	return fmt.Sprintf("%s|%s|%s", f.Type, f.Decoration, string(b))
}

func normalizeAny(in any) any {
	if in == nil {
		return nil
	}
	b, err := json.Marshal(in)
	if err != nil {
		return in
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		return in
	}
	return out
}
