package controller

import (
	"testing"
	"time"

	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestOperatorOptionsCreationPolicy(t *testing.T) {
	defaults := OperatorOptions{}
	if got := defaults.effectiveCreationPolicy(""); got != harborv1alpha1.CreationPolicyCreate {
		t.Fatalf("effectiveCreationPolicy(\"\") = %q, want %q", got, harborv1alpha1.CreationPolicyCreate)
	}

	options, err := NewOperatorOptions(OperatorConfig{
		DefaultCreationPolicy: harborv1alpha1.CreationPolicyAdopt,
		HarborRequestTimeout:  30 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewOperatorOptions() error = %v", err)
	}
	if got := options.effectiveCreationPolicy(""); got != harborv1alpha1.CreationPolicyAdopt {
		t.Fatalf("effectiveCreationPolicy(\"\") = %q, want %q", got, harborv1alpha1.CreationPolicyAdopt)
	}
	if got := options.effectiveCreationPolicy(harborv1alpha1.CreationPolicyCreateOrAdopt); got != harborv1alpha1.CreationPolicyCreateOrAdopt {
		t.Fatalf("explicit policy = %q, want %q", got, harborv1alpha1.CreationPolicyCreateOrAdopt)
	}
	if _, err := NewOperatorOptions(OperatorConfig{
		DefaultCreationPolicy: "Replace",
		HarborRequestTimeout:  30 * time.Second,
	}); err == nil {
		t.Fatal("NewOperatorOptions() accepted an unsupported policy")
	}
}

func TestOperatorOptionsDriftDetectionInterval(t *testing.T) {
	options, err := NewOperatorOptions(OperatorConfig{
		DefaultCreationPolicy:         harborv1alpha1.CreationPolicyCreate,
		DefaultDriftDetectionInterval: 10 * time.Minute,
		HarborRequestTimeout:          30 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewOperatorOptions() error = %v", err)
	}
	result, err := returnWithDriftDetection(options, &harborv1alpha1.HarborSpecBase{})
	if err != nil {
		t.Fatalf("returnWithDriftDetection() error = %v", err)
	}
	if result.RequeueAfter != 10*time.Minute {
		t.Fatalf("RequeueAfter = %v, want %v", result.RequeueAfter, 10*time.Minute)
	}

	zero := metav1.Duration{}
	result, err = returnWithDriftDetection(options, &harborv1alpha1.HarborSpecBase{DriftDetectionInterval: &zero})
	if err != nil {
		t.Fatalf("returnWithDriftDetection() with explicit zero error = %v", err)
	}
	if result.RequeueAfter != 0 {
		t.Fatalf("explicit zero RequeueAfter = %v, want 0", result.RequeueAfter)
	}

	oneMinute := metav1.Duration{Duration: time.Minute}
	result, err = returnWithDriftDetection(options, &harborv1alpha1.HarborSpecBase{DriftDetectionInterval: &oneMinute})
	if err != nil {
		t.Fatalf("returnWithDriftDetection() with explicit interval error = %v", err)
	}
	if result.RequeueAfter != time.Minute {
		t.Fatalf("explicit RequeueAfter = %v, want %v", result.RequeueAfter, time.Minute)
	}

	if _, err := NewOperatorOptions(OperatorConfig{
		DefaultCreationPolicy:         harborv1alpha1.CreationPolicyCreate,
		DefaultDriftDetectionInterval: -time.Second,
		HarborRequestTimeout:          30 * time.Second,
	}); err == nil {
		t.Fatal("NewOperatorOptions() accepted a negative drift detection interval")
	}
}

func TestOperatorOptionsHarborRequestTimeout(t *testing.T) {
	if got := (OperatorOptions{}).requestTimeout(); got != 30*time.Second {
		t.Fatalf("requestTimeout() = %v, want %v", got, 30*time.Second)
	}
	options, err := NewOperatorOptions(OperatorConfig{
		DefaultCreationPolicy: harborv1alpha1.CreationPolicyCreate,
		HarborRequestTimeout:  45 * time.Second,
	})
	if err != nil {
		t.Fatalf("NewOperatorOptions() error = %v", err)
	}
	harborClient, err := newHarborClient(options, "https://harbor.example.com", "user", "password", "")
	if err != nil {
		t.Fatalf("newHarborClient() error = %v", err)
	}
	if harborClient.HTTPClient.Timeout != 45*time.Second {
		t.Fatalf("HTTP client timeout = %v, want %v", harborClient.HTTPClient.Timeout, 45*time.Second)
	}
	if _, err := NewOperatorOptions(OperatorConfig{
		DefaultCreationPolicy: harborv1alpha1.CreationPolicyCreate,
	}); err == nil {
		t.Fatal("NewOperatorOptions() accepted a zero request timeout")
	}
}
