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
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
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
	logger := log.FromContext(ctx)

	// Fetch the HarborConnection instance
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

	// Build the check URL using the BaseURL from the CR.
	checkURL := fmt.Sprintf("%s/api/systeminfo", conn.Spec.BaseURL)
	logger.Info("Checking Harbor connectivity", "url", checkURL)

	// Create an HTTP client with a timeout.
	httpClient := &http.Client{Timeout: 10 * time.Second}

	// Perform a GET request to the Harbor API endpoint.
	resp, err := httpClient.Get(checkURL)
	if err != nil {
		logger.Error(err, "Failed to connect to Harbor", "url", checkURL)
		// Optionally, update a status condition or requeue after a delay.
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}
	defer resp.Body.Close()

	// Check that we received a successful status code.
	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		logger.Error(errMsg, "Harbor API check failed", "url", checkURL)
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	logger.Info("Successfully connected to Harbor", "url", checkURL)

	// Optionally: update status fields on conn to reflect connectivity.
	// For example:
	// conn.Status.Connected = true
	// if err := r.Status().Update(ctx, &conn); err != nil {
	//     logger.Error(err, "Failed to update HarborConnection status")
	//     return ctrl.Result{}, err
	// }

	// No requeue is necessary if everything is OK.
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *HarborConnectionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&harborv1alpha1.HarborConnection{}).
		Named("harborconnection").
		Complete(r)
}
