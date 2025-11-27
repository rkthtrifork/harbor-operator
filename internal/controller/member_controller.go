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
	"sigs.k8s.io/controller-runtime/pkg/log"

	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
	"github.com/rkthtrifork/harbor-operator/internal/harborclient"
)

// MemberReconciler reconciles a Member object.
type MemberReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	logger logr.Logger
}

// RBAC permissions.
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=members,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=members/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections,verbs=get;list;watch

// Reconcile implements the reconciliation loop for the Member resource.
func (r *MemberReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.FromContext(ctx).WithName(fmt.Sprintf("[Member:%s]", req.NamespacedName))

	// Fetch the Member instance.
	var member harborv1alpha1.Member
	if err := r.Get(ctx, req.NamespacedName, &member); err != nil {
		if errors.IsNotFound(err) {
			r.logger.Info("Member resource not found; it may have been deleted")
			return ctrl.Result{}, nil
		}
		r.logger.Error(err, "Failed to get Member")
		return ctrl.Result{}, err
	}

	// Resolve HarborConnection and credentials using shared helpers from common.go.
	conn, err := getHarborConnection(ctx, r.Client, member.Namespace, member.Spec.HarborConnectionRef)
	if err != nil {
		r.logger.Error(err, "Failed to get HarborConnection", "HarborConnectionRef", member.Spec.HarborConnectionRef)
		return ctrl.Result{}, err
	}

	if conn.Spec.Credentials == nil {
		err := fmt.Errorf("HarborConnection %s/%s has no credentials configured", conn.Namespace, conn.Name)
		r.logger.Error(err, "Cannot manage Harbor members without credentials")
		return ctrl.Result{}, err
	}

	user, pass, err := getHarborAuth(ctx, r.Client, conn)
	if err != nil {
		r.logger.Error(err, "Failed to get Harbor authentication credentials")
		return ctrl.Result{}, err
	}

	hc := harborclient.New(conn.Spec.BaseURL, user, pass)

	// Convert role name to Harbor role ID.
	roleID, err := convertRoleNameToID(member.Spec.Role)
	if err != nil {
		r.logger.Error(err, "Invalid role", "Role", member.Spec.Role)
		return ctrl.Result{}, err
	}

	// Build the Harbor member payload.
	reqBody := buildMemberRequest(&member, roleID)

	// Delegate the actual HTTP call to the Harbor client.
	// projectRef is the Harbor project ID or name, as accepted by the Harbor API.
	memberID, err := hc.CreateProjectMember(ctx, member.Spec.ProjectRef, reqBody)
	if err != nil {
		r.logger.Error(err, "Failed to ensure member in Harbor",
			"ProjectRef", member.Spec.ProjectRef,
			"RoleID", roleID)
		return ctrl.Result{}, err
	}

	if memberID != 0 {
		r.logger.Info("Successfully created member in Harbor",
			"ProjectRef", member.Spec.ProjectRef,
			"Role", member.Spec.Role,
			"MemberID", memberID)
	} else {
		// memberID==0 may mean "already existed" if you treat 409 as success in the client.
		r.logger.Info("Member already exists or created without known ID",
			"ProjectRef", member.Spec.ProjectRef,
			"Role", member.Spec.Role)
	}

	// You could update Status here later if you start tracking Harbor member IDs.
	return ctrl.Result{}, nil
}

// convertRoleNameToID converts a human-readable role name into the corresponding Harbor role ID.
// Mapping: "admin": 1, "developer": 2, "guest": 3, "maintainer": 4.
func convertRoleNameToID(role string) (int, error) {
	switch strings.ToLower(role) {
	case "admin":
		return 1, nil
	case "developer":
		return 2, nil
	case "guest":
		return 3, nil
	case "maintainer":
		return 4, nil
	default:
		return 0, fmt.Errorf("unsupported role: %s", role)
	}
}

// buildMemberRequest constructs the payload for the Harbor member creation/update call.
func buildMemberRequest(member *harborv1alpha1.Member, roleID int) harborclient.CreateMemberRequest {
	var user *harborclient.MemberUser
	var group *harborclient.MemberGroup

	if member.Spec.MemberUser != nil {
		user = &harborclient.MemberUser{
			UserID:   member.Spec.MemberUser.UserID,
			Username: member.Spec.MemberUser.Username,
		}
	}
	if member.Spec.MemberGroup != nil {
		group = &harborclient.MemberGroup{
			ID:          member.Spec.MemberGroup.ID,
			GroupName:   member.Spec.MemberGroup.GroupName,
			GroupType:   member.Spec.MemberGroup.GroupType,
			LDAPGroupDN: member.Spec.MemberGroup.LDAPGroupDN,
		}
	}

	return harborclient.CreateMemberRequest{
		RoleID:      roleID,
		MemberUser:  user,
		MemberGroup: group,
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *MemberReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&harborv1alpha1.Member{}).
		Named("member").
		Complete(r)
}
