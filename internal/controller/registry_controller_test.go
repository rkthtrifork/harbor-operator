/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
...
*/

package controller

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
)

var _ = Describe("Registry Controller", func() {
	const resourceName = "test-registry"
	ctx := context.Background()
	nsName := types.NamespacedName{
		Name:      resourceName,
		Namespace: "default",
	}

	var (
		testServer  *httptest.Server
		postCount   int32
		putCount    int32
		deleteCount int32
		// existingRegistry simulates a registry already in Harbor.
		existingRegistry *harborRegistryResponse
	)

	BeforeEach(func() {
		// Reset counters and state.
		atomic.StoreInt32(&postCount, 0)
		atomic.StoreInt32(&putCount, 0)
		atomic.StoreInt32(&deleteCount, 0)
		existingRegistry = nil

		// Start a fake Harbor server.
		testServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				// For GET /api/v2.0/registries, return the existing registry if set.
				if r.URL.Path == "/api/v2.0/registries" {
					w.Header().Set("Content-Type", "application/json")
					if existingRegistry != nil {
						json.NewEncoder(w).Encode([]harborRegistryResponse{*existingRegistry})
					} else {
						w.Write([]byte("[]"))
					}
					return
				}
			case http.MethodPost:
				// For POST /api/v2.0/registries, simulate creation.
				if r.URL.Path == "/api/v2.0/registries" {
					atomic.AddInt32(&postCount, 1)
					created := harborRegistryResponse{
						ID:          1,
						Name:        "test-registry",
						URL:         "http://example.com",
						Description: "Test Description",
						Type:        "github-ghcr",
						Insecure:    false,
					}
					existingRegistry = &created
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusCreated)
					json.NewEncoder(w).Encode(created)
					return
				}
			case http.MethodPut:
				// For PUT /api/v2.0/registries/{id}, simulate an update.
				if strings.HasPrefix(r.URL.Path, "/api/v2.0/registries/") {
					atomic.AddInt32(&putCount, 1)
					var payload createRegistryRequest
					err := json.NewDecoder(r.Body).Decode(&payload)
					Expect(err).NotTo(HaveOccurred())
					if existingRegistry != nil {
						existingRegistry.URL = payload.URL
						existingRegistry.Name = payload.Name
						existingRegistry.Description = payload.Description
						existingRegistry.Type = payload.Type
						existingRegistry.Insecure = payload.Insecure
					}
					w.WriteHeader(http.StatusOK)
					return
				}
			case http.MethodDelete:
				// For DELETE /api/v2.0/registries/{id}, simulate deletion.
				if strings.HasPrefix(r.URL.Path, "/api/v2.0/registries/") {
					atomic.AddInt32(&deleteCount, 1)
					existingRegistry = nil
					w.WriteHeader(http.StatusOK)
					return
				}
			}
			// Default to 404.
			w.WriteHeader(http.StatusNotFound)
		}))
	})

	AfterEach(func() {
		testServer.Close()

		// Clean up the Registry CR.
		reg := &harborv1alpha1.Registry{}
		err := k8sClient.Get(ctx, nsName, reg)
		if err == nil {
			Expect(k8sClient.Delete(ctx, reg)).To(Succeed())
		}
		// Optionally, clean up HarborConnection and Secret resources.
	})

	It("should create a registry in Harbor when it does not exist", func() {
		// Create a HarborConnection CR that points to our fake server.
		harborConn := &harborv1alpha1.HarborConnection{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-harbor",
				Namespace: "default",
			},
			Spec: harborv1alpha1.HarborConnectionSpec{
				BaseURL: testServer.URL,
				Credentials: &harborv1alpha1.Credentials{
					Type:            "basic",
					AccessKey:       "test-user",
					AccessSecretRef: "test-secret",
				},
			},
		}
		Expect(k8sClient.Create(ctx, harborConn)).To(Succeed())

		// Create a Secret for the HarborConnection.
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-secret",
				Namespace: "default",
			},
			Data: map[string][]byte{
				"access_secret": []byte("test-password"),
			},
		}
		Expect(k8sClient.Create(ctx, secret)).To(Succeed())

		// Create a Registry CR.
		registry := &harborv1alpha1.Registry{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resourceName,
				Namespace: "default",
			},
			Spec: harborv1alpha1.RegistrySpec{
				HarborSpecBase: harborv1alpha1.HarborSpecBase{
					HarborConnectionRef: "test-harbor",
				},
				Type:        "github-ghcr",
				Name:        "test-registry",
				Description: "Test Description",
				URL:         "http://example.com",
				Insecure:    false,
			},
		}
		Expect(k8sClient.Create(ctx, registry)).To(Succeed())

		// Reconcile the Registry.
		reconciler := &RegistryReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}
		_, err := reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: nsName})
		Expect(err).NotTo(HaveOccurred())

		// Verify that a POST request was made.
		Expect(atomic.LoadInt32(&postCount)).To(Equal(int32(1)))
	})

	It("should update a registry in Harbor when the CR is updated", func() {
		// Pre-populate existing registry in the fake Harbor.
		existingRegistry = &harborRegistryResponse{
			ID:          1,
			Name:        "test-registry",
			URL:         "http://example.com",
			Description: "Old Description",
			Type:        "github-ghcr",
			Insecure:    false,
		}

		// Ensure HarborConnection and Secret exist.
		harborConn := &harborv1alpha1.HarborConnection{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-harbor",
				Namespace: "default",
			},
			Spec: harborv1alpha1.HarborConnectionSpec{
				BaseURL: testServer.URL,
				Credentials: &harborv1alpha1.Credentials{
					Type:            "basic",
					AccessKey:       "test-user",
					AccessSecretRef: "test-secret",
				},
			},
		}
		err := k8sClient.Get(ctx, types.NamespacedName{Name: "test-harbor", Namespace: "default"}, harborConn)
		if err != nil && errors.IsNotFound(err) {
			Expect(k8sClient.Create(ctx, harborConn)).To(Succeed())
		}
		secret := &corev1.Secret{}
		err = k8sClient.Get(ctx, types.NamespacedName{Name: "test-secret", Namespace: "default"}, secret)
		if err != nil && errors.IsNotFound(err) {
			secret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"access_secret": []byte("test-password"),
				},
			}
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())
		}

		// Create or update the Registry CR with updated desired state.
		registry := &harborv1alpha1.Registry{}
		err = k8sClient.Get(ctx, nsName, registry)
		if err != nil && errors.IsNotFound(err) {
			registry = &harborv1alpha1.Registry{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
			}
			Expect(k8sClient.Create(ctx, registry)).To(Succeed())
		}
		registry.Spec = harborv1alpha1.RegistrySpec{
			HarborSpecBase: harborv1alpha1.HarborSpecBase{
				HarborConnectionRef: "test-harbor",
			},
			Type:        "github-ghcr",
			Name:        "test-registry",
			Description: "Updated Description",
			URL:         "http://example.com/updated",
			Insecure:    true,
		}
		Expect(k8sClient.Update(ctx, registry)).To(Succeed())

		// Wait briefly to let the update propagate.
		time.Sleep(2 * time.Second)

		// Reconcile the Registry.
		reconciler := &RegistryReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}
		_, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: nsName})
		Expect(err).NotTo(HaveOccurred())

		// Verify that a PUT request was made.
		Expect(atomic.LoadInt32(&putCount)).To(Equal(int32(1)))
	})

	It("should delete the registry in Harbor when the CR is deleted", func() {
		// Pre-populate existing registry.
		existingRegistry = &harborRegistryResponse{
			ID:          1,
			Name:        "test-registry",
			URL:         "http://example.com",
			Description: "Test Description",
			Type:        "github-ghcr",
			Insecure:    false,
		}

		// Ensure HarborConnection and Secret exist.
		harborConn := &harborv1alpha1.HarborConnection{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-harbor",
				Namespace: "default",
			},
			Spec: harborv1alpha1.HarborConnectionSpec{
				BaseURL: testServer.URL,
				Credentials: &harborv1alpha1.Credentials{
					Type:            "basic",
					AccessKey:       "test-user",
					AccessSecretRef: "test-secret",
				},
			},
		}
		err := k8sClient.Get(ctx, types.NamespacedName{Name: "test-harbor", Namespace: "default"}, harborConn)
		if err != nil && errors.IsNotFound(err) {
			Expect(k8sClient.Create(ctx, harborConn)).To(Succeed())
		}
		secret := &corev1.Secret{}
		err = k8sClient.Get(ctx, types.NamespacedName{Name: "test-secret", Namespace: "default"}, secret)
		if err != nil && errors.IsNotFound(err) {
			secret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-secret",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"access_secret": []byte("test-password"),
				},
			}
			Expect(k8sClient.Create(ctx, secret)).To(Succeed())
		}

		// Create or update the Registry CR.
		registry := &harborv1alpha1.Registry{}
		err = k8sClient.Get(ctx, nsName, registry)
		if err != nil && errors.IsNotFound(err) {
			registry = &harborv1alpha1.Registry{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
			}
			Expect(k8sClient.Create(ctx, registry)).To(Succeed())
		}

		// Simulate deletion by setting a deletion timestamp and ensuring the finalizer is present.
		now := metav1.Now()
		registry.SetDeletionTimestamp(&now)
		if !controllerutil.ContainsFinalizer(registry, finalizerName) {
			controllerutil.AddFinalizer(registry, finalizerName)
		}
		Expect(k8sClient.Update(ctx, registry)).To(Succeed())

		// Reconcile the Registry.
		reconciler := &RegistryReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}
		_, err = reconciler.Reconcile(ctx, reconcile.Request{NamespacedName: nsName})
		Expect(err).NotTo(HaveOccurred())

		// Verify that a DELETE request was made.
		Expect(atomic.LoadInt32(&deleteCount)).To(Equal(int32(1)))
	})
})
