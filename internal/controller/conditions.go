package controller

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Condition types following kstatus conventions.
// See: https://github.com/kubernetes-sigs/cli-utils/tree/master/pkg/kstatus
const (
	// TypeReady indicates the resource is ready and reconciled.
	TypeReady = "Ready"

	// TypeReconciling indicates the controller is actively working on the resource.
	TypeReconciling = "Reconciling"

	// TypeStalled indicates the resource is stuck in a failed state.
	TypeStalled = "Stalled"
)

// Condition reasons.
const (
	// ReasonReconcileSuccess indicates successful reconciliation.
	ReasonReconcileSuccess = "ReconcileSuccess"

	// ReasonReconcileError indicates a reconciliation error.
	ReasonReconcileError = "ReconcileError"

	// ReasonCreating indicates resource is being created.
	ReasonCreating = "Creating"

	// ReasonUpdating indicates resource is being updated.
	ReasonUpdating = "Updating"

	// ReasonAdopting indicates resource is being adopted.
	ReasonAdopting = "Adopting"

	// ReasonConnectionFailed indicates failed connection to Harbor.
	ReasonConnectionFailed = "ConnectionFailed"

	// ReasonInvalidSpec indicates the spec is invalid.
	ReasonInvalidSpec = "InvalidSpec"
)

// SetCondition adds or updates a condition in the conditions slice.
// If a condition with the same type already exists, it will be updated only if the status has changed.
func SetCondition(conditions *[]metav1.Condition, conditionType string, status metav1.ConditionStatus, reason, message string) {
	now := metav1.NewTime(time.Now())
	newCondition := metav1.Condition{
		Type:               conditionType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		LastTransitionTime: now,
	}

	for i, condition := range *conditions {
		if condition.Type == conditionType {
			// Only update if the status has changed
			if condition.Status != status {
				newCondition.LastTransitionTime = now
			} else {
				newCondition.LastTransitionTime = condition.LastTransitionTime
			}
			(*conditions)[i] = newCondition
			return
		}
	}

	// Condition doesn't exist, append it
	*conditions = append(*conditions, newCondition)
}

// GetCondition returns a condition by type from the conditions slice.
func GetCondition(conditions []metav1.Condition, conditionType string) *metav1.Condition {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return &condition
		}
	}
	return nil
}

// RemoveCondition removes a condition by type from the conditions slice.
func RemoveCondition(conditions *[]metav1.Condition, conditionType string) {
	newConditions := []metav1.Condition{}
	for _, condition := range *conditions {
		if condition.Type != conditionType {
			newConditions = append(newConditions, condition)
		}
	}
	*conditions = newConditions
}

// SetReadyCondition is a convenience function to set the Ready condition.
func SetReadyCondition(conditions *[]metav1.Condition, ready bool, reason, message string) {
	status := metav1.ConditionTrue
	if !ready {
		status = metav1.ConditionFalse
	}
	SetCondition(conditions, TypeReady, status, reason, message)
}

// SetReconcilingCondition is a convenience function to set the Reconciling condition.
func SetReconcilingCondition(conditions *[]metav1.Condition, reconciling bool, reason, message string) {
	status := metav1.ConditionTrue
	if !reconciling {
		status = metav1.ConditionFalse
	}
	SetCondition(conditions, TypeReconciling, status, reason, message)
}

// SetStalledCondition is a convenience function to set the Stalled condition.
func SetStalledCondition(conditions *[]metav1.Condition, stalled bool, reason, message string) {
	status := metav1.ConditionTrue
	if !stalled {
		status = metav1.ConditionFalse
	}
	SetCondition(conditions, TypeStalled, status, reason, message)
}
