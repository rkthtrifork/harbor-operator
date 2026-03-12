package controller

import (
	"context"
	"fmt"
	"sort"
	"strings"

	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type singletonCandidate struct {
	name      types.NamespacedName
	createdAt metav1.Time
}

func normalizedHarborConnectionRef(ref harborv1alpha1.HarborConnectionReference) harborv1alpha1.HarborConnectionReference {
	if ref.Kind == "" {
		ref.Kind = harborv1alpha1.HarborConnectionReferenceKindNamespaced
	}
	return ref
}

func harborConnectionRefsEqual(a, b harborv1alpha1.HarborConnectionReference) bool {
	a = normalizedHarborConnectionRef(a)
	b = normalizedHarborConnectionRef(b)
	return a.Name == b.Name && a.Kind == b.Kind
}

func normalizeBaseURL(baseURL string) string {
	return strings.TrimRight(baseURL, "/")
}

func singletonOwnerConflict(current client.Object, harborInstance string, peers []singletonCandidate, singletonKind string) error {
	if len(peers) == 0 {
		return nil
	}

	currentName := client.ObjectKeyFromObject(current)
	candidates := append(peers, singletonCandidate{
		name:      currentName,
		createdAt: current.GetCreationTimestamp(),
	})
	sort.Slice(candidates, func(i, j int) bool {
		if !candidates[i].createdAt.Equal(&candidates[j].createdAt) {
			return candidates[i].createdAt.Before(&candidates[j].createdAt)
		}
		if candidates[i].name.Namespace != candidates[j].name.Namespace {
			return candidates[i].name.Namespace < candidates[j].name.Namespace
		}
		return candidates[i].name.Name < candidates[j].name.Name
	})

	owner := candidates[0]
	if owner.name == currentName {
		return nil
	}

	return fmt.Errorf(
		"%s %s/%s conflicts with existing owner %s/%s for Harbor instance %q",
		singletonKind,
		current.GetNamespace(),
		current.GetName(),
		owner.name.Namespace,
		owner.name.Name,
		harborInstance,
	)
}

func ensureConfigurationSingletonOwner(ctx context.Context, c client.Client, current *harborv1alpha1.Configuration) error {
	currentConn, err := resolveHarborConnection(ctx, c, current.Namespace, current.Spec.HarborConnectionRef)
	if err != nil {
		return err
	}
	var list harborv1alpha1.ConfigurationList
	if err := c.List(ctx, &list); err != nil {
		return err
	}

	peers := make([]singletonCandidate, 0, len(list.Items))
	for i := range list.Items {
		item := &list.Items[i]
		if item.Name == current.Name && item.Namespace == current.Namespace {
			continue
		}
		if !item.DeletionTimestamp.IsZero() {
			continue
		}
		itemConn, err := resolveHarborConnection(ctx, c, item.Namespace, item.Spec.HarborConnectionRef)
		if err != nil || normalizeBaseURL(itemConn.baseURL) != normalizeBaseURL(currentConn.baseURL) {
			continue
		}
		peers = append(peers, singletonCandidate{
			name:      types.NamespacedName{Namespace: item.Namespace, Name: item.Name},
			createdAt: item.CreationTimestamp,
		})
	}
	return singletonOwnerConflict(current, currentConn.baseURL, peers, "Configuration")
}

func ensureGCScheduleSingletonOwner(ctx context.Context, c client.Client, current *harborv1alpha1.GCSchedule) error {
	currentConn, err := resolveHarborConnection(ctx, c, current.Namespace, current.Spec.HarborConnectionRef)
	if err != nil {
		return err
	}
	var list harborv1alpha1.GCScheduleList
	if err := c.List(ctx, &list); err != nil {
		return err
	}

	peers := make([]singletonCandidate, 0, len(list.Items))
	for i := range list.Items {
		item := &list.Items[i]
		if item.Name == current.Name && item.Namespace == current.Namespace {
			continue
		}
		if !item.DeletionTimestamp.IsZero() {
			continue
		}
		itemConn, err := resolveHarborConnection(ctx, c, item.Namespace, item.Spec.HarborConnectionRef)
		if err != nil || normalizeBaseURL(itemConn.baseURL) != normalizeBaseURL(currentConn.baseURL) {
			continue
		}
		peers = append(peers, singletonCandidate{
			name:      types.NamespacedName{Namespace: item.Namespace, Name: item.Name},
			createdAt: item.CreationTimestamp,
		})
	}
	return singletonOwnerConflict(current, currentConn.baseURL, peers, "GCSchedule")
}

func ensurePurgeAuditScheduleSingletonOwner(ctx context.Context, c client.Client, current *harborv1alpha1.PurgeAuditSchedule) error {
	currentConn, err := resolveHarborConnection(ctx, c, current.Namespace, current.Spec.HarborConnectionRef)
	if err != nil {
		return err
	}
	var list harborv1alpha1.PurgeAuditScheduleList
	if err := c.List(ctx, &list); err != nil {
		return err
	}

	peers := make([]singletonCandidate, 0, len(list.Items))
	for i := range list.Items {
		item := &list.Items[i]
		if item.Name == current.Name && item.Namespace == current.Namespace {
			continue
		}
		if !item.DeletionTimestamp.IsZero() {
			continue
		}
		itemConn, err := resolveHarborConnection(ctx, c, item.Namespace, item.Spec.HarborConnectionRef)
		if err != nil || normalizeBaseURL(itemConn.baseURL) != normalizeBaseURL(currentConn.baseURL) {
			continue
		}
		peers = append(peers, singletonCandidate{
			name:      types.NamespacedName{Namespace: item.Namespace, Name: item.Name},
			createdAt: item.CreationTimestamp,
		})
	}
	return singletonOwnerConflict(current, currentConn.baseURL, peers, "PurgeAuditSchedule")
}

func ensureScanAllScheduleSingletonOwner(ctx context.Context, c client.Client, current *harborv1alpha1.ScanAllSchedule) error {
	currentConn, err := resolveHarborConnection(ctx, c, current.Namespace, current.Spec.HarborConnectionRef)
	if err != nil {
		return err
	}
	var list harborv1alpha1.ScanAllScheduleList
	if err := c.List(ctx, &list); err != nil {
		return err
	}

	peers := make([]singletonCandidate, 0, len(list.Items))
	for i := range list.Items {
		item := &list.Items[i]
		if item.Name == current.Name && item.Namespace == current.Namespace {
			continue
		}
		if !item.DeletionTimestamp.IsZero() {
			continue
		}
		itemConn, err := resolveHarborConnection(ctx, c, item.Namespace, item.Spec.HarborConnectionRef)
		if err != nil || normalizeBaseURL(itemConn.baseURL) != normalizeBaseURL(currentConn.baseURL) {
			continue
		}
		peers = append(peers, singletonCandidate{
			name:      types.NamespacedName{Namespace: item.Namespace, Name: item.Name},
			createdAt: item.CreationTimestamp,
		})
	}
	return singletonOwnerConflict(current, currentConn.baseURL, peers, "ScanAllSchedule")
}
