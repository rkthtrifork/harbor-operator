package controller

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
)

var _ = Describe("CRD validation and defaulting", func() {
	ctx := context.Background()

	expectInvalid := func(obj client.Object) {
		err := k8sClient.Create(ctx, obj)
		Expect(err).To(HaveOccurred())
		Expect(apierrors.IsInvalid(err)).To(BeTrue(), err.Error())
	}

	It("defaults common Harbor spec fields", func() {
		project := &harborv1alpha1.Project{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      "defaults-project",
			},
			Spec: harborv1alpha1.ProjectSpec{
				HarborSpecBase: harborv1alpha1.HarborSpecBase{
					HarborConnectionRef: &harborv1alpha1.HarborConnectionReference{Name: "conn"},
				},
			},
		}

		Expect(k8sClient.Create(ctx, project)).To(Succeed())
		DeferCleanup(func() { _ = k8sClient.Delete(ctx, project) })

		var stored harborv1alpha1.Project
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(project), &stored)).To(Succeed())
		Expect(stored.Spec.HarborConnectionRef).NotTo(BeNil())
		Expect(stored.Spec.HarborConnectionRef.Kind).To(BeEmpty())
		Expect(stored.Spec.DeletionPolicy).To(Equal(harborv1alpha1.DeletionPolicyDelete))
	})

	It("defaults webhook enabled to true", func() {
		policy := &harborv1alpha1.WebhookPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      "default-enabled-webhook",
			},
			Spec: harborv1alpha1.WebhookPolicySpec{
				HarborSpecBase: harborv1alpha1.HarborSpecBase{
					HarborConnectionRef: &harborv1alpha1.HarborConnectionReference{Name: "conn"},
				},
				ProjectRef: &harborv1alpha1.ProjectReference{Name: "library"},
				EventTypes: []string{"PUSH_ARTIFACT"},
				Targets: []harborv1alpha1.WebhookTargetSpec{
					{Type: "http", Address: "https://example.invalid/hook"},
				},
			},
		}

		Expect(k8sClient.Create(ctx, policy)).To(Succeed())
		DeferCleanup(func() { _ = k8sClient.Delete(ctx, policy) })

		var stored harborv1alpha1.WebhookPolicy
		Expect(k8sClient.Get(ctx, client.ObjectKeyFromObject(policy), &stored)).To(Succeed())
		Expect(stored.Spec.Enabled).NotTo(BeNil())
		Expect(*stored.Spec.Enabled).To(BeTrue())
	})

	It("rejects HarborConnection with both caBundle and caBundleSecretRef", func() {
		expectInvalid(&harborv1alpha1.HarborConnection{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      "invalid-harbor-connection",
			},
			Spec: harborv1alpha1.HarborConnectionSpec{
				BaseURL:  "https://harbor.example.com",
				CABundle: "pem",
				CABundleSecretRef: &harborv1alpha1.SecretReference{
					Name: "ca-secret",
				},
			},
		})
	})

	It("rejects schedules without cron when the type is not Manual or None", func() {
		expectInvalid(&harborv1alpha1.GCSchedule{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      "invalid-gc-schedule",
			},
			Spec: harborv1alpha1.GCScheduleSpec{
				HarborSpecBase: harborv1alpha1.HarborSpecBase{
					HarborConnectionRef: &harborv1alpha1.HarborConnectionReference{Name: "conn"},
				},
				Schedule: harborv1alpha1.ScheduleSpec{Type: harborv1alpha1.ScheduleTypeDaily},
			},
		})
	})

	It("rejects scan all schedules with type Manual", func() {
		expectInvalid(&harborv1alpha1.ScanAllSchedule{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      "invalid-scan-all-manual",
			},
			Spec: harborv1alpha1.ScanAllScheduleSpec{
				HarborSpecBase: harborv1alpha1.HarborSpecBase{
					HarborConnectionRef: &harborv1alpha1.HarborConnectionReference{Name: "conn"},
				},
				Schedule: harborv1alpha1.ScheduleSpec{Type: harborv1alpha1.ScheduleTypeManual},
			},
		})
	})

	It("rejects members without exactly one identity", func() {
		expectInvalid(&harborv1alpha1.Member{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      "invalid-member-none",
			},
			Spec: harborv1alpha1.MemberSpec{
				HarborSpecBase: harborv1alpha1.HarborSpecBase{
					HarborConnectionRef: &harborv1alpha1.HarborConnectionReference{Name: "conn"},
				},
				ProjectRef: harborv1alpha1.ProjectReference{Name: "library"},
				Role:       "developer",
			},
		})

		expectInvalid(&harborv1alpha1.Member{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      "invalid-member-both",
			},
			Spec: harborv1alpha1.MemberSpec{
				HarborSpecBase: harborv1alpha1.HarborSpecBase{
					HarborConnectionRef: &harborv1alpha1.HarborConnectionReference{Name: "conn"},
				},
				ProjectRef:  harborv1alpha1.ProjectReference{Name: "library"},
				Role:        "developer",
				MemberUser:  &harborv1alpha1.MemberUser{UserRef: harborv1alpha1.UserReference{Name: "alice"}},
				MemberGroup: &harborv1alpha1.MemberGroup{GroupRef: harborv1alpha1.UserGroupReference{Name: "devs"}},
			},
		})
	})

	It("rejects members with an unsupported role", func() {
		expectInvalid(&harborv1alpha1.Member{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      "invalid-member-role",
			},
			Spec: harborv1alpha1.MemberSpec{
				HarborSpecBase: harborv1alpha1.HarborSpecBase{
					HarborConnectionRef: &harborv1alpha1.HarborConnectionReference{Name: "conn"},
				},
				ProjectRef: harborv1alpha1.ProjectReference{Name: "library"},
				Role:       "owner",
				MemberUser: &harborv1alpha1.MemberUser{UserRef: harborv1alpha1.UserReference{Name: "alice"}},
			},
		})
	})

	It("rejects webhook policies without projectRef", func() {
		expectInvalid(&harborv1alpha1.WebhookPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      "invalid-webhook-project-selector",
			},
			Spec: harborv1alpha1.WebhookPolicySpec{
				HarborSpecBase: harborv1alpha1.HarborSpecBase{
					HarborConnectionRef: &harborv1alpha1.HarborConnectionReference{Name: "conn"},
				},
				EventTypes: []string{"PUSH_ARTIFACT"},
				Targets: []harborv1alpha1.WebhookTargetSpec{
					{Type: "http", Address: "https://example.invalid/hook"},
				},
			},
		})
	})

	It("rejects webhook targets with both authHeader and authHeaderSecretRef", func() {
		expectInvalid(&harborv1alpha1.WebhookPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      "invalid-webhook-target-auth",
			},
			Spec: harborv1alpha1.WebhookPolicySpec{
				HarborSpecBase: harborv1alpha1.HarborSpecBase{
					HarborConnectionRef: &harborv1alpha1.HarborConnectionReference{Name: "conn"},
				},
				ProjectRef: &harborv1alpha1.ProjectReference{Name: "library"},
				EventTypes: []string{"PUSH_ARTIFACT"},
				Targets: []harborv1alpha1.WebhookTargetSpec{
					{
						Type:       "http",
						Address:    "https://example.invalid/hook",
						AuthHeader: "Bearer token",
						AuthHeaderSecretRef: &harborv1alpha1.SecretReference{
							Name: "auth-secret",
						},
					},
				},
			},
		})
	})

	It("rejects registries with both caCertificate and caCertificateRef", func() {
		expectInvalid(&harborv1alpha1.Registry{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      "invalid-registry-ca",
			},
			Spec: harborv1alpha1.RegistrySpec{
				HarborSpecBase: harborv1alpha1.HarborSpecBase{
					HarborConnectionRef: &harborv1alpha1.HarborConnectionReference{Name: "conn"},
				},
				Type:          "docker-registry",
				URL:           "https://registry.example.com",
				CACertificate: "pem",
				CACertificateRef: &harborv1alpha1.SecretReference{
					Name: "ca-secret",
				},
			},
		})
	})

	It("rejects replication policies without registry refs", func() {
		expectInvalid(&harborv1alpha1.ReplicationPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      "invalid-replication-registries",
			},
			Spec: harborv1alpha1.ReplicationPolicySpec{
				HarborSpecBase: harborv1alpha1.HarborSpecBase{
					HarborConnectionRef: &harborv1alpha1.HarborConnectionReference{Name: "conn"},
				},
			},
		})
	})

	It("rejects scheduled replication policies without cron", func() {
		expectInvalid(&harborv1alpha1.ReplicationPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      "invalid-replication-cron",
			},
			Spec: harborv1alpha1.ReplicationPolicySpec{
				HarborSpecBase: harborv1alpha1.HarborSpecBase{
					HarborConnectionRef: &harborv1alpha1.HarborConnectionReference{Name: "conn"},
				},
				SourceRegistryRef:      &harborv1alpha1.RegistryReference{Name: "src"},
				DestinationRegistryRef: &harborv1alpha1.RegistryReference{Name: "dest"},
				Trigger: &harborv1alpha1.ReplicationTriggerSpec{
					Type:     "scheduled",
					Settings: &harborv1alpha1.ReplicationTriggerSettings{},
				},
			},
		})
	})

	It("rejects scanner registrations with both credential sources", func() {
		expectInvalid(&harborv1alpha1.ScannerRegistration{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      "invalid-scanner-credential",
			},
			Spec: harborv1alpha1.ScannerRegistrationSpec{
				HarborSpecBase: harborv1alpha1.HarborSpecBase{
					HarborConnectionRef: &harborv1alpha1.HarborConnectionReference{Name: "conn"},
				},
				URL:              "https://scanner.example.com",
				AccessCredential: "token",
				AccessCredentialSecretRef: &harborv1alpha1.SecretReference{
					Name: "scanner-secret",
				},
			},
		})
	})

	It("rejects immutable tag rules without projectRef", func() {
		expectInvalid(&harborv1alpha1.ImmutableTagRule{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      "invalid-immutable-project-selector",
			},
			Spec: harborv1alpha1.ImmutableTagRuleSpec{
				HarborSpecBase: harborv1alpha1.HarborSpecBase{
					HarborConnectionRef: &harborv1alpha1.HarborConnectionReference{Name: "conn"},
				},
			},
		})
	})

	It("rejects labels with inconsistent scope and projectRef", func() {
		expectInvalid(&harborv1alpha1.Label{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      "invalid-label-project-ref",
			},
			Spec: harborv1alpha1.LabelSpec{
				HarborSpecBase: harborv1alpha1.HarborSpecBase{
					HarborConnectionRef: &harborv1alpha1.HarborConnectionReference{Name: "conn"},
				},
				Scope:      "g",
				ProjectRef: &harborv1alpha1.ProjectReference{Name: "project"},
			},
		})

		expectInvalid(&harborv1alpha1.Label{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      "invalid-label-scope",
			},
			Spec: harborv1alpha1.LabelSpec{
				HarborSpecBase: harborv1alpha1.HarborSpecBase{
					HarborConnectionRef: &harborv1alpha1.HarborConnectionReference{Name: "conn"},
				},
				Scope: "p",
			},
		})
	})

	It("rejects retention policies without a trigger", func() {
		expectInvalid(&harborv1alpha1.RetentionPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      "invalid-retention-trigger",
			},
			Spec: harborv1alpha1.RetentionPolicySpec{
				HarborSpecBase: harborv1alpha1.HarborSpecBase{
					HarborConnectionRef: &harborv1alpha1.HarborConnectionReference{Name: "conn"},
				},
				Rules: []harborv1alpha1.RetentionRule{{Action: "delete"}},
			},
		})
	})

	It("rejects retention policies with conflicting projectRef and scope.ref", func() {
		expectInvalid(&harborv1alpha1.RetentionPolicy{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      "invalid-retention-scope",
			},
			Spec: harborv1alpha1.RetentionPolicySpec{
				HarborSpecBase: harborv1alpha1.HarborSpecBase{
					HarborConnectionRef: &harborv1alpha1.HarborConnectionReference{Name: "conn"},
				},
				ProjectRef: &harborv1alpha1.ProjectReference{Name: "project"},
				Trigger:    &harborv1alpha1.RetentionTrigger{Kind: "Manual"},
				Scope: &harborv1alpha1.RetentionScope{
					Level: "project",
					Ref:   1,
				},
				Rules: []harborv1alpha1.RetentionRule{{Action: "delete"}},
			},
		})
	})

	It("rejects robots with invalid duration", func() {
		expectInvalid(&harborv1alpha1.Robot{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: testNamespace,
				Name:      "invalid-robot-duration",
			},
			Spec: harborv1alpha1.RobotSpec{
				HarborSpecBase: harborv1alpha1.HarborSpecBase{
					HarborConnectionRef: &harborv1alpha1.HarborConnectionReference{Name: "conn"},
				},
				Level:    "system",
				Duration: -2,
				Permissions: []harborv1alpha1.RobotPermission{
					{
						Kind: "system",
						Access: []harborv1alpha1.RobotAccess{
							{Resource: harborv1alpha1.RobotResourceProject, Action: harborv1alpha1.RobotActionRead},
						},
					},
				},
			},
		})
	})
})
