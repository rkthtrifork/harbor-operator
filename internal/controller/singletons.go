package controller

import (
	"context"
	"fmt"
	"sort"
	"strings"

	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type singletonCandidate struct {
	name      types.NamespacedName
	createdAt metav1.Time
}

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
	list client.ObjectList,
	getRef harborConnectionRefAccessor,
	singletonKind string,
) error {
	currentConn, err := resolveHarborConnection(ctx, options, c, current.GetNamespace(), getRef(current))
	if err != nil {
		return err
	}
	if err := c.List(ctx, list); err != nil {
		return err
	}

	currentName := client.ObjectKeyFromObject(current)
	currentBaseURL := normalizeBaseURL(currentConn.baseURL)
	peers := []singletonCandidate{}
	if err := apimeta.EachListItem(list, func(raw runtime.Object) error {
		item, ok := raw.(client.Object)
		if !ok {
			return nil
		}
		if client.ObjectKeyFromObject(item) == currentName || !item.GetDeletionTimestamp().IsZero() {
			return nil
		}
		itemConn, err := resolveHarborConnection(ctx, options, c, item.GetNamespace(), getRef(item))
		if err != nil || normalizeBaseURL(itemConn.baseURL) != currentBaseURL {
			return nil
		}
		peers = append(peers, singletonCandidate{
			name:      client.ObjectKeyFromObject(item),
			createdAt: item.GetCreationTimestamp(),
		})
		return nil
	}); err != nil {
		return err
	}
	return singletonOwnerConflict(current, currentConn.baseURL, peers, singletonKind)
}

func ensureConfigurationSingletonOwner(ctx context.Context, options OperatorOptions, c client.Client, current *harborv1alpha1.Configuration) error {
	return ensureSingletonOwner(ctx, options, c, current, &harborv1alpha1.ConfigurationList{}, func(obj client.Object) *harborv1alpha1.HarborConnectionReference {
		return obj.(*harborv1alpha1.Configuration).Spec.HarborConnectionRef
	}, "Configuration")
}

func ensureGCScheduleSingletonOwner(ctx context.Context, options OperatorOptions, c client.Client, current *harborv1alpha1.GCSchedule) error {
	return ensureSingletonOwner(ctx, options, c, current, &harborv1alpha1.GCScheduleList{}, func(obj client.Object) *harborv1alpha1.HarborConnectionReference {
		return obj.(*harborv1alpha1.GCSchedule).Spec.HarborConnectionRef
	}, "GCSchedule")
}

func ensurePurgeAuditScheduleSingletonOwner(ctx context.Context, options OperatorOptions, c client.Client, current *harborv1alpha1.PurgeAuditSchedule) error {
	return ensureSingletonOwner(ctx, options, c, current, &harborv1alpha1.PurgeAuditScheduleList{}, func(obj client.Object) *harborv1alpha1.HarborConnectionReference {
		return obj.(*harborv1alpha1.PurgeAuditSchedule).Spec.HarborConnectionRef
	}, "PurgeAuditSchedule")
}

func ensureScanAllScheduleSingletonOwner(ctx context.Context, options OperatorOptions, c client.Client, current *harborv1alpha1.ScanAllSchedule) error {
	return ensureSingletonOwner(ctx, options, c, current, &harborv1alpha1.ScanAllScheduleList{}, func(obj client.Object) *harborv1alpha1.HarborConnectionReference {
		return obj.(*harborv1alpha1.ScanAllSchedule).Spec.HarborConnectionRef
	}, "ScanAllSchedule")
}
