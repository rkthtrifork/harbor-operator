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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
)

var _ = Describe("Robot Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		robot := &harborv1alpha1.Robot{}
		var server *httptest.Server

		BeforeEach(func() {
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				user, pass, ok := r.BasicAuth()
				if !ok || user != "admin" || pass != "password" {
					w.WriteHeader(http.StatusUnauthorized)
					return
				}
				if r.Method == http.MethodPost && r.URL.Path == "/api/v2.0/robots" {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"id":9,"name":"robot$test","secret":"RobotSecret123","expires_at":0}`))
					return
				}
				if r.Method == http.MethodGet && r.URL.Path == "/api/v2.0/robots/9" {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"id":9,"name":"robot$test","level":"project","description":"","disable":false,"duration":-1,"expires_at":1,"permissions":[{"kind":"project","namespace":"demo","access":[{"resource":"repository","action":"pull","effect":"allow"}]}]}`))
					return
				}
				if r.Method == http.MethodPatch && r.URL.Path == "/api/v2.0/robots/9" {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"secret":"RotatedSecret456"}`))
					return
				}
				http.NotFound(w, r)
			}))

			Expect(createPasswordSecret(ctx, k8sClient, "default", "harbor-admin", "password", "password")).To(Succeed())
			Expect(createHarborConnection(ctx, k8sClient, "default", "harbor-conn", server.URL, "admin", "harbor-admin", "password")).To(Succeed())

			By("creating the custom resource for the Kind Robot")
			resource := &harborv1alpha1.Robot{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: "default",
				},
				Spec: harborv1alpha1.RobotSpec{
					HarborSpecBase: harborv1alpha1.HarborSpecBase{
						HarborConnectionRef: "harbor-conn",
					},
					Level:    "project",
					Duration: -1,
					Permissions: []harborv1alpha1.RobotPermission{
						{
							Kind:      "project",
							Namespace: "demo",
							Access: []harborv1alpha1.RobotAccess{
								{Resource: "repository", Action: "pull", Effect: "allow"},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())
		})

		AfterEach(func() {
			server.Close()
			resource := &harborv1alpha1.Robot{}
			_ = k8sClient.Get(ctx, typeNamespacedName, resource)

			By("Cleanup the specific resource instance Robot")
			_ = k8sClient.Delete(ctx, resource)

			conn := &harborv1alpha1.HarborConnection{}
			_ = k8sClient.Get(ctx, types.NamespacedName{Name: "harbor-conn", Namespace: "default"}, conn)
			_ = k8sClient.Delete(ctx, conn)
			secret := &corev1.Secret{}
			_ = k8sClient.Get(ctx, types.NamespacedName{Name: "harbor-admin", Namespace: "default"}, secret)
			_ = k8sClient.Delete(ctx, secret)
			secret = &corev1.Secret{}
			_ = k8sClient.Get(ctx, types.NamespacedName{Name: "test-resource-secret", Namespace: "default"}, secret)
			_ = k8sClient.Delete(ctx, secret)
		})

		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &RobotReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(k8sClient.Get(ctx, typeNamespacedName, robot)).To(Succeed())
			Expect(robot.Status.HarborRobotID).To(Equal(9))

			secret := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "test-resource-secret", Namespace: "default"}, secret)).To(Succeed())
			Expect(string(secret.Data["secret"])).To(Equal("RobotSecret123"))

			// Reconcile again to observe expiration and rotate.
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: "test-resource-secret", Namespace: "default"}, secret)).To(Succeed())
			Expect(string(secret.Data["secret"])).To(Equal("RotatedSecret456"))
		})
	})
})
