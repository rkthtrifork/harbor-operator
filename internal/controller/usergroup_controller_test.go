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

var _ = Describe("UserGroupClaim Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "user-group-claim"
		const harborGroupName = "external-user-group"
		const adminSecretName = "harbor-admin-group"
		const connName = "harbor-conn-group"

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
				if r.Method == http.MethodGet && r.URL.Path == "/api/v2.0/usergroups" {
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`[]`))
					return
				}
				if r.Method == http.MethodPost && r.URL.Path == "/api/v2.0/usergroups" {
					w.Header().Set("Location", "/api/v2.0/usergroups/3")
					w.WriteHeader(http.StatusCreated)
					return
				}
				http.NotFound(w, r)
			}))

			Expect(createPasswordSecret(ctx, k8sClient, adminSecretName, testPassword)).To(Succeed())
			Expect(createHarborConnection(ctx, k8sClient, connName, server.URL, adminSecretName)).To(Succeed())

			resource := &harborv1alpha1.UserGroupClaim{
				ObjectMeta: metav1.ObjectMeta{Name: resourceName, Namespace: "default"},
				Spec: harborv1alpha1.UserGroupClaimSpec{
					HarborClaimSpecBase: harborv1alpha1.HarborClaimSpecBase{HarborConnectionRef: &harborv1alpha1.HarborConnectionReference{Name: connName}},
					GroupName:           harborGroupName,
					GroupType:           2,
				},
			}
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())
		})

		AfterEach(func() {
			server.Close()
			resource := &harborv1alpha1.UserGroupClaim{}
			_ = k8sClient.Get(ctx, typeNamespacedName, resource)
			resource.Finalizers = nil
			_ = k8sClient.Update(ctx, resource)
			_ = k8sClient.Delete(ctx, resource)

			conn := &harborv1alpha1.HarborConnection{}
			_ = k8sClient.Get(ctx, types.NamespacedName{Name: connName, Namespace: "default"}, conn)
			_ = k8sClient.Delete(ctx, conn)

			secret := &corev1.Secret{}
			_ = k8sClient.Get(ctx, types.NamespacedName{Name: adminSecretName, Namespace: "default"}, secret)
			_ = k8sClient.Delete(ctx, secret)
		})

		It("should create a shared claim without owning the Harbor group", func() {
			controllerReconciler := &UserGroupClaimReconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())
			_, err = controllerReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())

			out := &harborv1alpha1.UserGroupClaim{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, out)).To(Succeed())
			Expect(out.Status.HarborGroupID).To(Equal(3))
			cond := meta.FindStatusCondition(out.Status.Conditions, ConditionReady)
			Expect(cond).NotTo(BeNil())
			Expect(cond.Status).To(Equal(metav1.ConditionTrue))
		})

		It("keeps the claim while a deleting Member still references it", func() {
			claimReconciler := &UserGroupClaimReconciler{Client: k8sClient, Scheme: k8sClient.Scheme()}
			_, err := claimReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
			Expect(err).NotTo(HaveOccurred())

			member := &harborv1alpha1.Member{
				ObjectMeta: metav1.ObjectMeta{Name: "claim-deletion-member", Namespace: "default", Finalizers: []string{finalizerName}},
				Spec: harborv1alpha1.MemberSpec{
					ProjectRef: harborv1alpha1.ProjectReference{Name: "demo"},
					Role:       "developer",
					MemberGroup: &harborv1alpha1.MemberGroup{
						GroupClaimRef: harborv1alpha1.UserGroupClaimReference{Name: resourceName},
					},
				},
			}
			Expect(k8sClient.Create(ctx, member)).To(Succeed())
			Expect(k8sClient.Delete(ctx, member)).To(Succeed())
			Expect(k8sClient.Delete(ctx, &harborv1alpha1.UserGroupClaim{ObjectMeta: metav1.ObjectMeta{Name: resourceName, Namespace: "default"}})).To(Succeed())

			_, err = claimReconciler.Reconcile(ctx, reconcile.Request{NamespacedName: typeNamespacedName})
			Expect(err).To(MatchError(ContainSubstring("still referenced by an active Member")))
			out := &harborv1alpha1.UserGroupClaim{}
			Expect(k8sClient.Get(ctx, typeNamespacedName, out)).To(Succeed())
			Expect(out.Finalizers).To(ContainElement(finalizerName))

			deletingMember := &harborv1alpha1.Member{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: member.Name, Namespace: member.Namespace}, deletingMember)).To(Succeed())
			deletingMember.Finalizers = nil
			Expect(k8sClient.Update(ctx, deletingMember)).To(Succeed())
		})
	})
})
