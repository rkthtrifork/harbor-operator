package controller

import (
	"context"
	"fmt"
	"reflect"
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

type UserGroupReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	logger logr.Logger
}

// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=usergroups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=usergroups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=usergroups/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections,verbs=get;list;watch

func (r *UserGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.FromContext(ctx).WithName(fmt.Sprintf("[UserGroup:%s]", req.NamespacedName))

	var cr harborv1alpha1.UserGroup
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
		if cr.Status.HarborGroupID == 0 {
			return nil
		}
		return hc.DeleteUserGroup(ctx, cr.Status.HarborGroupID)
	}); done {
		return ctrl.Result{}, err
	}

	if err := ensureFinalizer(ctx, r.Client, &cr); err != nil {
		return ctrl.Result{}, err
	}

	cr.Spec.GroupName = defaultString(cr.Spec.GroupName, cr.Name)

	desired := harborclient.UserGroup{
		GroupName:   cr.Spec.GroupName,
		GroupType:   cr.Spec.GroupType,
		LDAPGroupDN: cr.Spec.LDAPGroupDN,
	}

	if cr.Status.HarborGroupID == 0 && cr.Spec.AllowTakeover {
		adopted, err := r.adoptExisting(ctx, hc, &cr)
		if err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		}
		if adopted {
			r.logger.Info("Adopted existing user group", "ID", cr.Status.HarborGroupID)
			return ctrl.Result{Requeue: true}, nil
		}
	}

	if cr.Status.HarborGroupID == 0 {
		id, err := hc.CreateUserGroup(ctx, desired)
		if err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		}
		cr.Status.HarborGroupID = id
		if err := setReadyStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, "Created", "User group created"); err != nil {
			return ctrl.Result{}, err
		}
		return returnWithDriftDetection(&cr.Spec.HarborSpecBase)
	}

	current, err := hc.GetUserGroup(ctx, cr.Status.HarborGroupID)
	if err != nil {
		if harborclient.IsNotFound(err) {
			cr.Status.HarborGroupID = 0
			if err := setReconcilingStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, "NotFound", "User group not found in Harbor"); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}

	if userGroupNeedsUpdate(desired, current) {
		if err := hc.UpdateUserGroup(ctx, cr.Status.HarborGroupID, desired); err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		}
		r.logger.Info("Updated user group", "ID", cr.Status.HarborGroupID)
	}

	if err := setReadyStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, "Reconciled", "User group reconciled"); err != nil {
		return ctrl.Result{}, err
	}
	return returnWithDriftDetection(&cr.Spec.HarborSpecBase)
}

func (r *UserGroupReconciler) adoptExisting(ctx context.Context, hc *harborclient.Client, cr *harborv1alpha1.UserGroup) (bool, error) {
	groups, err := hc.SearchUserGroups(ctx, cr.Spec.GroupName)
	if err != nil {
		return false, err
	}
	for _, g := range groups {
		if strings.EqualFold(g.GroupName, cr.Spec.GroupName) {
			cr.Status.HarborGroupID = g.ID
			return true, r.Status().Update(ctx, cr)
		}
	}
	return false, nil
}

func (r *UserGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&harborv1alpha1.UserGroup{}).
		Named("usergroup").
		Complete(r)
}

func userGroupNeedsUpdate(desired harborclient.UserGroup, current *harborclient.UserGroup) bool {
	if current == nil {
		return true
	}
	nd := normalizeUserGroup(desired)
	nc := normalizeUserGroup(*current)
	return !reflect.DeepEqual(nd, nc)
}

func normalizeUserGroup(in harborclient.UserGroup) harborclient.UserGroup {
	in.ID = 0
	return in
}
