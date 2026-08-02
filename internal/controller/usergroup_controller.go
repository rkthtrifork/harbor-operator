package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
	"github.com/rkthtrifork/harbor-operator/internal/harborclient"
)

// UserGroupClaimReconciler ensures that an external group is registered in
// Harbor. Claims are non-owning and therefore never delete a Harbor group.
type UserGroupClaimReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	Options OperatorOptions
	logger  logr.Logger
}

// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=usergroupclaims,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=usergroupclaims/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=usergroupclaims/finalizers,verbs=update
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=members,verbs=get;list;watch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections;clusterharborconnections,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

func (r *UserGroupClaimReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.FromContext(ctx).WithName(fmt.Sprintf("[UserGroupClaim:%s]", req.NamespacedName))

	var claim harborv1alpha1.UserGroupClaim
	if found, err := loadResource(ctx, r.Client, req.NamespacedName, &claim, r.logger); err != nil {
		return ctrl.Result{}, err
	} else if !found {
		return ctrl.Result{}, nil
	}

	if err := markReconcilingIfNeeded(ctx, r.Client, &claim, &claim.Status.HarborStatusBase, claim.Generation); err != nil {
		return ctrl.Result{}, err
	}

	if !claim.DeletionTimestamp.IsZero() {
		if referenced, err := r.hasActiveMembers(ctx, &claim); err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &claim, &claim.Status.HarborStatusBase, claim.Generation, err)
		} else if referenced {
			err := fmt.Errorf("UserGroupClaim %s/%s is still referenced by an active Member", claim.Namespace, claim.Name)
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &claim, &claim.Status.HarborStatusBase, claim.Generation, err)
		}
		return ctrl.Result{}, removeFinalizer(ctx, r.Client, &claim)
	}

	if err := ensureFinalizer(ctx, r.Client, &claim); err != nil {
		return ctrl.Result{}, err
	}

	hc, err := getHarborClientForObject(ctx, r.Options, r.Client, &claim, &claim.Status.HarborStatusBase, claim.Namespace, claim.Spec.HarborConnectionRef)
	if err != nil {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &claim, &claim.Status.HarborStatusBase, claim.Generation, err)
	}

	desired := harborclient.UserGroup{
		GroupName:   claim.Spec.GroupName,
		GroupType:   claim.Spec.GroupType,
		LDAPGroupDN: claim.Spec.LDAPGroupDN,
	}
	current, found, err := findUserGroup(ctx, hc, desired)
	if err != nil {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &claim, &claim.Status.HarborStatusBase, claim.Generation, err)
	}
	if !found {
		id, createErr := hc.CreateUserGroup(ctx, desired)
		if createErr != nil && harborclient.IsConflict(createErr) {
			current, found, err = findUserGroup(ctx, hc, desired)
			if err != nil {
				return ctrl.Result{}, setErrorStatus(ctx, r.Client, &claim, &claim.Status.HarborStatusBase, claim.Generation, err)
			}
			if !found {
				createErr = fmt.Errorf("harbor reported a conflicting UserGroup for %q, but no compatible group could be found", desired.GroupName)
			} else {
				createErr = nil
			}
		}
		if createErr != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &claim, &claim.Status.HarborStatusBase, claim.Generation, createErr)
		}
		if !found {
			current = &harborclient.UserGroup{ID: id}
		}
	}

	if current == nil || current.ID == 0 {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &claim, &claim.Status.HarborStatusBase, claim.Generation, fmt.Errorf("harbor returned no UserGroup ID for %q", desired.GroupName))
	}
	claim.Status.HarborGroupID = current.ID
	if err := setReadyStatus(ctx, r.Client, &claim, &claim.Status.HarborStatusBase, claim.Generation, "Reconciled", "External group claim reconciled"); err != nil {
		return ctrl.Result{}, err
	}
	return returnWithDriftDetection(r.Options, &claim.Spec.HarborClaimSpecBase)
}

func findUserGroup(ctx context.Context, hc *harborclient.Client, desired harborclient.UserGroup) (*harborclient.UserGroup, bool, error) {
	groups, err := hc.ListUserGroups(ctx)
	if err != nil {
		return nil, false, err
	}
	for i := range groups {
		group := &groups[i]
		if !strings.EqualFold(group.GroupName, desired.GroupName) {
			continue
		}
		if group.GroupType != desired.GroupType || !strings.EqualFold(group.LDAPGroupDN, desired.LDAPGroupDN) {
			return nil, false, fmt.Errorf("harbor UserGroup %q exists with incompatible type or LDAP group DN", desired.GroupName)
		}
		return group, true, nil
	}
	return nil, false, nil
}

func (r *UserGroupClaimReconciler) hasActiveMembers(ctx context.Context, claim *harborv1alpha1.UserGroupClaim) (bool, error) {
	var members harborv1alpha1.MemberList
	if err := r.List(ctx, &members); err != nil {
		return false, err
	}
	for i := range members.Items {
		member := &members.Items[i]
		if member.Spec.MemberGroup == nil {
			continue
		}
		ref := member.Spec.MemberGroup.GroupClaimRef
		namespace := ref.Namespace
		if namespace == "" {
			namespace = member.Namespace
		}
		if namespace == claim.Namespace && ref.Name == claim.Name {
			return true, nil
		}
	}
	return false, nil
}

func (r *UserGroupClaimReconciler) SetupWithManager(mgr ctrl.Manager) error {
	builder, err := setupHarborBackedController(
		mgr,
		r.Options,
		&harborv1alpha1.UserGroupClaim{},
		func() client.ObjectList { return &harborv1alpha1.UserGroupClaimList{} },
		func(obj client.Object) *harborv1alpha1.HarborConnectionReference {
			return obj.(*harborv1alpha1.UserGroupClaim).Spec.HarborConnectionRef
		},
		"usergroupclaim",
	)
	if err != nil {
		return err
	}
	builder.Watches(&harborv1alpha1.Member{}, handler.EnqueueRequestsFromMapFunc(func(_ context.Context, object client.Object) []reconcile.Request {
		member := object.(*harborv1alpha1.Member)
		if member.Spec.MemberGroup == nil {
			return nil
		}
		ref := member.Spec.MemberGroup.GroupClaimRef
		namespace := ref.Namespace
		if namespace == "" {
			namespace = member.Namespace
		}
		return []reconcile.Request{{NamespacedName: client.ObjectKey{Namespace: namespace, Name: ref.Name}}}
	}))
	return builder.Complete(r)
}
