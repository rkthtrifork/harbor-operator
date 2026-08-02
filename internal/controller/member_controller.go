package controller

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"

	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
	"github.com/rkthtrifork/harbor-operator/internal/harborclient"
)

// MemberReconciler reconciles a Member object.
type MemberReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	Options OperatorOptions
	logger  logr.Logger
}

type observedMember struct {
	projectID int
	memberID  int
}

// RBAC permissions.
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=members,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=members/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=members/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=projects;users;usergroupclaims,verbs=get;list;watch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections;clusterharborconnections,verbs=get;list;watch

func (r *MemberReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.FromContext(ctx).WithName(fmt.Sprintf("[Member:%s]", req.NamespacedName))

	// Load CR
	var member harborv1alpha1.Member
	if found, err := loadResource(ctx, r.Client, req.NamespacedName, &member, r.logger); err != nil {
		r.logger.Error(err, "Failed to get Member")
		return ctrl.Result{}, err
	} else if !found {
		return ctrl.Result{}, nil
	}

	if err := markReconcilingIfNeeded(ctx, r.Client, &member, &member.Status.HarborStatusBase, member.Generation); err != nil {
		return ctrl.Result{}, err
	}

	// Resolve Harbor connection + typed client
	hc, err := getHarborClientForObject(ctx, r.Options, r.Client, &member, &member.Status.HarborStatusBase, member.Namespace, member.Spec.HarborConnectionRef)
	if err != nil {
		if done, finalErr := finalizeWithoutHarborConnection(ctx, r.Client, &member, member.Spec.GetDeletionPolicy(), true, err); done {
			return ctrl.Result{}, finalErr
		}
		r.logger.Error(err, "Failed to get HarborConnection", "HarborConnectionRef", member.Spec.HarborConnectionRef)
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &member, &member.Status.HarborStatusBase, member.Generation, err)
	}

	// Handle deletion with finalizer pattern
	if done, err := finalizeIfDeleting(ctx, r.Client, &member, member.Spec.GetDeletionPolicy(), func() error {
		return r.ensureMemberAbsent(ctx, hc, &member)
	}); done {
		return ctrl.Result{}, err
	}

	// Ensure finalizer is present
	if err := ensureFinalizer(ctx, r.Client, &member); err != nil {
		return ctrl.Result{}, err
	}

	// Convert role name to Harbor role ID.
	roleID, err := convertRoleNameToID(member.Spec.Role)
	if err != nil {
		r.logger.Error(err, "Invalid role", "Role", member.Spec.Role)
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &member, &member.Status.HarborStatusBase, member.Generation, err)
	}

	// Ensure desired member state in Harbor (create/update as needed).
	observed, err := r.ensureMemberPresent(ctx, hc, &member, roleID)
	if err != nil {
		r.logger.Error(err, "Failed to ensure member in Harbor",
			"ProjectRef", member.Spec.ProjectRef,
			"RoleID", roleID)
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &member, &member.Status.HarborStatusBase, member.Generation, err)
	}
	if observed.projectID == 0 || observed.memberID == 0 {
		err := fmt.Errorf("member reconciliation completed without resolved Harbor IDs")
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &member, &member.Status.HarborStatusBase, member.Generation, err)
	}

	statusChanged := member.Status.HarborProjectID != observed.projectID || member.Status.HarborMemberID != observed.memberID
	member.Status.HarborProjectID = observed.projectID
	member.Status.HarborMemberID = observed.memberID
	conditionChanged := markReady(&member.Status.HarborStatusBase, member.Generation, "Reconciled", "Member reconciled")
	if statusChanged || conditionChanged {
		sanitizeOptionalHarborConnectionRef(&member)
		if err := r.Status().Update(ctx, &member); err != nil {
			return ctrl.Result{}, err
		}
	}

	return returnWithDriftDetection(r.Options, &member.Spec.HarborSpecBase)
}

// ensureMemberPresent makes sure the Harbor project member exists and has the desired role.
// Declarative: list existing members and update/create as needed.
func (r *MemberReconciler) ensureMemberPresent(
	ctx context.Context,
	hc *harborclient.Client,
	member *harborv1alpha1.Member,
	roleID int,
) (observedMember, error) {
	projectKey, projectID, err := resolveProject(ctx, r.Options, r.Client, member.Namespace, &member.Spec.ProjectRef)
	if err != nil {
		return observedMember{}, err
	}

	// Determine desired identity (entity type + name) from spec.
	entityType, entityName, reqBody, err := r.desiredEntityFromSpec(ctx, member, roleID)
	if err != nil {
		return observedMember{}, err
	}

	// List members for this project.
	members, err := hc.ListProjectMembers(ctx, projectKey)
	if err != nil {
		return observedMember{}, err
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
		if err := requireCreationAllowed(r.Options, member.Spec.CreationPolicy); err != nil {
			return observedMember{}, err
		}
		newID, err := hc.CreateProjectMember(ctx, projectKey, reqBody)
		if err != nil {
			return observedMember{}, err
		}
		if newID == 0 {
			members, err = hc.ListProjectMembers(ctx, projectKey)
			if err != nil {
				return observedMember{}, err
			}
			for i := range members {
				candidate := &members[i]
				if strings.EqualFold(candidate.EntityType, entityType) && strings.EqualFold(candidate.EntityName, entityName) {
					newID = candidate.ID
					break
				}
			}
			if newID == 0 {
				return observedMember{}, fmt.Errorf("created Harbor project member but could not discover its ID")
			}
		}
		r.logger.Info("Created Harbor project member",
			"ProjectRef", projectKey,
			"EntityType", entityType,
			"EntityName", entityName,
			"RoleID", roleID,
			"MemberID", newID)
		return observedMember{projectID: projectID, memberID: newID}, nil
	}

	// A Member that has already recorded the exact Harbor identity is owned by
	// this CR, even if a transient reconciliation failure currently marks it
	// not ready. Do not require a Ready condition to recover from an outage;
	// the persisted project/member IDs are the ownership evidence. A different
	// or previously unknown membership still requires an adoption policy.
	if !allowsAdoption(r.Options, member.Spec.CreationPolicy) {
		owned := member.Status.HarborProjectID == projectID && member.Status.HarborMemberID == existing.ID
		if !owned {
			cond := meta.FindStatusCondition(member.Status.Conditions, ConditionReady)
			if cond == nil || cond.Status != metav1.ConditionTrue {
				return observedMember{}, fmt.Errorf("member already exists in Harbor and creationPolicy %q does not allow adoption", r.Options.effectiveCreationPolicy(member.Spec.CreationPolicy))
			}
		}
	}

	// Member exists → check if role matches; update if needed.
	if existing.RoleID != roleID {
		if err := hc.UpdateProjectMemberRole(ctx, projectKey, existing.ID, roleID); err != nil {
			return observedMember{}, err
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

	return observedMember{projectID: projectID, memberID: existing.ID}, nil
}

// ensureMemberAbsent ensures that the Harbor project member is removed when the CR is deleted.
func (r *MemberReconciler) ensureMemberAbsent(
	ctx context.Context,
	hc *harborclient.Client,
	member *harborv1alpha1.Member,
) error {
	if member.Status.HarborProjectID != 0 && member.Status.HarborMemberID != 0 {
		err := hc.DeleteProjectMember(ctx, strconv.Itoa(member.Status.HarborProjectID), member.Status.HarborMemberID)
		if harborclient.IsNotFound(err) {
			// Harbor may delete the project before this finalizer runs. The
			// membership is then already gone and deletion is complete.
			return nil
		}
		return err
	}
	if member.Status.HarborProjectID != 0 || member.Status.HarborMemberID != 0 {
		return fmt.Errorf("member status contains incomplete Harbor identity")
	}

	projectKey, _, err := resolveProject(ctx, r.Options, r.Client, member.Namespace, &member.Spec.ProjectRef)
	if err != nil {
		return nil
	}

	entityType, entityName, _, err := r.desiredEntityFromSpec(ctx, member, 0)
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

// desiredEntityFromSpec computes the logical member identity from the CR and
// returns the matching Harbor create payload.
func (r *MemberReconciler) desiredEntityFromSpec(
	ctx context.Context,
	member *harborv1alpha1.Member,
	roleID int,
) (string, string, harborclient.CreateMemberRequest, error) {
	u := member.Spec.MemberUser
	g := member.Spec.MemberGroup

	switch {
	case u == nil && g == nil:
		return "", "", harborclient.CreateMemberRequest{}, fmt.Errorf("exactly one of memberUser or memberGroup must be set (found none)")
	case u != nil && g != nil:
		return "", "", harborclient.CreateMemberRequest{}, fmt.Errorf("exactly one of memberUser or memberGroup must be set (found both)")
	}

	if u != nil {
		username, err := resolveUserName(ctx, r.Options, r.Client, member.Namespace, u.UserRef)
		if err != nil {
			return "", "", harborclient.CreateMemberRequest{}, err
		}
		return "u", username, harborclient.CreateMemberRequest{
			RoleID: roleID,
			MemberUser: &harborclient.MemberUser{
				Username: username,
			},
		}, nil
	}

	group, err := resolveUserGroup(ctx, r.Options, r.Client, member.Namespace, g.GroupClaimRef, member.Status.ResolvedHarborConnection)
	if err != nil {
		return "", "", harborclient.CreateMemberRequest{}, err
	}
	entityName := group.GroupName
	if entityName == "" {
		entityName = group.LDAPGroupDN
	}
	if entityName == "" {
		return "", "", harborclient.CreateMemberRequest{}, fmt.Errorf("resolved UserGroupClaim %q has no Harbor identity", g.GroupClaimRef.Name)
	}
	return "g", entityName, harborclient.CreateMemberRequest{
		RoleID:      roleID,
		MemberGroup: group,
	}, nil
}

// convertRoleNameToID converts a human-readable role name into the corresponding Harbor role ID.
// Mapping: admin=1, developer=2, guest=3, maintainer=4.
func convertRoleNameToID(role string) (int, error) {
	switch strings.ToLower(role) {
	case adminName:
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
	builder, err := setupHarborBackedController(
		mgr,
		r.Options,
		&harborv1alpha1.Member{},
		func() client.ObjectList { return &harborv1alpha1.MemberList{} },
		func(obj client.Object) *harborv1alpha1.HarborConnectionReference {
			return obj.(*harborv1alpha1.Member).Spec.HarborConnectionRef
		},
		"member",
	)
	if err != nil {
		return err
	}
	return builder.Watches(
		&harborv1alpha1.UserGroupClaim{},
		handler.EnqueueRequestsFromMapFunc(func(ctx context.Context, object client.Object) []ctrl.Request {
			claim := object.(*harborv1alpha1.UserGroupClaim)
			var members harborv1alpha1.MemberList
			if err := mgr.GetClient().List(ctx, &members); err != nil {
				return nil
			}
			requests := make([]ctrl.Request, 0)
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
					requests = append(requests, ctrl.Request{NamespacedName: client.ObjectKeyFromObject(member)})
				}
			}
			return requests
		}),
	).Complete(r)
}
