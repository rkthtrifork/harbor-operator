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

type singletonItemVisitor func(client.Object, *harborv1alpha1.HarborConnectionReference)
type visitSingletonItems func(singletonItemVisitor)

func normalizedHarborConnectionRef(ref *harborv1alpha1.HarborConnectionReference) harborv1alpha1.HarborConnectionReference {
	if ref == nil {
		return harborv1alpha1.HarborConnectionReference{}
	}
	out := *ref
	if out.Kind == "" {
		out.Kind = harborv1alpha1.HarborConnectionReferenceKindNamespaced
	}
	return out
}

func harborConnectionRefsEqual(a, b *harborv1alpha1.HarborConnectionReference) bool {
	normalizedA := normalizedHarborConnectionRef(a)
	normalizedB := normalizedHarborConnectionRef(b)
	return normalizedA.Name == normalizedB.Name && normalizedA.Kind == normalizedB.Kind
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

func ensureSingletonOwner(
	ctx context.Context,
	options OperatorOptions,
	c client.Client,
	current client.Object,
	currentRef *harborv1alpha1.HarborConnectionReference,
	list client.ObjectList,
	visitItems visitSingletonItems,
	singletonKind string,
) error {
	currentConn, err := resolveHarborConnection(ctx, options, c, current.GetNamespace(), currentRef)
	if err != nil {
		return err
	}
	if err := c.List(ctx, list); err != nil {
		return err
	}

	currentName := client.ObjectKeyFromObject(current)
	currentBaseURL := normalizeBaseURL(currentConn.baseURL)
	peers := []singletonCandidate{}
	visitItems(func(item client.Object, itemRef *harborv1alpha1.HarborConnectionReference) {
		if client.ObjectKeyFromObject(item) == currentName || !item.GetDeletionTimestamp().IsZero() {
			return
		}
		itemConn, err := resolveHarborConnection(ctx, options, c, item.GetNamespace(), itemRef)
		if err != nil || normalizeBaseURL(itemConn.baseURL) != currentBaseURL {
			return
		}
		peers = append(peers, singletonCandidate{
			name:      client.ObjectKeyFromObject(item),
			createdAt: item.GetCreationTimestamp(),
		})
	})
	return singletonOwnerConflict(current, currentConn.baseURL, peers, singletonKind)
}

func ensureConfigurationSingletonOwner(ctx context.Context, options OperatorOptions, c client.Client, current *harborv1alpha1.Configuration) error {
	list := &harborv1alpha1.ConfigurationList{}
	return ensureSingletonOwner(ctx, options, c, current, current.Spec.HarborConnectionRef, list, func(visit singletonItemVisitor) {
		for i := range list.Items {
			item := &list.Items[i]
			visit(item, item.Spec.HarborConnectionRef)
		}
	}, "Configuration")
}

func ensureGCScheduleSingletonOwner(ctx context.Context, options OperatorOptions, c client.Client, current *harborv1alpha1.GCSchedule) error {
	list := &harborv1alpha1.GCScheduleList{}
	return ensureSingletonOwner(ctx, options, c, current, current.Spec.HarborConnectionRef, list, func(visit singletonItemVisitor) {
		for i := range list.Items {
			item := &list.Items[i]
			visit(item, item.Spec.HarborConnectionRef)
		}
	}, "GCSchedule")
}

func ensurePurgeAuditScheduleSingletonOwner(ctx context.Context, options OperatorOptions, c client.Client, current *harborv1alpha1.PurgeAuditSchedule) error {
	list := &harborv1alpha1.PurgeAuditScheduleList{}
	return ensureSingletonOwner(ctx, options, c, current, current.Spec.HarborConnectionRef, list, func(visit singletonItemVisitor) {
		for i := range list.Items {
			item := &list.Items[i]
			visit(item, item.Spec.HarborConnectionRef)
		}
	}, "PurgeAuditSchedule")
}

func ensureScanAllScheduleSingletonOwner(ctx context.Context, options OperatorOptions, c client.Client, current *harborv1alpha1.ScanAllSchedule) error {
	list := &harborv1alpha1.ScanAllScheduleList{}
	return ensureSingletonOwner(ctx, options, c, current, current.Spec.HarborConnectionRef, list, func(visit singletonItemVisitor) {
		for i := range list.Items {
			item := &list.Items[i]
			visit(item, item.Spec.HarborConnectionRef)
		}
	}, "ScanAllSchedule")
}
