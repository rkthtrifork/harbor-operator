package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
	"github.com/rkthtrifork/harbor-operator/internal/harborclient"
)

type ConfigurationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	logger logr.Logger
}

// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=configurations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=configurations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections,verbs=get;list;watch

func (r *ConfigurationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.logger = log.FromContext(ctx).WithName(fmt.Sprintf("[Configuration:%s]", req.NamespacedName))

	var cr harborv1alpha1.Configuration
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

	if !cr.DeletionTimestamp.IsZero() {
		_ = removeFinalizer(ctx, r.Client, &cr)
		return ctrl.Result{}, nil
	}

	_ = ensureFinalizer(ctx, r.Client, &cr)

	desired, err := r.buildDesiredSettings(ctx, &cr)
	if err != nil {
		return ctrl.Result{}, setErrorStatus(ctx, r.Client, &cr, &cr.Status.HarborStatusBase, cr.Generation, err)
	}
	if len(desired) == 0 {
		r.logger.V(1).Info("No configuration settings specified; nothing to apply")
		if changed := markReady(&cr.Status.HarborStatusBase, cr.Generation, "Noop", "No configuration changes to apply"); changed {
			if err := r.Status().Update(ctx, &cr); err != nil {
				return ctrl.Result{}, err
			}
		}
		return returnWithDriftDetection(&cr.Spec.HarborSpecBase)
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

	if changed := markReady(&cr.Status.HarborStatusBase, cr.Generation, "Reconciled", "Configuration reconciled"); changed {
		if err := r.Status().Update(ctx, &cr); err != nil {
			return ctrl.Result{}, err
		}
	}
	return returnWithDriftDetection(&cr.Spec.HarborSpecBase)
}

func (r *ConfigurationReconciler) buildDesiredSettings(ctx context.Context, cr *harborv1alpha1.Configuration) (map[string]any, error) {
	desired := map[string]any{}

	for key, raw := range cr.Spec.Settings {
		if len(raw.Raw) == 0 {
			continue
		}
		var val any
		if err := json.Unmarshal(raw.Raw, &val); err != nil {
			return nil, fmt.Errorf("invalid settings value for %q: %w", key, err)
		}
		desired[key] = val
	}

	for key, ref := range cr.Spec.SecretSettings {
		secretValue, err := readSecretValue(ctx, r.Client, ref, cr.Namespace, "value")
		if err != nil {
			return nil, fmt.Errorf("failed to read secret for %q: %w", key, err)
		}
		var parsed any
		if err := json.Unmarshal([]byte(secretValue), &parsed); err == nil {
			desired[key] = parsed
		} else {
			desired[key] = secretValue
		}
	}

	return desired, nil
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
	return ctrl.NewControllerManagedBy(mgr).
		For(&harborv1alpha1.Configuration{}).
		Named("configuration").
		Complete(r)
}
