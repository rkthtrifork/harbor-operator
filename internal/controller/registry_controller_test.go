/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
...
*/

package controller

import (
	"context"
	"net/http"
	"net/http/httptest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
)

var _ = Describe("Registry Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		registry := &harborv1alpha1.Registry{}
		var server *httptest.Server

		BeforeEach(func() {
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				user, pass, ok := r.BasicAuth()
				if !ok || user != testAdminUser || pass != testPassword {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				if r.Method == http.MethodPost && r.URL.Path == "/api/v2.0/registries" {
					w.Header().Set("Location", "/api/v2.0/registries/7")
					w.WriteHeader(http.StatusCreated)
					return
				}
				http.NotFound(w, r)
			}))

			Expect(createPasswordSecret(ctx, k8sClient, "harbor-admin", testPassword)).To(Succeed())
			Expect(createHarborConnection(ctx, k8sClient, "harbor-conn", server.URL, "harbor-admin")).To(Succeed())

			By("creating the custom resource for the Kind Registry")
			resource := &harborv1alpha1.Registry{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: harborv1alpha1.RegistrySpec{
					HarborSpecBase: harborv1alpha1.HarborSpecBase{
						HarborConnectionRef: harborv1alpha1.HarborConnectionReference{Name: "harbor-conn"},
					},
					Name:        resourceName,
					URL:         "https://registry.example.com",
					Description: "test registry",
					Type:        "docker-hub",
					Insecure:    false,
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())
		})

		AfterEach(func() {
			server.Close()
			resource := &harborv1alpha1.Registry{}
			_ = k8sClient.Get(ctx, typeNamespacedName, resource)

			By("Cleanup the specific resource instance Registry")
			_ = k8sClient.Delete(ctx, resource)

			conn := &harborv1alpha1.HarborConnection{}
			_ = k8sClient.Get(ctx, types.NamespacedName{Name: "harbor-conn", Namespace: "default"}, conn)
			_ = k8sClient.Delete(ctx, conn)
			secret := &corev1.Secret{}
			_ = k8sClient.Get(ctx, types.NamespacedName{Name: "harbor-admin", Namespace: "default"}, secret)
			_ = k8sClient.Delete(ctx, secret)
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &RegistryReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(k8sClient.Get(ctx, typeNamespacedName, registry)).To(Succeed())
			Expect(registry.Status.HarborRegistryID).To(Equal(7))
		})
	})
})
