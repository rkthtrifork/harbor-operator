package controller

import (
	"fmt"
	"strings"
	"time"

	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const defaultHarborRequestTimeout = 30 * time.Second

// OperatorOptions is immutable runtime configuration shared by reconcilers.
// Construct it before starting the controller manager and pass it by value.
type OperatorOptions struct {
	forcedHarborConnection        string
	defaultCreationPolicy         harborv1alpha1.CreationPolicy
	defaultDriftDetectionInterval time.Duration
	harborRequestTimeout          time.Duration
	secretReader                  client.Reader
}

// OperatorConfig contains the validated startup values used to construct
// OperatorOptions.
type OperatorConfig struct {
	ForcedHarborConnection        string
	DefaultCreationPolicy         harborv1alpha1.CreationPolicy
	DefaultDriftDetectionInterval time.Duration
	HarborRequestTimeout          time.Duration
}

func NewOperatorOptions(config OperatorConfig) (OperatorOptions, error) {
	if err := validateDefaultCreationPolicy(config.DefaultCreationPolicy); err != nil {
		return OperatorOptions{}, err
	}
	if config.DefaultDriftDetectionInterval < 0 {
		return OperatorOptions{}, fmt.Errorf("default drift detection interval must not be negative")
	}
	if config.HarborRequestTimeout <= 0 {
		return OperatorOptions{}, fmt.Errorf("harbor request timeout must be greater than zero")
	}

	return OperatorOptions{
		forcedHarborConnection:        strings.TrimSpace(config.ForcedHarborConnection),
		defaultCreationPolicy:         config.DefaultCreationPolicy,
		defaultDriftDetectionInterval: config.DefaultDriftDetectionInterval,
		harborRequestTimeout:          config.HarborRequestTimeout,
	}, nil
}

func validateDefaultCreationPolicy(policy harborv1alpha1.CreationPolicy) error {
	switch policy {
	case harborv1alpha1.CreationPolicyCreate,
		harborv1alpha1.CreationPolicyAdopt,
		harborv1alpha1.CreationPolicyCreateOrAdopt:
		return nil
	default:
		return fmt.Errorf("unsupported default creation policy %q", policy)
	}
}

// WithSecretReader returns a copy configured to bypass the manager cache when
// reading Secrets.
func (o OperatorOptions) WithSecretReader(reader client.Reader) OperatorOptions {
	o.secretReader = reader
	return o
}

func (o OperatorOptions) effectiveCreationPolicy(policy harborv1alpha1.CreationPolicy) harborv1alpha1.CreationPolicy {
	if policy != "" {
		return policy
	}
	if o.defaultCreationPolicy == "" {
		return harborv1alpha1.CreationPolicyCreate
	}
	return o.defaultCreationPolicy
}

func (o OperatorOptions) requestTimeout() time.Duration {
	if o.harborRequestTimeout == 0 {
		return defaultHarborRequestTimeout
	}
	return o.harborRequestTimeout
}
