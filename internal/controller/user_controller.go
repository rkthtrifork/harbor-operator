package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
	"github.com/rkthtrifork/harbor-operator/internal/harborclient"
)

type UserReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	logger logr.Logger
}

// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=users,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=users/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections,verbs=get;list;watch

func (r *UserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.FromContext(ctx).WithName(fmt.Sprintf("[User:%s]", req.NamespacedName))

	// Load CR
	var cr harborv1alpha1.User
	if err := r.Get(ctx, req.NamespacedName, &cr); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	// Harbor client
	conn, err := getHarborConnection(ctx, r.Client, cr.Namespace, cr.Spec.HarborConnectionRef)
	if err != nil {
		return ctrl.Result{}, err
	}
	user, pass, err := getHarborAuth(ctx, r.Client, conn)
	if err != nil {
		return ctrl.Result{}, err
	}
	hc := harborclient.New(conn.Spec.BaseURL, user, pass)

	// Deletion
	if !cr.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&cr, finalizerName) {
			if err := r.deleteUser(ctx, hc, &cr); err != nil {
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(&cr, finalizerName)
			_ = r.Update(ctx, &cr)
		}
		return ctrl.Result{}, nil
	}

	// Finalizer
	if !controllerutil.ContainsFinalizer(&cr, finalizerName) {
		controllerutil.AddFinalizer(&cr, finalizerName)
		_ = r.Update(ctx, &cr)
	}

	// Defaults & adoption
	if cr.Spec.Username == "" {
		cr.Spec.Username = cr.Name
	}

	if cr.Status.HarborUserID == 0 && cr.Spec.AllowTakeover {
		if ok, err := r.adoptExisting(ctx, hc, &cr); err != nil {
			return ctrl.Result{}, err
		} else if ok {
			r.logger.Info("Adopted user", "ID", cr.Status.HarborUserID)
		}
	}

	// Desired payload
	createReq := r.buildCreateReq(cr)

	// Create / Update
	if cr.Status.HarborUserID == 0 {
		id, err := hc.CreateUser(ctx, createReq)
		if err != nil {
			return ctrl.Result{}, err
		}
		cr.Status.HarborUserID = id
		_ = r.Status().Update(ctx, &cr)
		return returnWithDriftDetection(&cr.Spec.HarborSpecBase)
	}

	current, err := hc.GetUserByID(ctx, cr.Status.HarborUserID)
	if err != nil {
		if harborclient.IsNotFound(err) {
			cr.Status.HarborUserID = 0
			_ = r.Status().Update(ctx, &cr)
			return ctrl.Result{Requeue: true}, nil
		}
		return ctrl.Result{}, err
	}

	if userNeedsUpdate(createReq, *current) {
		if err := hc.UpdateUser(ctx, current.UserID, createReq); err != nil {
			return ctrl.Result{}, err
		}
	}
	return returnWithDriftDetection(&cr.Spec.HarborSpecBase)
}

func (r *UserReconciler) buildCreateReq(cr harborv1alpha1.User) harborclient.CreateUserRequest {
	createReq := harborclient.CreateUserRequest{
		Email:    cr.Spec.Email,
		Realname: cr.Spec.Realname,
		Comment:  cr.Spec.Comment,
		Password: cr.Spec.Password,
		Username: cr.Spec.Username,
	}
	return createReq
}

func (r *UserReconciler) deleteUser(ctx context.Context, hc *harborclient.Client, cr *harborv1alpha1.User) error {
	if cr.Status.HarborUserID == 0 {
		return nil
	}
	err := hc.DeleteUser(ctx, cr.Status.HarborUserID)
	if harborclient.IsNotFound(err) {
		return nil
	}
	return err
}

func (r *UserReconciler) adoptExisting(ctx context.Context, hc *harborclient.Client, cr *harborv1alpha1.User) (bool, error) {
	users, err := hc.ListUsers(ctx, "username="+cr.Spec.Username)
	if err != nil {
		return false, err
	}
	for _, u := range users {
		if strings.EqualFold(u.Username, cr.Spec.Username) {
			cr.Status.HarborUserID = u.UserID
			return true, r.Status().Update(ctx, cr)
		}
	}
	return false, nil
}

func userNeedsUpdate(desired harborclient.CreateUserRequest, current harborclient.User) bool {
	return desired.Email != current.Email ||
		desired.Realname != current.Realname ||
		desired.Comment != current.Comment
}

func (r *UserReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&harborv1alpha1.User{}).
		Named("user").
		Complete(r)
}
