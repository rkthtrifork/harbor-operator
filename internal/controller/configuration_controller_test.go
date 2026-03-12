/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"net/http"
	"net/http/httptest"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
)

var _ = Describe("Configuration Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"
		const adminSecretName = "harbor-admin-config"
		const connName = "harbor-conn-config"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		config := &harborv1alpha1.Configuration{}
		var server *httptest.Server
		var putCount int

		BeforeEach(func() {
			putCount = 0
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				user, pass, ok := r.BasicAuth()
				if !ok || user != testAdminUser || pass != testPassword {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				switch {
				case r.Method == http.MethodGet && r.URL.Path == "/api/v2.0/configurations":
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"robot_name_prefix":{"value":"robot$","editable":true}}`))
					return
				case r.Method == http.MethodPut && r.URL.Path == "/api/v2.0/configurations":
					putCount++
					w.WriteHeader(http.StatusOK)
					return
				default:
					http.NotFound(w, r)
				}
			}))

			Expect(createPasswordSecret(ctx, k8sClient, adminSecretName, testPassword)).To(Succeed())
			Expect(createHarborConnection(ctx, k8sClient, connName, server.URL, adminSecretName)).To(Succeed())

			By("creating the custom resource for the Kind Configuration")
			resource := &harborv1alpha1.Configuration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: harborv1alpha1.ConfigurationSpec{
					HarborSpecBase: harborv1alpha1.HarborSpecBase{
						HarborConnectionRef: harborv1alpha1.HarborConnectionReference{Name: connName},
					},
					Settings: map[string]apiextensionsv1.JSON{
						"robot_name_prefix": {Raw: []byte(`"robot+"`)},
					},
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())
		})

		AfterEach(func() {
			server.Close()
			resource := &harborv1alpha1.Configuration{}
			if err := k8sClient.Get(ctx, typeNamespacedName, resource); err == nil {
				resource.Finalizers = nil
				_ = k8sClient.Update(ctx, resource)
				_ = k8sClient.Delete(ctx, resource)
			}

			By("Cleanup the specific resource instance Configuration")
			conn := &harborv1alpha1.HarborConnection{}
			_ = k8sClient.Get(ctx, types.NamespacedName{Name: connName, Namespace: "default"}, conn)
			_ = k8sClient.Delete(ctx, conn)
			secret := &corev1.Secret{}
			_ = k8sClient.Get(ctx, types.NamespacedName{Name: adminSecretName, Namespace: "default"}, secret)
			_ = k8sClient.Delete(ctx, secret)
		})

		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &ConfigurationReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(putCount).To(Equal(1))

			Expect(k8sClient.Get(ctx, typeNamespacedName, config)).To(Succeed())
		})

		It("should reject another configuration targeting the same Harbor instance through a different connection", func() {
			controllerReconciler := &ConfigurationReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(putCount).To(Equal(1))

			Expect(createHarborConnection(ctx, k8sClient, "harbor-conn-config-2", server.URL, adminSecretName)).To(Succeed())
			time.Sleep(1100 * time.Millisecond)
			second := &harborv1alpha1.Configuration{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "second-resource",
					Namespace: "default",
				},
				Spec: harborv1alpha1.ConfigurationSpec{
					HarborSpecBase: harborv1alpha1.HarborSpecBase{
						HarborConnectionRef: harborv1alpha1.HarborConnectionReference{Name: "harbor-conn-config-2"},
					},
					Settings: map[string]apiextensionsv1.JSON{
						"robot_name_prefix": {Raw: []byte(`"robot-second"`)},
					},
				},
			}
			Expect(k8sClient.Create(ctx, second)).To(Succeed())
			defer func() {
				_ = k8sClient.Delete(ctx, second)
				conn := &harborv1alpha1.HarborConnection{}
				_ = k8sClient.Get(ctx, types.NamespacedName{Name: "harbor-conn-config-2", Namespace: "default"}, conn)
				_ = k8sClient.Delete(ctx, conn)
			}()

			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: "second-resource", Namespace: "default"},
			})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("conflicts with existing owner default/test-resource"))
			Expect(putCount).To(Equal(1))

			out := &harborv1alpha1.Configuration{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "second-resource", Namespace: "default"}, out)).To(Succeed())
			Expect(meta.FindStatusCondition(out.Status.Conditions, ConditionReady)).NotTo(BeNil())
			Expect(meta.FindStatusCondition(out.Status.Conditions, ConditionReady).Status).To(Equal(metav1.ConditionFalse))
			Expect(meta.FindStatusCondition(out.Status.Conditions, ConditionReady).Reason).To(Equal("ReconcileError"))
		})
	})
})
