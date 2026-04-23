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

var _ = Describe("ReplicationPolicy Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "replication-policy"
		const adminSecretName = "harbor-admin-repl"
		const connName = "harbor-conn-repl"

		ctx := context.Background()
		typeNamespacedName := types.NamespacedName{Name: resourceName, Namespace: "default"}
		var server *httptest.Server

		BeforeEach(func() {
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				user, pass, ok := r.BasicAuth()
				if !ok || user != testAdminUser || pass != testPassword {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				if r.Method == http.MethodPost && r.URL.Path == "/api/v2.0/replication/policies" {
					w.Header().Set("Location", "/api/v2.0/replication/policies/55")
					w.WriteHeader(http.StatusCreated)
					return
				}
				http.NotFound(w, r)
			}))

			Expect(createPasswordSecret(ctx, k8sClient, adminSecretName, testPassword)).To(Succeed())
			Expect(createHarborConnection(ctx, k8sClient, connName, server.URL, adminSecretName)).To(Succeed())

			srcRegistry := &harborv1alpha1.Registry{
				ObjectMeta: metav1.ObjectMeta{Name: "src-reg", Namespace: "default"},
				Spec: harborv1alpha1.RegistrySpec{
					HarborSpecBase: harborv1alpha1.HarborSpecBase{HarborConnectionRef: &harborv1alpha1.HarborConnectionReference{Name: connName}},
					Type:           "docker-registry",
					URL:            "https://example.com",
				},
			}
			destRegistry := &harborv1alpha1.Registry{
				ObjectMeta: metav1.ObjectMeta{Name: "dest-reg", Namespace: "default"},
				Spec: harborv1alpha1.RegistrySpec{
					HarborSpecBase: harborv1alpha1.HarborSpecBase{HarborConnectionRef: &harborv1alpha1.HarborConnectionReference{Name: connName}},
					Type:           "docker-registry",
					URL:            "https://example.org",
				},
			}
			Expect(k8sClient.Create(ctx, srcRegistry)).To(Succeed())
			Expect(k8sClient.Create(ctx, destRegistry)).To(Succeed())
			srcRegistry.Status.HarborRegistryID = 11
			Expect(k8sClient.Status().Update(ctx, srcRegistry)).To(Succeed())
			destRegistry.Status.HarborRegistryID = 22
			Expect(k8sClient.Status().Update(ctx, destRegistry)).To(Succeed())

			resource := &harborv1alpha1.ReplicationPolicy{
				ObjectMeta: metav1.ObjectMeta{Name: resourceName, Namespace: "default"},
				Spec: harborv1alpha1.ReplicationPolicySpec{
					HarborSpecBase:         harborv1alpha1.HarborSpecBase{HarborConnectionRef: &harborv1alpha1.HarborConnectionReference{Name: connName}},
					SourceRegistryRef:      &harborv1alpha1.RegistryReference{Name: "src-reg"},
					DestinationRegistryRef: &harborv1alpha1.RegistryReference{Name: "dest-reg"},
					ReplicateDeletion:      func() *bool { v := true; return &v }(),
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())
		})

		AfterEach(func() {
			server.Close()
			resource := &harborv1alpha1.ReplicationPolicy{}
			_ = k8sClient.Get(ctx, typeNamespacedName, resource)
			_ = k8sClient.Delete(ctx, resource)

			srcRegistry := &harborv1alpha1.Registry{}
			_ = k8sClient.Get(ctx, types.NamespacedName{Name: "src-reg", Namespace: "default"}, srcRegistry)
			_ = k8sClient.Delete(ctx, srcRegistry)
			destRegistry := &harborv1alpha1.Registry{}
			_ = k8sClient.Get(ctx, types.NamespacedName{Name: "dest-reg", Namespace: "default"}, destRegistry)
			_ = k8sClient.Delete(ctx, destRegistry)

			conn := &harborv1alpha1.HarborConnection{}
			_ = k8sClient.Get(ctx, types.NamespacedName{Name: connName, Namespace: "default"}, conn)
			_ = k8sClient.Delete(ctx, conn)

			secret := &corev1.Secret{}
			_ = k8sClient.Get(ctx, types.NamespacedName{Name: adminSecretName, Namespace: "default"}, secret)
			_ = k8sClient.Delete(ctx, secret)
		})

		It("should successfully reconcile the resource", func() {
			controllerReconciler := &ReplicationPolicyReconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())

			out := &harborv1alpha1.ReplicationPolicy{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, out)).To(Succeed())
			Expect(out.Status.HarborReplicationPolicyID).To(Equal(55))
			cond := meta.FindStatusCondition(out.Status.Conditions, ConditionReady)
			Expect(cond).NotTo(BeNil())
			Expect(cond.Status).To(Equal(metav1.ConditionTrue))
		})
	})
})
