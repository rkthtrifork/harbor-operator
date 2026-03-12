package controller

import (
	"context"
	"testing"
	"time"

	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestFinalizeWithoutHarborConnection(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                 string
		deletionPolicy       harborv1alpha1.DeletionPolicy
		requiresRemoteDelete bool
		wantDone             bool
		wantFinalizer        bool
	}{
		{
			name:                 "removes finalizer for local-only resources",
			deletionPolicy:       harborv1alpha1.DeletionPolicyDelete,
			requiresRemoteDelete: false,
			wantDone:             true,
			wantFinalizer:        false,
		},
		{
			name:                 "keeps finalizer for remote cleanup resources with delete policy",
			deletionPolicy:       harborv1alpha1.DeletionPolicyDelete,
			requiresRemoteDelete: true,
			wantDone:             false,
			wantFinalizer:        true,
		},
		{
			name:                 "removes finalizer for remote cleanup resources with orphan policy",
			deletionPolicy:       harborv1alpha1.DeletionPolicyOrphan,
			requiresRemoteDelete: true,
			wantDone:             true,
			wantFinalizer:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			scheme := runtime.NewScheme()
			if err := harborv1alpha1.AddToScheme(scheme); err != nil {
				t.Fatalf("add scheme: %v", err)
			}

			now := metav1.NewTime(time.Now())
			obj := &harborv1alpha1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:              "test-project",
					Namespace:         "default",
					Finalizers:        []string{finalizerName},
					DeletionTimestamp: &now,
				},
			}

			c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(obj.DeepCopy()).Build()
			obj = &harborv1alpha1.Project{}
			if err := c.Get(context.Background(), types.NamespacedName{Name: "test-project", Namespace: "default"}, obj); err != nil {
				t.Fatalf("get current object: %v", err)
			}
			done, err := finalizeWithoutHarborConnection(
				context.Background(),
				c,
				obj,
				tt.deletionPolicy,
				tt.requiresRemoteDelete,
				apierrors.NewNotFound(schema.GroupResource{Group: harborv1alpha1.GroupVersion.Group, Resource: "harborconnections"}, "missing"),
			)
			if err != nil {
				t.Fatalf("finalizeWithoutHarborConnection returned error: %v", err)
			}
			if done != tt.wantDone {
				t.Fatalf("done = %v, want %v", done, tt.wantDone)
			}

			var stored harborv1alpha1.Project
			err = c.Get(context.Background(), types.NamespacedName{Name: "test-project", Namespace: "default"}, &stored)
			if !tt.wantFinalizer {
				if !apierrors.IsNotFound(err) {
					t.Fatalf("expected object to be deleted, got err=%v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("get stored object: %v", err)
			}
			hasFinalizer := false
			for _, finalizer := range stored.Finalizers {
				if finalizer == finalizerName {
					hasFinalizer = true
					break
				}
			}
			if hasFinalizer != tt.wantFinalizer {
				t.Fatalf("hasFinalizer = %v, want %v", hasFinalizer, tt.wantFinalizer)
			}
		})
	}
}
