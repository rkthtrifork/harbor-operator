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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
)

type registryCredential struct {
	Type         string `json:"type"`
	AccessKey    string `json:"access_key"`
	AccessSecret string `json:"access_secret"`
}

type createRegistryRequest struct {
	Type             string             `json:"type"`
	Name             string             `json:"name"`
	Description      string             `json:"description,omitempty"`
	URL              string             `json:"url"`
	VerifyRemoteCert bool               `json:"verify_remote_cert"`
	Credential       registryCredential `json:"credential"`
}

// RegistryReconciler reconciles a Registry object
type RegistryReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=registries,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=registries/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=registries/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Registry object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.2/pkg/reconcile
func (r *RegistryReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// 1. Fetch the Registry CR
	var registry harborv1alpha1.Registry
	if err := r.Get(ctx, req.NamespacedName, &registry); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("Registry resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get Registry")
		return ctrl.Result{}, err
	}

	// 2. Retrieve the HarborConnection using the HarborConnectionRef in the Registry
	var harborConn harborv1alpha1.HarborConnection
	connRef := registry.Spec.HarborConnectionRef
	connKey := types.NamespacedName{
		Namespace: connRef.Namespace,
		Name:      connRef.Name,
	}
	if err := r.Get(ctx, connKey, &harborConn); err != nil {
		logger.Error(err, "Failed to get HarborConnection", "HarborConnectionRef", connRef)
		return ctrl.Result{}, err
	}

	reqPayload := createRegistryRequest{
		Type:             registry.Spec.Type,
		Name:             registry.Spec.Name,
		Description:      registry.Spec.Description,
		URL:              registry.Spec.URL,
		VerifyRemoteCert: registry.Spec.VerifyRemoteCert,
	}

	if harborConn.Spec.Credentials != nil && harborConn.Spec.Credentials.Type == "basic" {
		secretKey := types.NamespacedName{
			Namespace: harborConn.Namespace,
			Name:      harborConn.Spec.Credentials.AccessSecretRef,
		}
		var secret corev1.Secret
		if err := r.Get(ctx, secretKey, &secret); err != nil {
			logger.Error(err, "Failed to get secret", "Secret", secretKey)
			return ctrl.Result{}, err
		}

		accessSecretBytes, ok := secret.Data["access_secret"]
		if !ok {
			err := fmt.Errorf("access_secret not found in secret %s/%s", secretKey.Namespace, secretKey.Name)
			logger.Error(err, "Secret data missing access_secret")
			return ctrl.Result{}, err
		}
		password := string(accessSecretBytes)

		reqPayload.Credential = registryCredential{
			Type:         "basic",
			AccessKey:    harborConn.Spec.Credentials.AccessKey,
			AccessSecret: password,
		}
	}

	// Marshal the payload to JSON.
	payloadBytes, err := json.Marshal(reqPayload)
	if err != nil {
		logger.Error(err, "Failed to marshal create registry payload")
		return ctrl.Result{}, err
	}

	// 6. Create the Harbor API request URL using the base URL from HarborConnection.
	requestURL := fmt.Sprintf("%s/api/registries", harborConn.Spec.BaseURL)

	// Create a new HTTP POST request.
	httpReq, err := http.NewRequest("POST", requestURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		logger.Error(err, "Failed to create HTTP request")
		return ctrl.Result{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	// 7. Send the HTTP request using a client with a timeout.
	httpClient := &http.Client{Timeout: 10 * time.Second}
	resp, err := httpClient.Do(httpReq)
	if err != nil {
		logger.Error(err, "Failed to send HTTP request")
		return ctrl.Result{}, err
	}
	defer resp.Body.Close()

	// Check for success status code (200 OK or 201 Created).
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		errMsg := fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		logger.Error(errMsg, "Harbor API returned an error", "status", resp.StatusCode)
		return ctrl.Result{}, errMsg
	}

	logger.Info("Successfully created registry in Harbor", "Registry", registry.Name)
	// Optionally, update the status of the Registry CR here.

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RegistryReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&harborv1alpha1.Registry{}).
		Named("registry").
		Complete(r)
}
