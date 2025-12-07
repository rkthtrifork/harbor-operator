package controller

import (
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSetCondition(t *testing.T) {
	conditions := []metav1.Condition{}

	// Test adding a new condition
	SetCondition(&conditions, TypeReady, metav1.ConditionTrue, ReasonReconcileSuccess, "Test message")

	if len(conditions) != 1 {
		t.Errorf("Expected 1 condition, got %d", len(conditions))
	}

	if conditions[0].Type != TypeReady {
		t.Errorf("Expected condition type %s, got %s", TypeReady, conditions[0].Type)
	}

	if conditions[0].Status != metav1.ConditionTrue {
		t.Errorf("Expected status True, got %s", conditions[0].Status)
	}

	if conditions[0].Reason != ReasonReconcileSuccess {
		t.Errorf("Expected reason %s, got %s", ReasonReconcileSuccess, conditions[0].Reason)
	}

	// Test updating an existing condition with same status (should not change timestamp)
	firstTimestamp := conditions[0].LastTransitionTime
	time.Sleep(10 * time.Millisecond) // Ensure time passes
	SetCondition(&conditions, TypeReady, metav1.ConditionTrue, ReasonReconcileSuccess, "Updated message")

	if len(conditions) != 1 {
		t.Errorf("Expected 1 condition after update, got %d", len(conditions))
	}

	if !conditions[0].LastTransitionTime.Equal(&firstTimestamp) {
		t.Errorf("Expected timestamp to remain the same when status doesn't change")
	}

	// Test updating with different status (should change timestamp)
	time.Sleep(10 * time.Millisecond)
	SetCondition(&conditions, TypeReady, metav1.ConditionFalse, ReasonReconcileError, "Error message")

	if len(conditions) != 1 {
		t.Errorf("Expected 1 condition after status change, got %d", len(conditions))
	}

	if conditions[0].Status != metav1.ConditionFalse {
		t.Errorf("Expected status False, got %s", conditions[0].Status)
	}

	if conditions[0].LastTransitionTime.Equal(&firstTimestamp) {
		t.Errorf("Expected timestamp to change when status changes")
	}

	// Test adding a second condition
	SetCondition(&conditions, TypeReconciling, metav1.ConditionTrue, ReasonCreating, "Creating resource")

	if len(conditions) != 2 {
		t.Errorf("Expected 2 conditions, got %d", len(conditions))
	}
}

func TestGetCondition(t *testing.T) {
	conditions := []metav1.Condition{
		{
			Type:               TypeReady,
			Status:             metav1.ConditionTrue,
			Reason:             ReasonReconcileSuccess,
			Message:            "Test",
			LastTransitionTime: metav1.Now(),
		},
	}

	// Test getting existing condition
	cond := GetCondition(conditions, TypeReady)
	if cond == nil {
		t.Error("Expected to find condition, got nil")
	}
	if cond.Type != TypeReady {
		t.Errorf("Expected condition type %s, got %s", TypeReady, cond.Type)
	}

	// Test getting non-existent condition
	cond = GetCondition(conditions, TypeStalled)
	if cond != nil {
		t.Error("Expected nil for non-existent condition")
	}
}

func TestRemoveCondition(t *testing.T) {
	conditions := []metav1.Condition{
		{
			Type:               TypeReady,
			Status:             metav1.ConditionTrue,
			Reason:             ReasonReconcileSuccess,
			LastTransitionTime: metav1.Now(),
		},
		{
			Type:               TypeReconciling,
			Status:             metav1.ConditionTrue,
			Reason:             ReasonCreating,
			LastTransitionTime: metav1.Now(),
		},
	}

	// Test removing existing condition
	RemoveCondition(&conditions, TypeReady)
	if len(conditions) != 1 {
		t.Errorf("Expected 1 condition after removal, got %d", len(conditions))
	}
	if conditions[0].Type != TypeReconciling {
		t.Errorf("Expected remaining condition to be %s, got %s", TypeReconciling, conditions[0].Type)
	}

	// Test removing non-existent condition (should be no-op)
	RemoveCondition(&conditions, TypeStalled)
	if len(conditions) != 1 {
		t.Errorf("Expected 1 condition after no-op removal, got %d", len(conditions))
	}
}

func TestSetReadyCondition(t *testing.T) {
	conditions := []metav1.Condition{}

	// Test setting ready to true
	SetReadyCondition(&conditions, true, ReasonReconcileSuccess, "Ready")
	if len(conditions) != 1 {
		t.Fatalf("Expected 1 condition, got %d", len(conditions))
	}
	if conditions[0].Status != metav1.ConditionTrue {
		t.Errorf("Expected True status, got %s", conditions[0].Status)
	}

	// Test setting ready to false
	SetReadyCondition(&conditions, false, ReasonReconcileError, "Not ready")
	if conditions[0].Status != metav1.ConditionFalse {
		t.Errorf("Expected False status, got %s", conditions[0].Status)
	}
}

func TestSetReconcilingCondition(t *testing.T) {
	conditions := []metav1.Condition{}

	// Test setting reconciling to true
	SetReconcilingCondition(&conditions, true, ReasonCreating, "Creating")
	if len(conditions) != 1 {
		t.Fatalf("Expected 1 condition, got %d", len(conditions))
	}
	if conditions[0].Status != metav1.ConditionTrue {
		t.Errorf("Expected True status, got %s", conditions[0].Status)
	}
	if conditions[0].Type != TypeReconciling {
		t.Errorf("Expected type %s, got %s", TypeReconciling, conditions[0].Type)
	}
}

func TestSetStalledCondition(t *testing.T) {
	conditions := []metav1.Condition{}

	// Test setting stalled to true
	SetStalledCondition(&conditions, true, ReasonConnectionFailed, "Connection failed")
	if len(conditions) != 1 {
		t.Fatalf("Expected 1 condition, got %d", len(conditions))
	}
	if conditions[0].Status != metav1.ConditionTrue {
		t.Errorf("Expected True status, got %s", conditions[0].Status)
	}
	if conditions[0].Type != TypeStalled {
		t.Errorf("Expected type %s, got %s", TypeStalled, conditions[0].Type)
	}

	// Test setting stalled to false
	SetStalledCondition(&conditions, false, ReasonReconcileSuccess, "")
	if conditions[0].Status != metav1.ConditionFalse {
		t.Errorf("Expected False status, got %s", conditions[0].Status)
	}
}
