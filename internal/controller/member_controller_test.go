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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
)

const projectMembersPath = "/api/v2.0/projects/demo/members"

var _ = Describe("Member Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"
		const adminSecretName = "harbor-admin-member"
		const connName = "harbor-conn-member"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		var server *httptest.Server

		BeforeEach(func() {
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				user, pass, ok := r.BasicAuth()
				if !ok || user != testAdminUser || pass != testPassword {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				if r.Method == http.MethodGet && r.URL.Path == projectMembersPath {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte("[]"))
					return
				}
				if r.Method == http.MethodPost && r.URL.Path == projectMembersPath {
					w.Header().Set("Location", projectMembersPath+"/11")
					w.WriteHeader(http.StatusCreated)
					return
				}
				http.NotFound(w, r)
			}))

			Expect(createPasswordSecret(ctx, k8sClient, adminSecretName, testPassword)).To(Succeed())
			Expect(createHarborConnection(ctx, k8sClient, connName, server.URL, adminSecretName)).To(Succeed())

			By("creating the custom resource for the Kind Member")
			resource := &harborv1alpha1.Member{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: harborv1alpha1.MemberSpec{
					HarborSpecBase: harborv1alpha1.HarborSpecBase{
						HarborConnectionRef: harborv1alpha1.HarborConnectionReference{Name: connName},
					},
					ProjectRef: "demo",
					Role:       "developer",
					MemberUser: &harborv1alpha1.MemberUser{
						Username: "alice",
					},
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())
		})

		AfterEach(func() {
			server.Close()
			resource := &harborv1alpha1.Member{}
			_ = k8sClient.Get(ctx, typeNamespacedName, resource)

			By("Cleanup the specific resource instance Member")
			_ = k8sClient.Delete(ctx, resource)

			conn := &harborv1alpha1.HarborConnection{}
			_ = k8sClient.Get(ctx, types.NamespacedName{Name: connName, Namespace: "default"}, conn)
			_ = k8sClient.Delete(ctx, conn)
			secret := &corev1.Secret{}
			_ = k8sClient.Get(ctx, types.NamespacedName{Name: adminSecretName, Namespace: "default"}, secret)
			_ = k8sClient.Delete(ctx, secret)
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &MemberReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())
			out := &harborv1alpha1.Member{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, out)).To(Succeed())
			cond := meta.FindStatusCondition(out.Status.Conditions, ConditionReady)
			Expect(cond).NotTo(BeNil())
			Expect(cond.Status).To(Equal(metav1.ConditionTrue))
			// Example: If you expect a certain status condition after reconciliation, verify it here.
		})
	})

	Context("When member already exists and takeover is disabled", func() {
		const resourceName = "existing-member"
		const adminSecretName = "harbor-admin-member-existing"
		const connName = "harbor-conn-member-existing"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		var server *httptest.Server

		BeforeEach(func() {
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				user, pass, ok := r.BasicAuth()
				if !ok || user != testAdminUser || pass != testPassword {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				if r.Method == http.MethodGet && r.URL.Path == projectMembersPath {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`[{"id":9,"entity_name":"alice","entity_type":"u","role_id":2}]`))
					return
				}
				http.NotFound(w, r)
			}))

			Expect(createPasswordSecret(ctx, k8sClient, adminSecretName, testPassword)).To(Succeed())
			Expect(createHarborConnection(ctx, k8sClient, connName, server.URL, adminSecretName)).To(Succeed())

			resource := &harborv1alpha1.Member{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: harborv1alpha1.MemberSpec{
					HarborSpecBase: harborv1alpha1.HarborSpecBase{
						HarborConnectionRef: harborv1alpha1.HarborConnectionReference{Name: connName},
					},
					AllowTakeover: false,
					ProjectRef:    "demo",
					Role:          "developer",
					MemberUser: &harborv1alpha1.MemberUser{
						Username: "alice",
					},
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())
		})

		AfterEach(func() {
			server.Close()
			resource := &harborv1alpha1.Member{}
			_ = k8sClient.Get(ctx, typeNamespacedName, resource)
			_ = k8sClient.Delete(ctx, resource)

			conn := &harborv1alpha1.HarborConnection{}
			_ = k8sClient.Get(ctx, types.NamespacedName{Name: connName, Namespace: "default"}, conn)
			_ = k8sClient.Delete(ctx, conn)
			secret := &corev1.Secret{}
			_ = k8sClient.Get(ctx, types.NamespacedName{Name: adminSecretName, Namespace: "default"}, secret)
			_ = k8sClient.Delete(ctx, secret)
		})

		It("should return an error", func() {
			controllerReconciler := &MemberReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).To(HaveOccurred())
		})
	})
})
