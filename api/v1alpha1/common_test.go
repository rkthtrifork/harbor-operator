package v1alpha1

import "testing"

func TestCreationPolicyCapabilities(t *testing.T) {
	tests := []struct {
		name           string
		policy         CreationPolicy
		allowsCreate   bool
		allowsAdoption bool
	}{
		{name: "zero value defaults to create", allowsCreate: true},
		{name: "create", policy: CreationPolicyCreate, allowsCreate: true},
		{name: "adopt", policy: CreationPolicyAdopt, allowsAdoption: true},
		{name: "create or adopt", policy: CreationPolicyCreateOrAdopt, allowsCreate: true, allowsAdoption: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.policy.AllowsCreation(); got != tt.allowsCreate {
				t.Fatalf("AllowsCreation() = %t, want %t", got, tt.allowsCreate)
			}
			if got := tt.policy.AllowsAdoption(); got != tt.allowsAdoption {
				t.Fatalf("AllowsAdoption() = %t, want %t", got, tt.allowsAdoption)
			}
		})
	}
}
