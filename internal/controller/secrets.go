package controller

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
)

const (
	managedByLabelKey      = "harbor.harbor-operator.io/managed-by"
	managedByLabelValue    = "harbor-operator"
	ownerKindAnnotationKey = "harbor.harbor-operator.io/owner-kind"
	ownerNameAnnotationKey = "harbor.harbor-operator.io/owner-name"
	ownerNamespaceAnnKey   = "harbor.harbor-operator.io/owner-namespace"
)

func readSecretValue(ctx context.Context, c client.Client, ref harborv1alpha1.SecretReference, defaultNamespace, defaultKey string) (string, error) {
	namespace := ref.Namespace
	if namespace == "" {
		namespace = defaultNamespace
	}
	key := ref.Key
	if key == "" {
		key = defaultKey
	}
	if key == "" {
		return "", fmt.Errorf("secret key must be set for %s/%s", namespace, ref.Name)
	}

	var secret corev1.Secret
	if err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: ref.Name}, &secret); err != nil {
		return "", err
	}
	value, ok := secret.Data[key]
	if !ok {
		return "", fmt.Errorf("key %s not found in secret %s/%s", key, namespace, ref.Name)
	}
	return string(value), nil
}

func hashSecret(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func upsertOwnedSecretValue(ctx context.Context, c client.Client, owner client.Object, ownerKind, namespace, name, key, value string) error {
	if key == "" {
		return fmt.Errorf("secret key must be set for %s/%s", namespace, name)
	}
	var secret corev1.Secret
	err := c.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &secret)
	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		secret = corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      name,
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				key: []byte(value),
			},
		}
		ensureSecretOwnershipMetadata(&secret, owner, ownerKind)
		return c.Create(ctx, &secret)
	}
	if !secretOwnedBy(&secret, owner, ownerKind) {
		return fmt.Errorf("secret %s/%s already exists and is not managed by %s %s/%s", namespace, name, ownerKind, owner.GetNamespace(), owner.GetName())
	}

	changed := ensureSecretOwnershipMetadata(&secret, owner, ownerKind)
	if secret.Data == nil {
		secret.Data = map[string][]byte{}
		changed = true
	}
	if existing, ok := secret.Data[key]; !ok || string(existing) != value {
		secret.Data[key] = []byte(value)
		changed = true
	}
	if !changed {
		return nil
	}
	return c.Update(ctx, &secret)
}

func secretOwnedBy(secret *corev1.Secret, owner client.Object, ownerKind string) bool {
	return secret.Labels[managedByLabelKey] == managedByLabelValue &&
		secret.Annotations[ownerKindAnnotationKey] == ownerKind &&
		secret.Annotations[ownerNamespaceAnnKey] == owner.GetNamespace() &&
		secret.Annotations[ownerNameAnnotationKey] == owner.GetName()
}

func ensureSecretOwnershipMetadata(secret *corev1.Secret, owner client.Object, ownerKind string) bool {
	changed := false
	if secret.Labels == nil {
		secret.Labels = map[string]string{}
	}
	if secret.Annotations == nil {
		secret.Annotations = map[string]string{}
	}
	if secret.Labels[managedByLabelKey] != managedByLabelValue {
		secret.Labels[managedByLabelKey] = managedByLabelValue
		changed = true
	}
	if secret.Annotations[ownerKindAnnotationKey] != ownerKind {
		secret.Annotations[ownerKindAnnotationKey] = ownerKind
		changed = true
	}
	if secret.Annotations[ownerNamespaceAnnKey] != owner.GetNamespace() {
		secret.Annotations[ownerNamespaceAnnKey] = owner.GetNamespace()
		changed = true
	}
	if secret.Annotations[ownerNameAnnotationKey] != owner.GetName() {
		secret.Annotations[ownerNameAnnotationKey] = owner.GetName()
		changed = true
	}
	return changed
}
