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

// HarborConnectionReconciler reconciles a HarborConnection object
type HarborConnectionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=harbor.harbor-operator.io,resources=harborconnections/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the HarborConnection object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.20.2/pkg/reconcile
func (r *HarborConnectionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx).WithName("[HarborConnection:" + req.NamespacedName.String() + "]")

	// Fetch the HarborConnection instance.
	var conn harborv1alpha1.HarborConnection
	if err := r.Get(ctx, req.NamespacedName, &conn); err != nil {
		if errors.IsNotFound(err) {
			// HarborConnection resource not found. It might have been deleted.
			logger.Info("HarborConnection resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get HarborConnection")
		return ctrl.Result{}, err
	}

	// Parse the BaseURL to ensure it's a valid URL.
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

	// If no credentials are provided, check connectivity using a non-authorized endpoint.
	if conn.Spec.Credentials == nil {
		// Use the Docker Registry API ping endpoint.
		pingURL := fmt.Sprintf("%s/api/v2.0/ping", conn.Spec.BaseURL)
		logger.Info("No credentials provided in HarborConnection spec; checking connectivity using non-authorized endpoint", "url", pingURL)

		httpReq, err := http.NewRequest("GET", pingURL, nil)
		if err != nil {
			logger.Error(err, "Failed to create HTTP request for connectivity check")
			return ctrl.Result{}, err
		}

		client := &http.Client{}
		resp, err := client.Do(httpReq)
		if err != nil {
			logger.Error(err, "Failed to perform connectivity check on Harbor API")
			return ctrl.Result{}, err
		}
		defer resp.Body.Close()

		// For a non-authorized endpoint, a status code of 200 (OK) or 401 (Unauthorized) indicates connectivity.
		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusUnauthorized {
			body, _ := io.ReadAll(resp.Body)
			err := fmt.Errorf("connectivity check failed, unexpected status code: %d, body: %s", resp.StatusCode, string(body))
			logger.Error(err, "Harbor connectivity check failed")
			return ctrl.Result{}, err
		}
		logger.Info("Successfully checked connectivity on Harbor API using non-authorized endpoint")
		return ctrl.Result{}, nil
	}

	// If credentials are provided, proceed with authenticated login using Basic Auth.
	username := conn.Spec.Credentials.AccessKey

	// Use the AccessSecretRef to fetch the secret that contains the access secret.
	secretKey := types.NamespacedName{
		Namespace: conn.Namespace,
		Name:      conn.Spec.Credentials.AccessSecretRef,
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

	// Build the Harbor API URL for verifying credentials via /users/current.
	authURL := fmt.Sprintf("%s/api/v2.0/users/current", conn.Spec.BaseURL)
	logger.Info("Verifying Harbor API credentials via /users/current", "url", authURL)

	// Create an HTTP request for the authentication check.
	authReq, err := http.NewRequest("GET", authURL, nil)
	if err != nil {
		logger.Error(err, "Failed to create HTTP request for credential check")
		return ctrl.Result{}, err
	}

	// Set Basic Auth with the username and password.
	authReq.SetBasicAuth(username, password)

	// Perform the HTTP request.
	client := &http.Client{}
	resp, err := client.Do(authReq)
	if err != nil {
		logger.Error(err, "Failed to perform HTTP request for credential check")
		return ctrl.Result{}, err
	}
	defer resp.Body.Close()

	// Check for a successful status code (HTTP 200).
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		err = fmt.Errorf("harbor API credential check failed with status: %d, body: %s", resp.StatusCode, string(body))
		logger.Error(err, "Harbor API authentication failed")
		return ctrl.Result{}, err
	}

	logger.Info("Successfully authenticated with Harbor API using /users/current endpoint")
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *HarborConnectionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&harborv1alpha1.HarborConnection{}).
		Named("harborconnection").
		Complete(r)
}
