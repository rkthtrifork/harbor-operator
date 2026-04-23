package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=users/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections;clusterharborconnections,verbs=get;list;watch

func (r *UserReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.FromContext(ctx).WithName(fmt.Sprintf("[User:%s]", req.NamespacedName))

	// Load CR
	var cr harborv1alpha1.User
	if found, err := loadResource(ctx, r.Client, req.NamespacedName, &cr, r.logger); err != nil {
		return ctrl.Result{}, err
	} else if !found {
		return ctrl.Result{}, nil
	}

	if err := markReconcilingIfNeeded(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation); err != nil {
		return ctrl.Result{}, err
	}

	// Harbor client
	hc, err := getHarborClient(ctx, r.Client, cr.Namespace, cr.Spec.HarborConnectionRef)
	if err != nil {
		if done, finalErr := finalizeWithoutHarborConnection(ctx, r.Client, &cr, cr.Spec.GetDeletionPolicy(), true, err); done {
			return ctrl.Result{}, finalErr
		}
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}

	// Deletion
	if done, err := finalizeIfDeleting(ctx, r.Client, &cr, cr.Spec.GetDeletionPolicy(), func() error {
		return r.deleteUser(ctx, hc, &cr)
	}); done {
		return ctrl.Result{}, err
	}

	// Finalizer
	if err := ensureFinalizer(ctx, r.Client, &cr); err != nil {
		return ctrl.Result{}, err
	}

	// Defaults & adoption
	if cr.Status.HarborUserID == 0 && cr.Spec.AllowTakeover {
		if ok, err := r.adoptExisting(ctx, hc, &cr); err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		} else if ok {
			r.logger.Info("Adopted user", "ID", cr.Status.HarborUserID)
		}
	}

	// Desired payload
	userPassword, err := r.getUserPassword(ctx, cr)
	if err != nil {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}
	createReq := r.buildCreateReq(cr, userPassword)

	// Create / Update
	if cr.Status.HarborUserID == 0 {
		id, err := hc.CreateUser(ctx, createReq)
		if err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		}
		cr.Status.HarborUserID = id
		if err := setReadyStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, "Created", "User created"); err != nil {
			return ctrl.Result{}, err
		}
		return returnWithDriftDetection(&cr.Spec.HarborSpecBase)
	}

	current, err := hc.GetUserByID(ctx, cr.Status.HarborUserID)
	if err != nil {
		if harborclient.IsNotFound(err) {
			return requeueOnRemoteNotFound(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, func() {
				cr.Status.HarborUserID = 0
			}, "User not found in Harbor")
		}
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}

	if userNeedsUpdate(createReq, current) {
		if err := hc.UpdateUser(ctx, current.UserID, createReq); err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		}
	}
	if err := setReadyStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, "Reconciled", "User reconciled"); err != nil {
		return ctrl.Result{}, err
	}
	return returnWithDriftDetection(&cr.Spec.HarborSpecBase)
}

func (r *UserReconciler) getUserPassword(ctx context.Context, cr harborv1alpha1.User) (string, error) {
	password, err := readSecretValue(ctx, r.Client, harborv1alpha1.SecretReference{
		Name: cr.Spec.PasswordSecretRef.Name,
		Key:  cr.Spec.PasswordSecretRef.Key,
	}, cr.Namespace, "")
	if err != nil {
		return "", fmt.Errorf("failed to read user password secret: %w", err)
	}
	return password, nil
}

func (r *UserReconciler) buildCreateReq(cr harborv1alpha1.User, password string) harborclient.CreateUserRequest {
	realname := cr.Spec.Realname
	if realname == "" {
		realname = cr.Name
	}

	return harborclient.CreateUserRequest{
		Email:    cr.Spec.Email,
		Realname: realname,
		Comment:  cr.Spec.Comment,
		Password: password,
		Username: cr.Name,
	}
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
	users, err := hc.ListUsers(ctx, "username="+cr.Name)
	if err != nil {
		return false, err
	}
	for _, u := range users {
		if strings.EqualFold(u.Username, cr.Name) {
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
	builder, err := setupHarborBackedController(
		mgr,
		&harborv1alpha1.User{},
		func() client.ObjectList { return &harborv1alpha1.UserList{} },
		func(obj client.Object) *harborv1alpha1.HarborConnectionReference {
			return obj.(*harborv1alpha1.User).Spec.HarborConnectionRef
		},
		"user",
	)
	if err != nil {
		return err
	}
	return builder.Complete(r)
}
