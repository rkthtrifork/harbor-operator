package controller

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
	"github.com/rkthtrifork/harbor-operator/internal/harborclient"
)

type RobotReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	logger logr.Logger
}

// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=robots,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=robots/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=robots/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections;clusterharborconnections,verbs=get;list;watch

func (r *RobotReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.FromContext(ctx).WithName(fmt.Sprintf("[Robot:%s]", req.NamespacedName))

	var cr harborv1alpha1.Robot
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

	if done, err := finalizeIfDeleting(ctx, r.Client, &cr, func() error {
		return r.deleteRobot(ctx, hc, &cr)
	}); done {
		return ctrl.Result{}, err
	}

	if err := ensureFinalizer(ctx, r.Client, &cr); err != nil {
		return ctrl.Result{}, err
	}

	cr.Spec.Name = defaultString(cr.Spec.Name, cr.Name)

	if err := validateRobotSpec(&cr); err != nil {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}

	secretRef, err := resolveRobotSecretRef(&cr)
	if err != nil {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}

	if cr.Status.HarborRobotID == 0 && cr.Spec.AllowTakeover {
		if ok, err := r.adoptExisting(ctx, hc, &cr); err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		} else if ok {
			r.logger.Info("Adopted robot", "ID", cr.Status.HarborRobotID)
		}
	}

	if cr.Status.HarborRobotID == 0 {
		return r.createRobot(ctx, hc, &cr, secretRef)
	}

	return r.reconcileExisting(ctx, hc, &cr, secretRef)
}

func (r *RobotReconciler) createRobot(
	ctx context.Context,
	hc *harborclient.Client,
	cr *harborv1alpha1.Robot,
	secretRef harborv1alpha1.SecretReference,
) (ctrl.Result, error) {
	createReq := buildRobotCreateRequest(cr)
	created, err := hc.CreateRobot(ctx, createReq)
	if err != nil {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}
	cr.Status.HarborRobotID = created.ID
	storedSecret := created.Secret
	if storedSecret == "" {
		storedSecret, err = rotateRobotSecret(ctx, hc, cr.Status.HarborRobotID)
		if err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		}
	}
	if err := upsertOwnedSecretValue(ctx, r.Client, cr, "Robot", secretRef.Namespace, secretRef.Name, secretRef.Key, storedSecret); err != nil {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}
	now := metav1.Now()
	cr.Status.LastRotatedAt = &now
	if err := setReadyStatus(ctx, r.Client, cr, &cr.Status.HarborStatusBase, cr.Generation, "Created", "Robot created"); err != nil {
		return ctrl.Result{}, err
	}
	r.logger.Info("Created robot", "ID", created.ID)
	return returnWithDriftDetection(&cr.Spec.HarborSpecBase)
}

func (r *RobotReconciler) reconcileExisting(
	ctx context.Context,
	hc *harborclient.Client,
	cr *harborv1alpha1.Robot,
	secretRef harborv1alpha1.SecretReference,
) (ctrl.Result, error) {
	current, err := hc.GetRobotByID(ctx, cr.Status.HarborRobotID)
	if err != nil {
		if harborclient.IsNotFound(err) {
			return requeueOnRemoteNotFound(ctx, r.Client, cr, &cr.Status.HarborStatusBase, cr.Generation, func() {
				cr.Status.HarborRobotID = 0
			}, "Robot not found in Harbor")
		}
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}

	if !robotLevelMatches(cr.Spec.Level, current.Level) {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, cr, &cr.Status.HarborStatusBase, cr.Generation, fmt.Errorf("robot level mismatch: desired %q, current %q", cr.Spec.Level, current.Level))
	}

	desired := buildRobotUpdateRequest(cr, current)
	if robotNeedsUpdate(desired, current) {
		if err := hc.UpdateRobot(ctx, current.ID, desired); err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		}
		r.logger.Info("Updated robot", "ID", current.ID)
	}

	statusChanged := updateRobotExpiryStatus(cr, current.ExpiresAt)

	if shouldRotateRobot(cr) {
		rotatedSecret, err := rotateRobotSecret(ctx, hc, current.ID)
		if err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		}
		if err := upsertOwnedSecretValue(ctx, r.Client, cr, "Robot", secretRef.Namespace, secretRef.Name, secretRef.Key, rotatedSecret); err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		}
		now := metav1.Now()
		cr.Status.LastRotatedAt = &now
		refreshed, err := hc.GetRobotByID(ctx, current.ID)
		if err == nil {
			updateRobotExpiryStatus(cr, refreshed.ExpiresAt)
		}
		statusChanged = true
		r.logger.Info("Rotated robot secret", "ID", current.ID)
	}

	condChanged := markReady(&cr.Status.HarborStatusBase, cr.Generation, "Reconciled", "Robot reconciled")
	if statusChanged || condChanged {
		if err := r.Status().Update(ctx, cr); err != nil {
			return ctrl.Result{}, err
		}
	}

	return returnWithDriftDetection(&cr.Spec.HarborSpecBase)
}
func (r *RobotReconciler) deleteRobot(ctx context.Context, hc *harborclient.Client, cr *harborv1alpha1.Robot) error {
	if cr.Status.HarborRobotID == 0 {
		return nil
	}
	return hc.DeleteRobot(ctx, cr.Status.HarborRobotID)
}

func (r *RobotReconciler) adoptExisting(ctx context.Context, hc *harborclient.Client, cr *harborv1alpha1.Robot) (bool, error) {
	query := "name=" + cr.Spec.Name
	robots, err := hc.ListRobots(ctx, query)
	if err != nil {
		return false, err
	}
	if ok, err := r.findAndAdoptRobot(ctx, cr, robots); ok || err != nil {
		return ok, err
	}

	robots, err = hc.ListRobots(ctx, "")
	if err != nil {
		return false, err
	}
	return r.findAndAdoptRobot(ctx, cr, robots)
}

func (r *RobotReconciler) findAndAdoptRobot(ctx context.Context, cr *harborv1alpha1.Robot, robots []harborclient.Robot) (bool, error) {
	for _, robot := range robots {
		if robotNameMatches(cr.Spec.Name, robot.Name) && robotLevelMatches(cr.Spec.Level, robot.Level) {
			cr.Status.HarborRobotID = robot.ID
			return true, r.Status().Update(ctx, cr)
		}
	}
	return false, nil
}

func validateRobotSpec(cr *harborv1alpha1.Robot) error {
	if cr.Spec.Level == "" {
		return fmt.Errorf("spec.level is required")
	}
	if len(cr.Spec.Permissions) == 0 {
		return fmt.Errorf("spec.permissions must contain at least one permission")
	}
	if cr.Spec.Duration == 0 {
		return fmt.Errorf("spec.duration must be either -1 or a positive integer")
	}
	return nil
}

func resolveRobotSecretRef(cr *harborv1alpha1.Robot) (harborv1alpha1.SecretReference, error) {
	if cr.Spec.SecretRef == nil {
		return harborv1alpha1.SecretReference{
			Name:      fmt.Sprintf("%s-secret", cr.Name),
			Key:       "secret",
			Namespace: cr.Namespace,
		}, nil
	}
	if cr.Spec.SecretRef.Name == "" {
		return harborv1alpha1.SecretReference{}, fmt.Errorf("spec.secretRef.name is required when secretRef is set")
	}
	ref := *cr.Spec.SecretRef
	if ref.Namespace == "" {
		ref.Namespace = cr.Namespace
	}
	if ref.Key == "" {
		ref.Key = "secret"
	}
	return ref, nil
}

func buildRobotCreateRequest(cr *harborv1alpha1.Robot) harborclient.RobotCreateRequest {
	return harborclient.RobotCreateRequest{
		Name:        cr.Spec.Name,
		Description: cr.Spec.Description,
		Secret:      "",
		Level:       cr.Spec.Level,
		Disable:     cr.Spec.Disable,
		Duration:    &cr.Spec.Duration,
		Permissions: buildRobotPermissions(cr),
	}
}

func buildRobotUpdateRequest(cr *harborv1alpha1.Robot, current *harborclient.Robot) harborclient.Robot {
	duration := cr.Spec.Duration
	if duration == 0 {
		duration = -1
	}
	desired := harborclient.Robot{
		ID:          current.ID,
		Name:        current.Name,
		Description: cr.Spec.Description,
		Level:       current.Level,
		Disable:     current.Disable,
		Duration:    current.Duration,
		Permissions: buildRobotPermissions(cr),
	}
	if cr.Spec.Disable != nil {
		desired.Disable = *cr.Spec.Disable
	}
	desired.Duration = &duration
	return desired
}

func robotNeedsUpdate(desired harborclient.Robot, current *harborclient.Robot) bool {
	if desired.Description != current.Description {
		return true
	}
	if desired.Disable != current.Disable {
		return true
	}
	if !intPtrEqual(desired.Duration, current.Duration) {
		return true
	}
	return !robotPermissionsEqual(desired.Permissions, current.Permissions)
}

func robotPermissionsEqual(a, b []harborclient.RobotPermission) bool {
	normA := normalizeRobotPermissions(a)
	normB := normalizeRobotPermissions(b)
	if len(normA) != len(normB) {
		return false
	}
	for i := range normA {
		if normA[i].Kind != normB[i].Kind || normA[i].Namespace != normB[i].Namespace {
			return false
		}
		if len(normA[i].Access) != len(normB[i].Access) {
			return false
		}
		for j := range normA[i].Access {
			if normA[i].Access[j] != normB[i].Access[j] {
				return false
			}
		}
	}
	return true
}

func normalizeRobotPermissions(perms []harborclient.RobotPermission) []harborclient.RobotPermission {
	out := make([]harborclient.RobotPermission, 0, len(perms))
	for _, perm := range perms {
		access := make([]harborclient.Access, len(perm.Access))
		copy(access, perm.Access)
		sort.Slice(access, func(i, j int) bool {
			if access[i].Resource != access[j].Resource {
				return access[i].Resource < access[j].Resource
			}
			if access[i].Action != access[j].Action {
				return access[i].Action < access[j].Action
			}
			return access[i].Effect < access[j].Effect
		})
		out = append(out, harborclient.RobotPermission{
			Kind:      perm.Kind,
			Namespace: perm.Namespace,
			Access:    access,
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].Namespace < out[j].Namespace
	})
	return out
}

func buildRobotPermissions(cr *harborv1alpha1.Robot) []harborclient.RobotPermission {
	perms := make([]harborclient.RobotPermission, 0, len(cr.Spec.Permissions))
	for _, perm := range cr.Spec.Permissions {
		access := make([]harborclient.Access, 0, len(perm.Access))
		for _, rule := range perm.Access {
			effect := rule.Effect
			if effect == "" {
				effect = "allow"
			}
			access = append(access, harborclient.Access{
				Resource: string(rule.Resource),
				Action:   string(rule.Action),
				Effect:   effect,
			})
		}
		perms = append(perms, harborclient.RobotPermission{
			Kind:      perm.Kind,
			Namespace: perm.Namespace,
			Access:    access,
		})
	}
	return perms
}

func robotLevelMatches(desired, current string) bool {
	return strings.EqualFold(desired, current)
}

func robotNameMatches(desired, current string) bool {
	if strings.EqualFold(desired, current) {
		return true
	}
	if strings.HasSuffix(strings.ToLower(current), "$"+strings.ToLower(desired)) {
		return true
	}
	if strings.HasSuffix(strings.ToLower(current), "+"+strings.ToLower(desired)) {
		return true
	}
	return false
}

func intPtrEqual(a, b *int) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func shouldRotateRobot(cr *harborv1alpha1.Robot) bool {
	if cr.Status.ExpiresAt == nil {
		return false
	}
	return time.Now().After(cr.Status.ExpiresAt.Time)
}

func rotateRobotSecret(ctx context.Context, hc *harborclient.Client, robotID int) (string, error) {
	sec, err := hc.RefreshRobotSecret(ctx, robotID, "")
	if err != nil {
		return "", err
	}
	if sec.Secret == "" {
		return "", fmt.Errorf("harbor did not return a robot secret")
	}
	return sec.Secret, nil
}

func updateRobotExpiryStatus(cr *harborv1alpha1.Robot, expiresAt int) bool {
	var updated bool
	if expiresAt > 0 {
		exp := metav1.NewTime(time.Unix(int64(expiresAt), 0))
		if cr.Status.ExpiresAt == nil || !cr.Status.ExpiresAt.Equal(&exp) {
			cr.Status.ExpiresAt = &exp
			updated = true
		}
	} else if cr.Status.ExpiresAt != nil {
		cr.Status.ExpiresAt = nil
		updated = true
	}
	return updated
}

func (r *RobotReconciler) SetupWithManager(mgr ctrl.Manager) error {
	builder, err := setupHarborBackedController(
		mgr,
		&harborv1alpha1.Robot{},
		func() client.ObjectList { return &harborv1alpha1.RobotList{} },
		func(obj client.Object) harborv1alpha1.HarborConnectionReference {
			return obj.(*harborv1alpha1.Robot).Spec.HarborConnectionRef
		},
		"robot",
	)
	if err != nil {
		return err
	}
	return builder.Complete(r)
}
