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

var _ = Describe("Quota Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "quota"
		const adminSecretName = "harbor-admin-quota"
		const connName = "harbor-conn-quota"

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
				switch {
				case r.Method == http.MethodGet && r.URL.Path == "/api/v2.0/quotas":
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`[{"id":99,"hard":{"storage":1000}}]`))
					return
				case r.Method == http.MethodGet && r.URL.Path == "/api/v2.0/quotas/99":
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"id":99,"hard":{"storage":1000}}`))
					return
				case r.Method == http.MethodPut && r.URL.Path == "/api/v2.0/quotas/99":
					w.WriteHeader(http.StatusOK)
					return
				default:
					http.NotFound(w, r)
				}
			}))

			Expect(createPasswordSecret(ctx, k8sClient, adminSecretName, testPassword)).To(Succeed())
			Expect(createHarborConnection(ctx, k8sClient, connName, server.URL, adminSecretName)).To(Succeed())

			project := &harborv1alpha1.Project{
				ObjectMeta: metav1.ObjectMeta{Name: "demo", Namespace: "default"},
				Spec: harborv1alpha1.ProjectSpec{
					HarborSpecBase: harborv1alpha1.HarborSpecBase{HarborConnectionRef: &harborv1alpha1.HarborConnectionReference{Name: connName}},
					Public:         true,
				},
			}
			Expect(k8sClient.Create(ctx, project)).To(Succeed())
			project.Status.HarborProjectID = 42
			Expect(k8sClient.Status().Update(ctx, project)).To(Succeed())

			resource := &harborv1alpha1.Quota{
				ObjectMeta: metav1.ObjectMeta{Name: resourceName, Namespace: "default"},
				Spec: harborv1alpha1.QuotaSpec{
					HarborSpecBase: harborv1alpha1.HarborSpecBase{HarborConnectionRef: &harborv1alpha1.HarborConnectionReference{Name: connName}},
					ProjectRef:     &harborv1alpha1.ProjectReference{Name: "demo"},
					Hard:           map[string]int64{"storage": 2000},
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())
		})

		AfterEach(func() {
			server.Close()
			resource := &harborv1alpha1.Quota{}
			_ = k8sClient.Get(ctx, typeNamespacedName, resource)
			_ = k8sClient.Delete(ctx, resource)

			project := &harborv1alpha1.Project{}
			_ = k8sClient.Get(ctx, types.NamespacedName{Name: "demo", Namespace: "default"}, project)
			_ = k8sClient.Delete(ctx, project)

			conn := &harborv1alpha1.HarborConnection{}
			_ = k8sClient.Get(ctx, types.NamespacedName{Name: connName, Namespace: "default"}, conn)
			_ = k8sClient.Delete(ctx, conn)

			secret := &corev1.Secret{}
			_ = k8sClient.Get(ctx, types.NamespacedName{Name: adminSecretName, Namespace: "default"}, secret)
			_ = k8sClient.Delete(ctx, secret)
		})

		It("should successfully reconcile the resource", func() {
			controllerReconciler := &QuotaReconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())

			out := &harborv1alpha1.Quota{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, out)).To(Succeed())
			Expect(out.Status.HarborQuotaID).To(Equal(99))
			cond := meta.FindStatusCondition(out.Status.Conditions, ConditionReady)
			Expect(cond).NotTo(BeNil())
			Expect(cond.Status).To(Equal(metav1.ConditionTrue))
		})
	})
})
