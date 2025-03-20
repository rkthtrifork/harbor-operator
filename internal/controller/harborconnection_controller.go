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
	"fmt"
	"io"
	"net/http"
	"net/url"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
)

// HarborConnectionReconciler reconciles a HarborConnection object.
type HarborConnectionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile compares the desired state (HarborConnection object) with the actual cluster state.
func (r *HarborConnectionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithName(fmt.Sprintf("[HarborConnection:%s]", req.NamespacedName))

	// Fetch the HarborConnection instance.
	var conn harborv1alpha1.HarborConnection
	if err := r.Get(ctx, req.NamespacedName, &conn); err != nil {
		if errors.IsNotFound(err) {
			logger.Info("HarborConnection resource not found; it may have been deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get HarborConnection")
		return ctrl.Result{}, err
	}

	// Validate BaseURL.
	parsedURL, err := url.Parse(conn.Spec.BaseURL)
	if err != nil {
		logger.Error(err, "Invalid baseURL format")
		return ctrl.Result{}, err
	}
	if parsedURL.Scheme == "" {
		err := fmt.Errorf("baseURL %s is missing a protocol scheme", conn.Spec.BaseURL)
		logger.Error(err, "Invalid baseURL")
		return ctrl.Result{}, err
	}

	// Check connectivity.
	if conn.Spec.Credentials == nil {
		// Non-authenticated connectivity check using the Docker Registry API ping endpoint.
		pingURL := fmt.Sprintf("%s/api/v2.0/ping", conn.Spec.BaseURL)
		logger.Info("No credentials provided; checking connectivity using non-authorized endpoint", "url", pingURL)

		pingReq, err := http.NewRequest("GET", pingURL, nil)
		if err != nil {
			logger.Error(err, "Failed to create HTTP request for connectivity check")
			return ctrl.Result{}, err
		}

		resp, err := http.DefaultClient.Do(pingReq)
		if err != nil {
			logger.Error(err, "Failed to perform connectivity check on Harbor API")
			return ctrl.Result{}, err
		}
		defer resp.Body.Close()

		// Both 200 (OK) and 401 (Unauthorized) indicate connectivity.
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusUnauthorized {
			body, _ := io.ReadAll(resp.Body)
			err := fmt.Errorf("connectivity check failed: unexpected status code %d, body: %s", resp.StatusCode, string(body))
			logger.Error(err, "Harbor connectivity check failed")
			return ctrl.Result{}, err
		}

		logger.Info("Successfully checked connectivity on Harbor API using non-authorized endpoint")
		return ctrl.Result{}, nil
	}

	// Authenticated connectivity check.
	username := conn.Spec.Credentials.AccessKey
	secretKey := types.NamespacedName{
		Namespace: conn.Namespace,
		Name:      conn.Spec.Credentials.AccessSecretRef,
	}

	// Retrieve the secret containing the access secret.
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

	// Verify credentials via /users/current endpoint.
	authURL := fmt.Sprintf("%s/api/v2.0/users/current", conn.Spec.BaseURL)
	logger.Info("Verifying Harbor API credentials", "url", authURL)

	authReq, err := http.NewRequest("GET", authURL, nil)
	if err != nil {
		logger.Error(err, "Failed to create HTTP request for credential check")
		return ctrl.Result{}, err
	}

	authReq.SetBasicAuth(username, password)
	resp, err := http.DefaultClient.Do(authReq)
	if err != nil {
		logger.Error(err, "Failed to perform HTTP request for credential check")
		return ctrl.Result{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		err = fmt.Errorf("harbor API credential check failed with status %d, body: %s", resp.StatusCode, string(body))
		logger.Error(err, "Harbor API authentication failed")
		return ctrl.Result{}, err
	}

	logger.Info("Successfully authenticated with Harbor API")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *HarborConnectionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&harborv1alpha1.HarborConnection{}).
		Named("harborconnection").
		Complete(r)
}
