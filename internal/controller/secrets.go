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

func upsertSecretValue(ctx context.Context, c client.Client, namespace, name, key, value string) error {
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
		return c.Create(ctx, &secret)
	}
	if secret.Data == nil {
		secret.Data = map[string][]byte{}
	}
	secret.Data[key] = []byte(value)
	return c.Update(ctx, &secret)
}
