package controller

import (
	"testing"
	"time"

	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestHarborConnectionRefsEqualDefaultsKind(t *testing.T) {
	a := &harborv1alpha1.HarborConnectionReference{Name: "conn"}
	b := &harborv1alpha1.HarborConnectionReference{
		Name: "conn",
		Kind: harborv1alpha1.HarborConnectionReferenceKindNamespaced,
	}

	if !harborConnectionRefsEqual(a, b) {
		t.Fatalf("expected refs to match when kind defaults to HarborConnection")
	}
}

func TestSingletonOwnerConflictChoosesOldestCandidate(t *testing.T) {
	now := time.Now()
	current := &harborv1alpha1.Configuration{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "config-b",
			Namespace:         "team-a",
			CreationTimestamp: metav1.NewTime(now),
		},
	}

	err := singletonOwnerConflict(current, "http://harbor.example", []singletonCandidate{
		{
			name:      types.NamespacedName{Namespace: "team-a", Name: "config-a"},
			createdAt: metav1.NewTime(now.Add(-time.Minute)),
		},
	}, "Configuration")
	if err == nil {
		t.Fatalf("expected conflict when an older singleton exists")
	}
}

func TestSingletonOwnerConflictAllowsCurrentOwner(t *testing.T) {
	now := time.Now()
	current := &harborv1alpha1.Configuration{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "config-a",
			Namespace:         "team-a",
			CreationTimestamp: metav1.NewTime(now.Add(-time.Minute)),
		},
	}

	err := singletonOwnerConflict(current, "http://harbor.example", []singletonCandidate{
		{
			name:      types.NamespacedName{Namespace: "team-a", Name: "config-b"},
			createdAt: metav1.NewTime(now),
		},
	}, "Configuration")
	if err != nil {
		t.Fatalf("expected current object to remain owner, got %v", err)
	}
}
