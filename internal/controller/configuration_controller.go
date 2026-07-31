package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
	"github.com/rkthtrifork/harbor-operator/internal/harborclient"
)

type ConfigurationReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	Options OperatorOptions
	logger  logr.Logger
}

// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=configurations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=configurations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=configurations/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections;clusterharborconnections,verbs=get;list;watch

func (r *ConfigurationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.FromContext(ctx).WithName(fmt.Sprintf("[Configuration:%s]", req.NamespacedName))

	var cr harborv1alpha1.Configuration
	if found, err := loadResource(ctx, r.Client, req.NamespacedName, &cr, r.logger); err != nil {
		return ctrl.Result{}, err
	} else if !found {
		return ctrl.Result{}, nil
	}

	if err := markReconcilingIfNeeded(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation); err != nil {
		return ctrl.Result{}, err
	}

	hc, err := getHarborClient(ctx, r.Options, r.Client, cr.Namespace, cr.Spec.HarborConnectionRef)
	if err != nil {
		if done, finalErr := finalizeWithoutHarborConnection(ctx, r.Client, &cr, cr.Spec.GetDeletionPolicy(), false, err); done {
			return ctrl.Result{}, finalErr
		}
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}

	if done, err := finalizeIfDeleting(ctx, r.Client, &cr, cr.Spec.GetDeletionPolicy(), nil); done {
		return ctrl.Result{}, err
	}

	if err := ensureFinalizer(ctx, r.Client, &cr); err != nil {
		return ctrl.Result{}, err
	}
	if err := ensureConfigurationSingletonOwner(ctx, r.Options, r.Client, &cr); err != nil {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}

	desired, err := r.buildDesiredSettings(ctx, &cr)
	if err != nil {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}
	if len(desired) == 0 {
		r.logger.V(1).Info("No configuration settings specified; nothing to apply")
		if err := setReadyStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, "Noop", "No configuration changes to apply"); err != nil {
			return ctrl.Result{}, err
		}
		return returnWithDriftDetection(r.Options, &cr.Spec.HarborSpecBase)
	}

	current, err := hc.GetConfigurations(ctx)
	if err != nil {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}

	if err := ensureEditableSettings(desired, current); err != nil {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}

	if configurationNeedsUpdate(desired, current) {
		if err := hc.UpdateConfigurations(ctx, desired); err != nil {
			return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
		}
		r.logger.Info("Updated Harbor configurations")
	}

	if err := setReadyStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, "Reconciled", "Configuration reconciled"); err != nil {
		return ctrl.Result{}, err
	}
	return returnWithDriftDetection(r.Options, &cr.Spec.HarborSpecBase)
}

func (r *ConfigurationReconciler) buildDesiredSettings(ctx context.Context, cr *harborv1alpha1.Configuration) (map[string]any, error) {
	desired := map[string]any{}

	for key, setting := range cr.Spec.Settings {
		value, err := r.resolveConfigurationValue(ctx, cr.Namespace, key, setting)
		if err != nil {
			return nil, err
		}
		desired[key] = value
	}

	return desired, nil
}

func (r *ConfigurationReconciler) resolveConfigurationValue(
	ctx context.Context,
	namespace string,
	key string,
	setting harborv1alpha1.ConfigurationValue,
) (any, error) {
	if setting.Value != nil {
		var value any
		if err := json.Unmarshal(setting.Value.Raw, &value); err != nil {
			return nil, fmt.Errorf("invalid settings value for %q: %w", key, err)
		}
		return value, nil
	}

	if setting.ValueFrom == nil {
		return nil, fmt.Errorf("setting %q must define value or valueFrom", key)
	}

	secretValue, err := readSecretValue(ctx, r.Options, r.Client, setting.ValueFrom.SecretKeyRef, namespace, "value")
	if err != nil {
		return nil, fmt.Errorf("failed to read secret for %q: %w", key, err)
	}

	var value any
	if err := json.Unmarshal([]byte(secretValue), &value); err == nil {
		return value, nil
	}
	return secretValue, nil
}

func ensureEditableSettings(desired map[string]any, current map[string]harborclient.ConfigurationItem) error {
	for key := range desired {
		item, ok := current[key]
		if !ok {
			continue
		}
		if !item.Editable {
			return fmt.Errorf("configuration %q is not editable", key)
		}
	}
	return nil
}

func configurationNeedsUpdate(desired map[string]any, current map[string]harborclient.ConfigurationItem) bool {
	for key, desiredVal := range desired {
		item, ok := current[key]
		if !ok {
			return true
		}
		if !jsonValuesEqual(desiredVal, item.Value) {
			return true
		}
	}
	return false
}

func jsonValuesEqual(desired any, current json.RawMessage) bool {
	desiredJSON, err := json.Marshal(desired)
	if err != nil {
		return false
	}
	var desiredVal any
	if err := json.Unmarshal(desiredJSON, &desiredVal); err != nil {
		return false
	}
	var currentVal any
	if err := json.Unmarshal(current, &currentVal); err != nil {
		return false
	}
	return reflect.DeepEqual(desiredVal, currentVal)
}

func (r *ConfigurationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	builder, err := setupHarborBackedController(
		mgr,
		r.Options,
		&harborv1alpha1.Configuration{},
		func() client.ObjectList { return &harborv1alpha1.ConfigurationList{} },
		func(obj client.Object) *harborv1alpha1.HarborConnectionReference {
			return obj.(*harborv1alpha1.Configuration).Spec.HarborConnectionRef
		},
		"configuration",
	)
	if err != nil {
		return err
	}
	return builder.Complete(r)
}
