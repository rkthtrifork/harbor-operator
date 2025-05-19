// Copyright 2025 The Harbor-Operator Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	harborv1alpha1 "github.com/rkthtrifork/harbor-operator/api/v1alpha1"
)

// -----------------------------------------------------------------------------
// Shared controller helpers
// -----------------------------------------------------------------------------

// finalizerName is reused across every Harbor-operator owned resource.
const finalizerName = "harbor.operator/finalizer"

// returnWithDriftDetection converts the optional DriftDetectionInterval field
// into the appropriate controller-runtime Result.
func returnWithDriftDetection(base *harborv1alpha1.HarborSpecBase) (ctrl.Result, error) {
	if base == nil || base.DriftDetectionInterval == nil {
		return ctrl.Result{}, nil
	}
	if d := base.DriftDetectionInterval.Duration; d > 0 {
		return ctrl.Result{RequeueAfter: d}, nil
	}
	return ctrl.Result{}, nil
}

// getHarborConnection resolves the HarborConnection reference in the same
// namespace as the owning custom resource.
func getHarborConnection(
	ctx context.Context,
	c client.Client,
	namespace, name string,
) (*harborv1alpha1.HarborConnection, error) {

	key := types.NamespacedName{Namespace: namespace, Name: name}
	var conn harborv1alpha1.HarborConnection
	if err := c.Get(ctx, key, &conn); err != nil {
		return nil, err
	}
	return &conn, nil
}

// getHarborAuth extracts <username,password> from the HarborConnection,
// honouring a cross-namespace Secret reference when provided.
func getHarborAuth(
	ctx context.Context,
	c client.Client,
	conn *harborv1alpha1.HarborConnection,
) (string, string, error) {

	if conn.Spec.Credentials == nil {
		return "", "", nil
	}

	user := conn.Spec.Credentials.AccessKey

	// Resolve Secret namespace (default to HarborConnection.Namespace).
	ns := conn.Namespace
	if conn.Spec.Credentials.AccessSecretRef.Namespace != "" {
		ns = conn.Spec.Credentials.AccessSecretRef.Namespace
	}

	secretKey := types.NamespacedName{
		Namespace: ns,
		Name:      conn.Spec.Credentials.AccessSecretRef.Name,
	}
	var secret corev1.Secret
	if err := c.Get(ctx, secretKey, &secret); err != nil {
		return "", "", err
	}

	dataKey := conn.Spec.Credentials.AccessSecretRef.Key
	if dataKey == "" {
		dataKey = "access_secret"
	}

	value, ok := secret.Data[dataKey]
	if !ok {
		return "", "", fmt.Errorf("key %q not found in secret %s/%s", dataKey, secret.Namespace, secret.Name)
	}

	return user, string(value), nil
}

// nowUTC is a tiny helper to keep tests deterministic.
func nowUTC() metav1.Time {
	return metav1.NewTime(time.Now().UTC())
}
