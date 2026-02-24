package controller

import (
	context "context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
)

func createPasswordSecret(ctx context.Context, c client.Client, namespace, name, key, value string) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			key: []byte(value),
		},
	}
	return c.Create(ctx, secret)
}

func createHarborConnection(ctx context.Context, c client.Client, namespace, name, baseURL, username, secretName, secretKey string) error {
	conn := &harborv1alpha1.HarborConnection{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
		Spec: harborv1alpha1.HarborConnectionSpec{
			BaseURL: baseURL,
			Credentials: &harborv1alpha1.Credentials{
				Type:     "basic",
				Username: username,
				PasswordSecretRef: corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
					Key:                  secretKey,
				},
			},
		},
	}
	return c.Create(ctx, conn)
}
