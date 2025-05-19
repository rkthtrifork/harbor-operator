// Copyright 2025 The Harbor-Operator Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// -----------------------------------------------------------------------------
// kstatus-compatible Condition helpers
// -----------------------------------------------------------------------------

// Standard condition types adopted from sigs.k8s.io/kustomize/kstatus.
//
// All of them follow the **abnormal-true** polarity pattern:
//
//   - Reconciling=True – the controller is actively reconciling the resource.
//   - Stalled=True     – reconciliation hit an error or made insufficient progress.
//   - Ready=True       – (fallback) the resource is fully reconciled.
//
// A well-behaved reconciler should ensure:
//
//  1. `Reconciling=True` while it works, then flip to False (or remove) on success
//     *before* setting Ready=True.
//  2. `Stalled=True` *and* `Reconciling=False` when it surfaces an error.
//  3. `Ready=True` only when the observedGeneration equals metadata.generation
//     **and** the resource is in its desired state.
const (
	ConditionReconciling string = "Reconciling"
	ConditionStalled     string = "Stalled"
	ConditionReady       string = "Ready"
)

// SetStatusCondition inserts or updates the given condition in the slice.
// `LastTransitionTime` is automatically refreshed when the condition’s Status
// field changes.
//
// Usage:
//
//	harborv1alpha1.SetStatusCondition(&cr.Status.Conditions, metav1.Condition{
//	    Type:   harborv1alpha1.ConditionReconciling,
//	    Status: metav1.ConditionTrue,
//	    Reason: "DryRun",
//	    Message: "Reconciling resource in dry-run mode",
//	})
func SetStatusCondition(conditions *[]metav1.Condition, cond metav1.Condition) {
	if conditions == nil {
		return
	}
	for i := range *conditions {
		c := (*conditions)[i]
		if c.Type != cond.Type {
			continue
		}

		// Only touch LastTransitionTime when .Status actually flips.
		if c.Status != cond.Status {
			cond.LastTransitionTime = metav1.Now()
		} else {
			cond.LastTransitionTime = c.LastTransitionTime
		}
		(*conditions)[i] = cond
		return
	}

	// Not found – add new
	if cond.LastTransitionTime.IsZero() {
		cond.LastTransitionTime = metav1.Now()
	}
	*conditions = append(*conditions, cond)
}

// RemoveCondition deletes all conditions of the specified type from the slice.
func RemoveCondition(conditions *[]metav1.Condition, condType string) {
	if conditions == nil || len(*conditions) == 0 {
		return
	}
	filtered := (*conditions)[:0]
	for _, c := range *conditions {
		if c.Type != condType {
			filtered = append(filtered, c)
		}
	}
	*conditions = filtered
}

// IsConditionTrue returns true if a condition of the given type exists and its
// Status is metav1.ConditionTrue.
func IsConditionTrue(conditions []metav1.Condition, condType string) bool {
	for _, c := range conditions {
		if c.Type == condType {
			return c.Status == metav1.ConditionTrue
		}
	}
	return false
}
