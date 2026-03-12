package controller

import (
	context "context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
)

const (
	testNamespace = "default"
	testAdminUser = adminName
	testPassword  = "password"
)

func createPasswordSecret(ctx context.Context, c client.Client, name, value string) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      name,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			testPassword: []byte(value),
		},
	}
	return c.Create(ctx, secret)
}

func createHarborConnection(ctx context.Context, c client.Client, name, baseURL, secretName string) error {
	conn := &harborv1alpha1.HarborConnection{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: testNamespace,
			Name:      name,
		},
		Spec: harborv1alpha1.HarborConnectionSpec{
			BaseURL: baseURL,
			Credentials: &harborv1alpha1.Credentials{
				Type:     "basic",
				Username: testAdminUser,
				PasswordSecretRef: harborv1alpha1.SecretReference{
					Name: secretName,
					Key:  testPassword,
				},
			},
		},
	}
	return c.Create(ctx, conn)
}
