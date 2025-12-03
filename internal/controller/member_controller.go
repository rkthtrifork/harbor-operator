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

func (r *MemberReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.FromContext(ctx).WithName(fmt.Sprintf("[Member:%s]", req.NamespacedName))

	// Load CR
	var member harborv1alpha1.Member
	if err := r.Get(ctx, req.NamespacedName, &member); err != nil {
		if errors.IsNotFound(err) {
			r.logger.V(1).Info("Member resource disappeared")
			return ctrl.Result{}, nil
		}
		r.logger.Error(err, "Failed to get Member")
		return ctrl.Result{}, err
	}

	// Resolve Harbor connection + typed client
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

	// Handle deletion with finalizer pattern
	if !member.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(&member, finalizerName) {
			if err := r.ensureMemberAbsent(ctx, hc, &member); err != nil {
				return ctrl.Result{}, err
			}
			controllerutil.RemoveFinalizer(&member, finalizerName)
			if err := r.Update(ctx, &member); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Ensure finalizer is present
	if !controllerutil.ContainsFinalizer(&member, finalizerName) {
		controllerutil.AddFinalizer(&member, finalizerName)
		if err := r.Update(ctx, &member); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Convert role name to Harbor role ID.
	roleID, err := convertRoleNameToID(member.Spec.Role)
	if err != nil {
		r.logger.Error(err, "Invalid role", "Role", member.Spec.Role)
		return ctrl.Result{}, err
	}

	// Ensure desired member state in Harbor (create/update as needed).
	if err := r.ensureMemberPresent(ctx, hc, &member, roleID); err != nil {
		r.logger.Error(err, "Failed to ensure member in Harbor",
			"ProjectRef", member.Spec.ProjectRef,
			"RoleID", roleID)
		return ctrl.Result{}, err
	}

	return returnWithDriftDetection(&member.Spec.HarborSpecBase)
}

// ensureMemberPresent makes sure the Harbor project member exists and has the desired role.
// Declarative: list existing members and update/create as needed.
func (r *MemberReconciler) ensureMemberPresent(
	ctx context.Context,
	hc *harborclient.Client,
	member *harborv1alpha1.Member,
	roleID int,
) error {
	projectKey := member.Spec.ProjectRef
	if projectKey == "" {
		return fmt.Errorf("spec.projectRef must not be empty")
	}

	// Determine desired identity (entity type + name) from spec.
	entityType, entityName, err := desiredEntityFromSpec(member)
	if err != nil {
		return err
	}

	// List members for this project.
	members, err := hc.ListProjectMembers(ctx, projectKey)
	if err != nil {
		return err
	}

	// Find existing membership for this identity.
	var existing *harborclient.ProjectMember
	for i := range members {
		m := &members[i]
		if strings.EqualFold(m.EntityType, entityType) &&
			strings.EqualFold(m.EntityName, entityName) {
			existing = m
			break
		}
	}

	if existing == nil {
		// Member does not exist → create it.
		reqBody := buildMemberCreateRequest(member, roleID)
		newID, err := hc.CreateProjectMember(ctx, projectKey, reqBody)
		if err != nil {
			return err
		}
		if newID != 0 {
			r.logger.Info("Created Harbor project member",
				"ProjectRef", projectKey,
				"EntityType", entityType,
				"EntityName", entityName,
				"RoleID", roleID,
				"MemberID", newID)
		} else {
			r.logger.Info("Created Harbor project member (no member ID returned)",
				"ProjectRef", projectKey,
				"EntityType", entityType,
				"EntityName", entityName,
				"RoleID", roleID)
		}
		return nil
	}

	// Member exists → check if role matches; update if needed.
	if existing.RoleID != roleID {
		if err := hc.UpdateProjectMemberRole(ctx, projectKey, existing.ID, roleID); err != nil {
			return err
		}
		r.logger.Info("Updated Harbor project member role",
			"ProjectRef", projectKey,
			"EntityType", entityType,
			"EntityName", entityName,
			"OldRoleID", existing.RoleID,
			"NewRoleID", roleID,
			"MemberID", existing.ID)
	} else {
		r.logger.V(2).Info("Harbor project member already up to date",
			"ProjectRef", projectKey,
			"EntityType", entityType,
			"EntityName", entityName,
			"RoleID", roleID,
			"MemberID", existing.ID)
	}

	return nil
}

// ensureMemberAbsent ensures that the Harbor project member is removed when the CR is deleted.
func (r *MemberReconciler) ensureMemberAbsent(
	ctx context.Context,
	hc *harborclient.Client,
	member *harborv1alpha1.Member,
) error {
	projectKey := member.Spec.ProjectRef
	if projectKey == "" {
		// nothing we can do; treat as already gone
		return nil
	}

	entityType, entityName, err := desiredEntityFromSpec(member)
	if err != nil {
		return err
	}

	members, err := hc.ListProjectMembers(ctx, projectKey)
	if harborclient.IsNotFound(err) {
		// Project or membership list gone → nothing to delete.
		r.logger.V(1).Info("Project not found in Harbor when deleting member; assuming already removed",
			"ProjectRef", projectKey)
		return nil
	} else if err != nil {
		return err
	}

	removedAny := false
	for _, pm := range members {
		if strings.EqualFold(pm.EntityType, entityType) &&
			strings.EqualFold(pm.EntityName, entityName) {
			if err := hc.DeleteProjectMember(ctx, projectKey, pm.ID); err != nil {
				if harborclient.IsNotFound(err) {
					// Already gone; ignore.
					continue
				}
				return err
			}
			removedAny = true
			r.logger.Info("Deleted Harbor project member",
				"ProjectRef", projectKey,
				"EntityType", entityType,
				"EntityName", entityName,
				"MemberID", pm.ID)
		}
	}

	if !removedAny {
		r.logger.V(1).Info("No matching Harbor project member found to delete",
			"ProjectRef", projectKey,
			"EntityType", entityType,
			"EntityName", entityName)
	}

	return nil
}

// desiredEntityFromSpec computes the logical member identity from the CR.
// It enforces that exactly one of member_user or member_group is set.
func desiredEntityFromSpec(member *harborv1alpha1.Member) (string, string, error) {
	u := member.Spec.MemberUser
	g := member.Spec.MemberGroup

	switch {
	case u == nil && g == nil:
		return "", "", fmt.Errorf("exactly one of member_user or member_group must be set (found none)")
	case u != nil && g != nil:
		return "", "", fmt.Errorf("exactly one of member_user or member_group must be set (found both)")
	}

	if u != nil {
		// Users → entity_type "u". Use username as stable identity key.
		if u.Username == "" {
			return "", "", fmt.Errorf("member_user.username must be set")
		}
		return "u", u.Username, nil
	}

	// Groups → entity_type "g".
	if g.GroupName == "" && g.LDAPGroupDN == "" {
		return "", "", fmt.Errorf("member_group must specify group_name or ldap_group_dn")
	}

	// Prefer group_name as primary identity. If only DN is provided, fall back to it.
	if g.GroupName != "" {
		return "g", g.GroupName, nil
	}
	return "g", g.LDAPGroupDN, nil
}

// buildMemberCreateRequest constructs the payload for the Harbor member creation call.
// It passes through user/group fields and the resolved role ID.
func buildMemberCreateRequest(member *harborv1alpha1.Member, roleID int) harborclient.CreateMemberRequest {
	var user *harborclient.MemberUser
	var group *harborclient.MemberGroup

	if member.Spec.MemberUser != nil {
		user = &harborclient.MemberUser{
			Username: member.Spec.MemberUser.Username,
		}
	}
	if member.Spec.MemberGroup != nil {
		group = &harborclient.MemberGroup{
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

// SetupWithManager sets up the controller with the Manager.
func (r *MemberReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&harborv1alpha1.Member{}).
		Named("member").
		Complete(r)
}
