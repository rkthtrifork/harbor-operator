package controller

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
)

const (
	ConditionReady       = "Ready"
	ConditionReconciling = "Reconciling"
	ConditionStalled     = "Stalled"
)

func setCondition(conditions *[]metav1.Condition, cond metav1.Condition) bool {
	existing := meta.FindStatusCondition(*conditions, cond.Type)
	if existing != nil &&
		existing.Status == cond.Status &&
		existing.Reason == cond.Reason &&
		existing.Message == cond.Message &&
		existing.ObservedGeneration == cond.ObservedGeneration {
		return false
	}
	meta.SetStatusCondition(conditions, cond)
	return true
}

func markReconciling(base *harborv1alpha1.HarborStatusBase, generation int64, reason, message string) bool {
	var changed bool
	if base.ObservedGeneration != generation {
		base.ObservedGeneration = generation
		changed = true
	}
	if reason == "" {
		reason = "Reconciling"
	}
	if message == "" {
		message = "Reconciling resource"
	}
	changed = setCondition(&base.Conditions, metav1.Condition{
		Type:               ConditionReady,
		Status:             metav1.ConditionFalse,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: generation,
		LastTransitionTime: metav1.Now(),
	}) || changed
	changed = setCondition(&base.Conditions, metav1.Condition{
		Type:               ConditionReconciling,
		Status:             metav1.ConditionTrue,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: generation,
		LastTransitionTime: metav1.Now(),
	}) || changed
	changed = setCondition(&base.Conditions, metav1.Condition{
		Type:               ConditionStalled,
		Status:             metav1.ConditionFalse,
		Reason:             "NotStalled",
		Message:            "Resource is not stalled",
		ObservedGeneration: generation,
		LastTransitionTime: metav1.Now(),
	}) || changed
	return changed
}

func markReady(base *harborv1alpha1.HarborStatusBase, generation int64, reason, message string) bool {
	var changed bool
	if base.ObservedGeneration != generation {
		base.ObservedGeneration = generation
		changed = true
	}
	if reason == "" {
		reason = "Reconciled"
	}
	if message == "" {
		message = "Resource is ready"
	}
	changed = setCondition(&base.Conditions, metav1.Condition{
		Type:               ConditionReady,
		Status:             metav1.ConditionTrue,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: generation,
		LastTransitionTime: metav1.Now(),
	}) || changed
	changed = setCondition(&base.Conditions, metav1.Condition{
		Type:               ConditionReconciling,
		Status:             metav1.ConditionFalse,
		Reason:             "Reconciled",
		Message:            "Reconciliation complete",
		ObservedGeneration: generation,
		LastTransitionTime: metav1.Now(),
	}) || changed
	changed = setCondition(&base.Conditions, metav1.Condition{
		Type:               ConditionStalled,
		Status:             metav1.ConditionFalse,
		Reason:             "NotStalled",
		Message:            "Resource is not stalled",
		ObservedGeneration: generation,
		LastTransitionTime: metav1.Now(),
	}) || changed
	return changed
}

func markError(base *harborv1alpha1.HarborStatusBase, generation int64, err error) bool {
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	if msg == "" {
		msg = "Reconcile error"
	}
	return markReconciling(base, generation, "ReconcileError", fmt.Sprintf("Reconcile error: %s", msg))
}

func setReadyStatus(ctx context.Context, c client.Client, obj client.Object, base *harborv1alpha1.HarborStatusBase, generation int64, reason, message string) error {
	if changed := markReady(base, generation, reason, message); changed {
		sanitizeOptionalHarborConnectionRef(obj)
		return c.Status().Update(ctx, obj)
	}
	return nil
}

func setReconcilingStatus(ctx context.Context, c client.Client, obj client.Object, base *harborv1alpha1.HarborStatusBase, generation int64, reason, message string) error {
	if changed := markReconciling(base, generation, reason, message); changed {
		sanitizeOptionalHarborConnectionRef(obj)
		return c.Status().Update(ctx, obj)
	}
	return nil
}

func setErrorStatus(ctx context.Context, c client.Client, obj client.Object, base *harborv1alpha1.HarborStatusBase, generation int64, err error) error {
	if changed := markError(base, generation, err); changed {
		sanitizeOptionalHarborConnectionRef(obj)
		if updateErr := c.Status().Update(ctx, obj); updateErr != nil {
			return updateErr
		}
	}
	return err
}
