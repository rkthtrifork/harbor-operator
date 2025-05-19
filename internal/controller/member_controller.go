package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
	"github.com/rkthtrifork/harbor-operator/internal/harborclient"
)

// -----------------------------------------------------------------------------
// Reconciler
// -----------------------------------------------------------------------------

type MemberReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	logger   logr.Logger
	recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=members,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=members/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections,verbs=get;list;watch

func (r *MemberReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.FromContext(ctx).WithName(fmt.Sprintf("[Member:%s]", req.NamespacedName))

	//---------------------------------------------------------------------
	// Load CR
	//---------------------------------------------------------------------
	var cr harborv1alpha1.Member
	if err := r.Get(ctx, req.NamespacedName, &cr); err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	//---------------------------------------------------------------------
	// Mark Reconciling
	//---------------------------------------------------------------------
	r.markReconciling(&cr)
	_ = r.Status().Update(ctx, &cr)

	//---------------------------------------------------------------------
	// Harbor client
	//---------------------------------------------------------------------
	conn, err := getHarborConnection(ctx, r.Client, cr.Namespace, cr.Spec.HarborConnectionRef)
	if err != nil {
		r.fail(&cr, "NoConnection", err.Error())
		return ctrl.Result{}, err
	}
	user, pass, err := getHarborAuth(ctx, r.Client, conn)
	if err != nil {
		r.fail(&cr, "SecretError", err.Error())
		return ctrl.Result{}, err
	}
	hc := harborclient.New(conn.Spec.BaseURL, user, pass)

	//---------------------------------------------------------------------
	// Deletion
	//---------------------------------------------------------------------
	if !cr.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&cr, finalizerName) {
			if err := r.deleteMember(ctx, hc, &cr); err != nil {
				r.fail(&cr, "DeleteError", err.Error())
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(&cr, finalizerName)
			_ = r.Update(ctx, &cr)
			r.recorder.Event(&cr, corev1.EventTypeNormal, "Deleted", "Member removed from Harbor")
		}
		return ctrl.Result{}, nil
	}

	//---------------------------------------------------------------------
	// Finalizer
	//---------------------------------------------------------------------
	if !controllerutil.ContainsFinalizer(&cr, finalizerName) {
		controllerutil.AddFinalizer(&cr, finalizerName)
		_ = r.Update(ctx, &cr)
	}

	//---------------------------------------------------------------------
	// Defaults & adoption
	//---------------------------------------------------------------------
	if cr.Status.HarborMemberID == 0 && cr.Spec.AllowTakeover {
		if ok, err := r.adoptExisting(ctx, hc, &cr); err != nil {
			r.fail(&cr, "AdoptionError", err.Error())
			return ctrl.Result{}, err
		} else if ok {
			r.logger.Info("Adopted member", "ID", cr.Status.HarborMemberID)
			r.recorder.Event(&cr, corev1.EventTypeNormal, "Adopted",
				fmt.Sprintf("Existing member adopted (ID=%d)", cr.Status.HarborMemberID))
		}
	}

	//---------------------------------------------------------------------
	// Desired payload
	//---------------------------------------------------------------------
	roleID, ok := roleNameToID[strings.ToLower(cr.Spec.Role)]
	if !ok {
		err := fmt.Errorf("unknown role %q", cr.Spec.Role)
		r.fail(&cr, "InvalidRole", err.Error())
		return ctrl.Result{}, err
	}
	createReq := r.buildCreateReq(cr, roleID)

	//---------------------------------------------------------------------
	// Create / Update
	//---------------------------------------------------------------------
	if cr.Status.HarborMemberID == 0 {
		id, err := hc.CreateProjectMember(ctx, cr.Spec.Project, createReq)
		if err != nil {
			r.fail(&cr, "CreateError", err.Error())
			return ctrl.Result{}, err
		}
		cr.Status.HarborMemberID = id
		_ = r.Status().Update(ctx, &cr)
		r.recorder.Event(&cr, corev1.EventTypeNormal, "Created",
			fmt.Sprintf("Member created in Harbor (ID=%d)", id))

		r.markReady(&cr)
		_ = r.Status().Update(ctx, &cr)
		return returnWithDriftDetection(&cr.Spec.HarborSpecBase)
	}

	current, err := hc.GetProjectMember(ctx, cr.Spec.Project, cr.Status.HarborMemberID)
	if err != nil {
		if harborclient.IsNotFound(err) {
			cr.Status.HarborMemberID = 0
			_ = r.Status().Update(ctx, &cr)
			return ctrl.Result{Requeue: true}, nil
		}
		r.fail(&cr, "GetError", err.Error())
		return ctrl.Result{}, err
	}

	if current.RoleID != roleID {
		if err := hc.UpdateProjectMemberRole(ctx, cr.Spec.Project, current.ID, roleID); err != nil {
			r.fail(&cr, "UpdateError", err.Error())
			return ctrl.Result{}, err
		}
		r.recorder.Event(&cr, corev1.EventTypeNormal, "Updated",
			fmt.Sprintf("Member role updated in Harbor (ID=%d)", current.ID))
	}

	//---------------------------------------------------------------------
	// Success
	//---------------------------------------------------------------------
	r.markReady(&cr)
	_ = r.Status().Update(ctx, &cr)
	return returnWithDriftDetection(&cr.Spec.HarborSpecBase)
}

// -----------------------------------------------------------------------------
// Role mapping helpers
// -----------------------------------------------------------------------------

// Harbor core role IDs (project scope):
//
//	1 projectAdmin, 2 maintainer, 3 developer, 4 guest, 5 limitedGuest
var roleNameToID = map[string]int{
	"projectadmin": 1,
	"maintainer":   2,
	"developer":    3,
	"guest":        4,
	"limitedguest": 5,
}

// -----------------------------------------------------------------------------
// Status helpers
// -----------------------------------------------------------------------------

func (r *MemberReconciler) markReconciling(cr *harborv1alpha1.Member) {
	harborv1alpha1.SetStatusCondition(&cr.Status.Conditions, metav1.Condition{
		Type:    harborv1alpha1.ConditionReconciling,
		Status:  metav1.ConditionTrue,
		Reason:  "Reconciling",
		Message: "Reconciling resource",
	})
	harborv1alpha1.RemoveCondition(&cr.Status.Conditions, harborv1alpha1.ConditionStalled)
	harborv1alpha1.RemoveCondition(&cr.Status.Conditions, harborv1alpha1.ConditionReady)
	cr.Status.ObservedGeneration = cr.Generation
}

func (r *MemberReconciler) markReady(cr *harborv1alpha1.Member) {
	harborv1alpha1.SetStatusCondition(&cr.Status.Conditions, metav1.Condition{
		Type:    harborv1alpha1.ConditionReady,
		Status:  metav1.ConditionTrue,
		Reason:  "Reconciled",
		Message: "Resource is ready",
	})
	harborv1alpha1.RemoveCondition(&cr.Status.Conditions, harborv1alpha1.ConditionReconciling)
	harborv1alpha1.RemoveCondition(&cr.Status.Conditions, harborv1alpha1.ConditionStalled)
	cr.Status.ObservedGeneration = cr.Generation
}

func (r *MemberReconciler) fail(cr *harborv1alpha1.Member, reason, msg string) {
	harborv1alpha1.SetStatusCondition(&cr.Status.Conditions, metav1.Condition{
		Type:    harborv1alpha1.ConditionStalled,
		Status:  metav1.ConditionTrue,
		Reason:  reason,
		Message: msg,
	})
	harborv1alpha1.RemoveCondition(&cr.Status.Conditions, harborv1alpha1.ConditionReconciling)
	harborv1alpha1.RemoveCondition(&cr.Status.Conditions, harborv1alpha1.ConditionReady)
	cr.Status.ObservedGeneration = cr.Generation
	_ = r.Status().Update(context.TODO(), cr)
	r.recorder.Event(cr, corev1.EventTypeWarning, reason, msg)
}

// -----------------------------------------------------------------------------
// CRUD helpers
// -----------------------------------------------------------------------------

func (r *MemberReconciler) buildCreateReq(cr harborv1alpha1.Member, roleID int) harborclient.CreateMemberRequest {
	switch strings.ToLower(cr.Spec.Kind) {
	case "user":
		return harborclient.CreateMemberRequest{
			RoleID: roleID,
			MemberUser: &harborclient.MemberUser{
				Username: cr.Spec.Name,
			},
		}
	case "group":
		return harborclient.CreateMemberRequest{
			RoleID: roleID,
			MemberGroup: &harborclient.MemberGroup{
				GroupName: cr.Spec.Name,
				GroupType: 1, // 1 = LDAP, 2 = Harbor internal
			},
		}
	case "robot":
		// Robot accounts are treated as users in the member API.
		return harborclient.CreateMemberRequest{
			RoleID: roleID,
			MemberUser: &harborclient.MemberUser{
				Username: cr.Spec.Name,
			},
		}
	default:
		// Should never happen – enum validation on CRD
		return harborclient.CreateMemberRequest{}
	}
}

func (r *MemberReconciler) deleteMember(ctx context.Context, hc *harborclient.Client, cr *harborv1alpha1.Member) error {
	if cr.Status.HarborMemberID == 0 {
		return nil
	}
	return hc.DeleteProjectMember(ctx, cr.Spec.Project, cr.Status.HarborMemberID)
}

func (r *MemberReconciler) adoptExisting(ctx context.Context, hc *harborclient.Client, cr *harborv1alpha1.Member) (bool, error) {
	members, err := hc.ListProjectMembers(ctx, cr.Spec.Project)
	if err != nil {
		return false, err
	}
	for _, m := range members {
		if strings.EqualFold(m.EntityName, cr.Spec.Name) &&
			strings.EqualFold(m.EntityType, cr.Spec.Kind) {
			cr.Status.HarborMemberID = m.ID
			return true, r.Status().Update(ctx, cr)
		}
	}
	return false, nil
}

// -----------------------------------------------------------------------------
// Setup
// -----------------------------------------------------------------------------

func (r *MemberReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.recorder = mgr.GetEventRecorderFor("harbor-operator")
	return ctrl.NewControllerManagedBy(mgr).
		For(&harborv1alpha1.Member{}).
		Named("member").
		Complete(r)
}
