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
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
)

var _ = Describe("Retention Policy Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "retention-policy"
		const adminSecretName = "harbor-admin-retention"
		const connName = "harbor-conn-retention"

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
				if r.Method == http.MethodPost && r.URL.Path == "/api/v2.0/retentions" {
					w.Header().Set("Location", "/api/v2.0/retentions/17")
					w.WriteHeader(http.StatusCreated)
					return
				}
				http.NotFound(w, r)
			}))

			Expect(createPasswordSecret(ctx, k8sClient, adminSecretName, testPassword)).To(Succeed())
			Expect(createHarborConnection(ctx, k8sClient, connName, server.URL, adminSecretName)).To(Succeed())

			project := &harborv1alpha1.Project{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "demo",
					Namespace: "default",
				},
				Spec: harborv1alpha1.ProjectSpec{
					HarborSpecBase: harborv1alpha1.HarborSpecBase{
						HarborConnectionRef: connName,
					},
					Name:   "demo",
					Public: true,
				},
			}
			Expect(k8sClient.Create(ctx, project)).To(Succeed())
			project.Status.HarborProjectID = 42
			Expect(k8sClient.Status().Update(ctx, project)).To(Succeed())

			resource := &harborv1alpha1.RetentionPolicy{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: harborv1alpha1.RetentionPolicySpec{
					HarborSpecBase: harborv1alpha1.HarborSpecBase{
						HarborConnectionRef: connName,
					},
					ProjectRef: &harborv1alpha1.ProjectReference{Name: "demo"},
					Algorithm:  "or",
					Trigger: &harborv1alpha1.RetentionTrigger{
						Kind: "Schedule",
						Settings: map[string]apiextensionsv1.JSON{
							"cron": {Raw: []byte(`"0 0 0 * * *"`)},
						},
					},
					Rules: []harborv1alpha1.RetentionRule{
						{
							Action:   "retain",
							Template: "latestPushedK",
							Params: map[string]apiextensionsv1.JSON{
								"latestPushedK": {Raw: []byte(`{"value":3}`)},
							},
							TagSelectors: []harborv1alpha1.RetentionSelector{
								{Kind: "doublestar", Decoration: "matches", Pattern: "**"},
							},
							ScopeSelectors: map[string][]harborv1alpha1.RetentionSelector{
								"repository": {
									{Kind: "doublestar", Decoration: "repoMatches", Pattern: "**"},
								},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())
		})

		AfterEach(func() {
			server.Close()
			resource := &harborv1alpha1.RetentionPolicy{}
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
			controllerReconciler := &RetentionPolicyReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			out := &harborv1alpha1.RetentionPolicy{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, out)).To(Succeed())
			Expect(out.Status.HarborRetentionID).To(Equal(17))
			cond := meta.FindStatusCondition(out.Status.Conditions, ConditionReady)
			Expect(cond).NotTo(BeNil())
			Expect(cond.Status).To(Equal(metav1.ConditionTrue))
		})
	})
})
