//go:build e2e
// +build e2e

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

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/rkthtrifork/harbor-operator/test/utils"
)

// namespace where the project is deployed in
const namespace = "harbor-operator-system"

// serviceAccountName created for the project
const serviceAccountName = "harbor-operator"

// metricsServiceName is the name of the metrics service of the project
const metricsServiceName = "harbor-operator-metrics"

// metricsRoleBindingName is the name of the RBAC that will be created to allow get the metrics data
const metricsRoleBindingName = "harbor-operator-metrics-binding"
const metricsRoleName = "harbor-operator-metrics-reader"

var _ = Describe("Manager", Ordered, func() {
	var controllerPodName string

	// Before running the tests, set up the environment by creating the namespace,
	// enforce the restricted security policy to the namespace, installing CRDs,
	// and deploying the controller.
	BeforeAll(func() {
		By("creating manager namespace")
		cmd := exec.Command("bash", "-lc", fmt.Sprintf("kubectl create namespace %s --dry-run=client -o yaml | kubectl apply -f -", namespace))
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create namespace")

		By("labeling the namespace to enforce the restricted security policy")
		cmd = exec.Command("kubectl", "label", "--overwrite", "ns", namespace,
			"pod-security.kubernetes.io/enforce=restricted")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to label namespace with restricted policy")

		By("deploying the controller-manager")
		cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", projectImage))
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to deploy the controller-manager")

		By("enabling metrics for the Helm deployment")
		cmd = exec.Command("helm", "upgrade", "--install", "harbor-operator", "./charts/harbor-operator",
			"--namespace", namespace,
			"--set", fmt.Sprintf("image.repository=%s", strings.Split(projectImage, ":")[0]),
			"--set", fmt.Sprintf("image.tag=%s", strings.Split(projectImage, ":")[1]),
			"--set", "metrics.enabled=true",
			"--wait",
		)
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to enable metrics on the controller-manager deployment")
	})

	// After all tests have been executed, clean up by undeploying the controller, uninstalling CRDs,
	// and deleting the namespace.
	AfterAll(func() {
		By("cleaning up the curl pod for metrics")
		cmd := exec.Command("kubectl", "delete", "pod", "curl-metrics", "-n", namespace)
		_, _ = utils.Run(cmd)
		cmd = exec.Command("kubectl", "delete", "clusterrolebinding", metricsRoleBindingName)
		_, _ = utils.Run(cmd)
		cmd = exec.Command("kubectl", "delete", "clusterrole", metricsRoleName)
		_, _ = utils.Run(cmd)

		By("undeploying the controller-manager")
		cmd = exec.Command("make", "undeploy")
		_, _ = utils.Run(cmd)

		By("uninstalling CRDs")
		cmd = exec.Command("make", "uninstall")
		_, _ = utils.Run(cmd)

		By("removing manager namespace")
		cmd = exec.Command("kubectl", "delete", "ns", namespace)
		_, _ = utils.Run(cmd)
	})

	// After each test, check for failures and collect logs, events,
	// and pod descriptions for debugging.
	AfterEach(func() {
		specReport := CurrentSpecReport()
		if specReport.Failed() {
			var cmd *exec.Cmd
			if controllerPodName != "" {
				By("Fetching controller manager pod logs")
				cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
				controllerLogs, err := utils.Run(cmd)
				if err == nil {
					_, _ = fmt.Fprintf(GinkgoWriter, "Controller logs:\n %s", controllerLogs)
				} else {
					_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Controller logs: %s", err)
				}
			}

			By("Fetching Kubernetes events")
			cmd = exec.Command("kubectl", "get", "events", "-n", namespace, "--sort-by=.lastTimestamp")
			eventsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Kubernetes events:\n%s", eventsOutput)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Kubernetes events: %s", err)
			}

			By("Fetching curl-metrics logs")
			cmd = exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
			metricsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Metrics logs:\n %s", metricsOutput)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get curl-metrics logs: %s", err)
			}

			if controllerPodName != "" {
				By("Fetching controller manager pod description")
				cmd = exec.Command("kubectl", "describe", "pod", controllerPodName, "-n", namespace)
				podDescription, err := utils.Run(cmd)
				if err == nil {
					fmt.Println("Pod description:\n", podDescription)
				} else {
					fmt.Println("Failed to describe controller pod")
				}
			}
		}
	})

	SetDefaultEventuallyTimeout(2 * time.Minute)
	SetDefaultEventuallyPollingInterval(time.Second)

	Context("Manager", func() {
		It("should run successfully", func() {
			By("validating that the controller-manager pod is running as expected")
			verifyControllerUp := func(g Gomega) {
				// Get the name of the controller-manager pod
				cmd := exec.Command("kubectl", "get",
					"pods", "-l", "app.kubernetes.io/name=harbor-operator,app.kubernetes.io/instance=harbor-operator",
					"-o", "go-template={{ range .items }}"+
						"{{ if not .metadata.deletionTimestamp }}"+
						"{{ .metadata.name }}"+
						"{{ \"\\n\" }}{{ end }}{{ end }}",
					"-n", namespace,
				)

				podOutput, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to retrieve controller-manager pod information")
				podNames := utils.GetNonEmptyLines(podOutput)
				g.Expect(podNames).To(HaveLen(1), "expected 1 controller pod running")
				controllerPodName = podNames[0]
				g.Expect(controllerPodName).To(ContainSubstring("harbor-operator"))

				// Validate the pod's status
				cmd = exec.Command("kubectl", "get",
					"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
					"-n", namespace,
				)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Running"), "Incorrect controller-manager pod status")
			}
			Eventually(verifyControllerUp).Should(Succeed())
		})

		It("should ensure the metrics endpoint is serving metrics", func() {
			By("creating a ClusterRoleBinding for the service account to allow access to metrics")
			roleFile := writeTempFile(fmt.Sprintf(`apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: %s
rules:
- nonResourceURLs:
  - /metrics
  verbs:
  - get
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: %s
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: %s
subjects:
- kind: ServiceAccount
  name: %s
  namespace: %s
`, metricsRoleName, metricsRoleBindingName, metricsRoleName, serviceAccountName, namespace))
			defer os.Remove(roleFile)
			cmd := exec.Command("kubectl", "apply", "-f", roleFile)
			_, err := utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create ClusterRoleBinding")

			By("validating that the metrics service is available")
			cmd = exec.Command("kubectl", "get", "service", metricsServiceName, "-n", namespace)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Metrics service should exist")

			By("getting the service account token")
			token, err := serviceAccountToken()
			Expect(err).NotTo(HaveOccurred())
			Expect(token).NotTo(BeEmpty())

			By("waiting for the metrics endpoint to be ready")
			verifyMetricsEndpointReady := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "endpoints", metricsServiceName, "-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("8443"), "Metrics endpoint is not ready")
			}
			Eventually(verifyMetricsEndpointReady).Should(Succeed())

			By("verifying that the controller manager is serving the metrics server")
			verifyMetricsServerStarted := func(g Gomega) {
				cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("controller-runtime.metrics\tServing metrics server"),
					"Metrics server not yet started")
			}
			Eventually(verifyMetricsServerStarted).Should(Succeed())

			By("creating the curl-metrics pod to access the metrics endpoint")
			cmd = exec.Command("kubectl", "run", "curl-metrics", "--restart=Never",
				"--namespace", namespace,
				"--image=curlimages/curl:latest",
				"--overrides",
				fmt.Sprintf(`{
					"spec": {
						"containers": [{
							"name": "curl",
							"image": "curlimages/curl:latest",
							"command": ["/bin/sh", "-c"],
							"args": ["curl -v -k -H 'Authorization: Bearer %s' https://%s.%s.svc.cluster.local:8443/metrics"],
							"securityContext": {
								"allowPrivilegeEscalation": false,
								"capabilities": {
									"drop": ["ALL"]
								},
								"runAsNonRoot": true,
								"runAsUser": 1000,
								"seccompProfile": {
									"type": "RuntimeDefault"
								}
							}
						}],
						"serviceAccount": "%s"
					}
				}`, token, metricsServiceName, namespace, serviceAccountName))
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create curl-metrics pod")

			By("waiting for the curl-metrics pod to complete.")
			verifyCurlUp := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "pods", "curl-metrics",
					"-o", "jsonpath={.status.phase}",
					"-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Succeeded"), "curl pod in wrong status")
			}
			Eventually(verifyCurlUp, 5*time.Minute).Should(Succeed())

			By("getting the metrics by checking curl-metrics logs")
			metricsOutput := getMetricsOutput()
			Expect(metricsOutput).To(ContainSubstring("process_cpu_seconds_total"))
		})

		Context("Harbor Flow", func() {
			It("should reconcile Harbor resources end-to-end", func() {
				baseURL := os.Getenv("HARBOR_BASE_URL")
				inClusterBaseURL := os.Getenv("HARBOR_IN_CLUSTER_BASE_URL")
				adminUser := os.Getenv("HARBOR_ADMIN_USER")
				adminPass := os.Getenv("HARBOR_ADMIN_PASSWORD")
				if baseURL == "" || adminUser == "" || adminPass == "" {
					Skip("HARBOR_BASE_URL, HARBOR_ADMIN_USER, and HARBOR_ADMIN_PASSWORD must be set to run Harbor flow")
				}
				if inClusterBaseURL == "" {
					inClusterBaseURL = baseURL
				}

				suffix := fmt.Sprintf("%d", time.Now().UnixNano())
				registryName := "e2e-registry-" + suffix
				projectName := "e2e-project-" + suffix
				retentionName := "e2e-retention-" + suffix
				userName := "e2e-user-" + suffix
				adminSecretName := "harbor-e2e-admin-pass-" + suffix
				userSecretName := "harbor-e2e-user-pass-" + suffix
				connName := "harbor-e2e-conn-" + suffix

				crs := fmt.Sprintf(`---
apiVersion: v1
kind: Secret
metadata:
  name: %s
  namespace: %s
type: Opaque
stringData:
  password: "%s"
---
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: HarborConnection
metadata:
  name: %s
  namespace: %s
spec:
  baseURL: %s
  credentials:
    type: basic
    username: %s
    passwordSecretRef:
      name: %s
      key: password
---
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: Registry
metadata:
  name: %s
  namespace: %s
spec:
  harborConnectionRef:
    name: %s
    kind: HarborConnection
  type: docker-registry
  url: https://registry-1.docker.io
  insecure: false
---
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: Project
metadata:
  name: %s
  namespace: %s
spec:
  harborConnectionRef:
    name: %s
    kind: HarborConnection
  public: true
  metadata:
    public: "true"
  registryRef:
    name: %s
---
apiVersion: v1
kind: Secret
metadata:
  name: %s
  namespace: %s
type: Opaque
stringData:
  password: "ChangeMe-123!"
---
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: User
metadata:
  name: %s
  namespace: %s
spec:
  harborConnectionRef:
    name: %s
    kind: HarborConnection
  email: %s@example.com
  realname: e2e user
  passwordSecretRef:
    name: %s
    key: password
---
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: RetentionPolicy
metadata:
  name: %s
  namespace: %s
spec:
  harborConnectionRef:
    name: %s
    kind: HarborConnection
  projectRef:
    name: %s
  algorithm: or
  trigger:
    kind: Schedule
    settings:
      cron: 0 0 0 * * *
  rules:
    - action: retain
      template: latestPushedK
      params:
        latestPushedK:
          value: 3
      tagSelectors:
        - kind: doublestar
          decoration: matches
          pattern: "**"
      scopeSelectors:
        repository:
          - kind: doublestar
            decoration: repoMatches
            pattern: "**"
`, adminSecretName, namespace, adminPass, connName, namespace, inClusterBaseURL, adminUser, adminSecretName,
					registryName, namespace, connName,
					projectName, namespace, connName, registryName,
					userSecretName, namespace, userName, namespace, connName, userName, userSecretName,
					retentionName, namespace, connName, projectName)

				applyFile := writeTempFile(crs)
				DeferCleanup(func() {
					_ = os.Remove(applyFile)
				})

				By("applying Harbor CRs")
				cmd := exec.Command("kubectl", "apply", "-f", applyFile)
				_, err := utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred())

				By("waiting for CRs to be Ready")
				waitReady("harborconnection", connName)
				waitReady("registry", registryName)
				waitReady("project", projectName)
				waitReady("user", userName)
				waitReady("retentionpolicy", retentionName)

				By("verifying Harbor objects via API")
				Expect(harborHasObject(baseURL, adminUser, adminPass, "/api/v2.0/registries", fmt.Sprintf(`"name":"%s"`, registryName))).To(BeTrue())
				Expect(harborHasObject(baseURL, adminUser, adminPass, "/api/v2.0/projects", fmt.Sprintf(`"name":"%s"`, projectName))).To(BeTrue())
				Expect(harborHasObject(baseURL, adminUser, adminPass, "/api/v2.0/users", fmt.Sprintf(`"username":"%s"`, userName))).To(BeTrue())

				By("deleting Harbor CRs in dependency order")
				deleteCR("retentionpolicy", retentionName)
				deleteCR("project", projectName)
				deleteCR("registry", registryName)
				deleteCR("user", userName)
				deleteCR("harborconnection", connName)
				deleteCR("secret", userSecretName)
				deleteCR("secret", adminSecretName)

				By("verifying Harbor objects are gone")
				Eventually(func() bool {
					return !harborHasObject(baseURL, adminUser, adminPass, "/api/v2.0/registries", fmt.Sprintf(`"name":"%s"`, registryName))
				}, 2*time.Minute, 5*time.Second).Should(BeTrue())
			})

			It("should reconcile resources through a ClusterHarborConnection", func() {
				baseURL := os.Getenv("HARBOR_BASE_URL")
				inClusterBaseURL := os.Getenv("HARBOR_IN_CLUSTER_BASE_URL")
				adminUser := os.Getenv("HARBOR_ADMIN_USER")
				adminPass := os.Getenv("HARBOR_ADMIN_PASSWORD")
				if baseURL == "" || adminUser == "" || adminPass == "" {
					Skip("HARBOR_BASE_URL, HARBOR_ADMIN_USER, and HARBOR_ADMIN_PASSWORD must be set to run Harbor flow")
				}
				if inClusterBaseURL == "" {
					inClusterBaseURL = baseURL
				}

				suffix := fmt.Sprintf("%d", time.Now().UnixNano())
				secretName := "harbor-e2e-cluster-pass-" + suffix
				connName := "harbor-e2e-cluster-conn-" + suffix
				projectName := "e2e-cluster-project-" + suffix

				crs := fmt.Sprintf(`---
apiVersion: v1
kind: Secret
metadata:
  name: %s
  namespace: %s
type: Opaque
stringData:
  password: "%s"
---
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: ClusterHarborConnection
metadata:
  name: %s
spec:
  baseURL: %s
  credentials:
    type: basic
    username: %s
    passwordSecretRef:
      name: %s
      namespace: %s
      key: password
---
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: Project
metadata:
  name: %s
  namespace: %s
spec:
  harborConnectionRef:
    name: %s
    kind: ClusterHarborConnection
  public: false
`, secretName, namespace, adminPass, connName, inClusterBaseURL, adminUser, secretName, namespace, projectName, namespace, connName)

				applyFile := writeTempFile(crs)
				DeferCleanup(func() {
					_ = os.Remove(applyFile)
				})

				By("applying cluster-scoped Harbor connection resources")
				cmd := exec.Command("kubectl", "apply", "-f", applyFile)
				_, err := utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred())

				waitReady("clusterharborconnection", connName)
				waitReady("project", projectName)
				Expect(harborHasObject(baseURL, adminUser, adminPass, "/api/v2.0/projects", fmt.Sprintf(`"name":"%s"`, projectName))).To(BeTrue())

				deleteCR("project", projectName)
				cmd = exec.Command("kubectl", "delete", "clusterharborconnection.harbor.harbor-operator.io/"+connName, "--wait=true")
				_, _ = utils.Run(cmd)
				cmd = exec.Command("kubectl", "delete", "secret", secretName, "-n", namespace, "--wait=true")
				_, _ = utils.Run(cmd)
			})

			It("should honor deletionPolicy Orphan when the Harbor connection is deleted first", func() {
				baseURL := os.Getenv("HARBOR_BASE_URL")
				inClusterBaseURL := os.Getenv("HARBOR_IN_CLUSTER_BASE_URL")
				adminUser := os.Getenv("HARBOR_ADMIN_USER")
				adminPass := os.Getenv("HARBOR_ADMIN_PASSWORD")
				if baseURL == "" || adminUser == "" || adminPass == "" {
					Skip("HARBOR_BASE_URL, HARBOR_ADMIN_USER, and HARBOR_ADMIN_PASSWORD must be set to run Harbor flow")
				}
				if inClusterBaseURL == "" {
					inClusterBaseURL = baseURL
				}

				suffix := fmt.Sprintf("%d", time.Now().UnixNano())
				secretName := "harbor-e2e-orphan-pass-" + suffix
				connName := "harbor-e2e-orphan-conn-" + suffix
				projectName := "e2e-orphan-project-" + suffix

				crs := fmt.Sprintf(`---
apiVersion: v1
kind: Secret
metadata:
  name: %s
  namespace: %s
type: Opaque
stringData:
  password: "%s"
---
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: HarborConnection
metadata:
  name: %s
  namespace: %s
spec:
  baseURL: %s
  credentials:
    type: basic
    username: %s
    passwordSecretRef:
      name: %s
      key: password
---
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: Project
metadata:
  name: %s
  namespace: %s
spec:
  harborConnectionRef:
    name: %s
    kind: HarborConnection
  deletionPolicy: Orphan
  public: false
`, secretName, namespace, adminPass, connName, namespace, inClusterBaseURL, adminUser, secretName, projectName, namespace, connName)

				applyFile := writeTempFile(crs)
				DeferCleanup(func() {
					_ = os.Remove(applyFile)
				})

				cmd := exec.Command("kubectl", "apply", "-f", applyFile)
				_, err := utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred())

				waitReady("harborconnection", connName)
				waitReady("project", projectName)
				Expect(harborHasObject(baseURL, adminUser, adminPass, "/api/v2.0/projects", fmt.Sprintf(`"name":"%s"`, projectName))).To(BeTrue())

				deleteCR("harborconnection", connName)
				deleteCR("project", projectName)
				waitDeleted("project", projectName)

				Expect(harborHasObject(baseURL, adminUser, adminPass, "/api/v2.0/projects", fmt.Sprintf(`"name":"%s"`, projectName))).To(BeTrue())

				deleteHarborProjectByName(baseURL, adminUser, adminPass, projectName)
				cmd = exec.Command("kubectl", "delete", "secret", secretName, "-n", namespace, "--wait=true")
				_, _ = utils.Run(cmd)
			})

			It("should report a conflict for singleton resources targeting the same Harbor instance", func() {
				baseURL := os.Getenv("HARBOR_BASE_URL")
				inClusterBaseURL := os.Getenv("HARBOR_IN_CLUSTER_BASE_URL")
				adminUser := os.Getenv("HARBOR_ADMIN_USER")
				adminPass := os.Getenv("HARBOR_ADMIN_PASSWORD")
				if baseURL == "" || adminUser == "" || adminPass == "" {
					Skip("HARBOR_BASE_URL, HARBOR_ADMIN_USER, and HARBOR_ADMIN_PASSWORD must be set to run Harbor flow")
				}
				if inClusterBaseURL == "" {
					inClusterBaseURL = baseURL
				}

				suffix := fmt.Sprintf("%d", time.Now().UnixNano())
				secretName := "harbor-e2e-singleton-pass-" + suffix
				connA := "harbor-e2e-singleton-conn-a-" + suffix
				connB := "harbor-e2e-singleton-conn-b-" + suffix
				cfgA := "e2e-config-a-" + suffix
				cfgB := "e2e-config-b-" + suffix

				crs := fmt.Sprintf(`---
apiVersion: v1
kind: Secret
metadata:
  name: %s
  namespace: %s
type: Opaque
stringData:
  password: "%s"
---
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: HarborConnection
metadata:
  name: %s
  namespace: %s
spec:
  baseURL: %s
  credentials:
    type: basic
    username: %s
    passwordSecretRef:
      name: %s
      key: password
---
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: HarborConnection
metadata:
  name: %s
  namespace: %s
spec:
  baseURL: %s
  credentials:
    type: basic
    username: %s
    passwordSecretRef:
      name: %s
      key: password
---
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: Configuration
metadata:
  name: %s
  namespace: %s
spec:
  harborConnectionRef:
    name: %s
    kind: HarborConnection
  settings:
    robot_token_duration: 44
---
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: Configuration
metadata:
  name: %s
  namespace: %s
spec:
  harborConnectionRef:
    name: %s
    kind: HarborConnection
  settings:
    robot_token_duration: 66
`, secretName, namespace, adminPass,
					connA, namespace, inClusterBaseURL, adminUser, secretName,
					connB, namespace, inClusterBaseURL, adminUser, secretName,
					cfgA, namespace, connA,
					cfgB, namespace, connB)

				applyFile := writeTempFile(crs)
				DeferCleanup(func() {
					_ = os.Remove(applyFile)
				})

				cmd := exec.Command("kubectl", "apply", "-f", applyFile)
				_, err := utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred())

				waitReady("harborconnection", connA)
				waitReady("harborconnection", connB)
				waitReady("configuration", cfgA)
				waitConditionReason("configuration", cfgB, "Ready", "ReconcileError")

				message := getConditionMessage("configuration", cfgB, "Ready")
				Expect(message).To(ContainSubstring("conflicts with existing owner"))

				deleteCR("configuration", cfgB)
				deleteCR("configuration", cfgA)
				deleteCR("harborconnection", connB)
				deleteCR("harborconnection", connA)
				cmd = exec.Command("kubectl", "delete", "secret", secretName, "-n", namespace, "--wait=true")
				_, _ = utils.Run(cmd)
			})

			It("should reconcile dependents when the backing HarborConnection changes", func() {
				baseURL := os.Getenv("HARBOR_BASE_URL")
				inClusterBaseURL := os.Getenv("HARBOR_IN_CLUSTER_BASE_URL")
				adminUser := os.Getenv("HARBOR_ADMIN_USER")
				adminPass := os.Getenv("HARBOR_ADMIN_PASSWORD")
				if baseURL == "" || adminUser == "" || adminPass == "" {
					Skip("HARBOR_BASE_URL, HARBOR_ADMIN_USER, and HARBOR_ADMIN_PASSWORD must be set to run Harbor flow")
				}
				if inClusterBaseURL == "" {
					inClusterBaseURL = baseURL
				}

				suffix := fmt.Sprintf("%d", time.Now().UnixNano())
				secretName := "harbor-e2e-connwatch-pass-" + suffix
				connName := "harbor-e2e-connwatch-conn-" + suffix
				projectName := "e2e-connwatch-project-" + suffix

				crs := fmt.Sprintf(`---
apiVersion: v1
kind: Secret
metadata:
  name: %s
  namespace: %s
type: Opaque
stringData:
  password: "%s"
---
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: HarborConnection
metadata:
  name: %s
  namespace: %s
spec:
  baseURL: %s
  credentials:
    type: basic
    username: %s
    passwordSecretRef:
      name: %s
      key: password
---
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: Project
metadata:
  name: %s
  namespace: %s
spec:
  harborConnectionRef:
    name: %s
    kind: HarborConnection
  public: false
`, secretName, namespace, adminPass, connName, namespace, inClusterBaseURL, adminUser, secretName, projectName, namespace, connName)

				applyFile := writeTempFile(crs)
				DeferCleanup(func() { _ = os.Remove(applyFile) })
				cmd := exec.Command("kubectl", "apply", "-f", applyFile)
				_, err := utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred())

				waitReady("harborconnection", connName)
				waitReady("project", projectName)

				cmd = exec.Command("kubectl", "patch", "harborconnection.harbor.harbor-operator.io/"+connName,
					"-n", namespace, "--type=merge", "-p", `{"spec":{"baseURL":"http://does-not-resolve.invalid"}}`)
				_, err = utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred())

				waitConditionReason("project", projectName, "Ready", "ReconcileError")

				cmd = exec.Command("kubectl", "patch", "harborconnection.harbor.harbor-operator.io/"+connName,
					"-n", namespace, "--type=merge", "-p", fmt.Sprintf(`{"spec":{"baseURL":%q}}`, inClusterBaseURL))
				_, err = utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred())

				waitReady("project", projectName)

				deleteCR("project", projectName)
				deleteCR("harborconnection", connName)
				cmd = exec.Command("kubectl", "delete", "secret", secretName, "-n", namespace, "--wait=true")
				_, _ = utils.Run(cmd)
			})
		})

		// +kubebuilder:scaffold:e2e-webhooks-checks
	})
})

// serviceAccountToken returns a token for the specified service account in the given namespace.
// It uses the Kubernetes TokenRequest API to generate a token by directly sending a request
// and parsing the resulting token from the API response.
func serviceAccountToken() (string, error) {
	const tokenRequestRawString = `{
		"apiVersion": "authentication.k8s.io/v1",
		"kind": "TokenRequest"
	}`

	// Temporary file to store the token request
	secretName := fmt.Sprintf("%s-token-request", serviceAccountName)
	tokenRequestFile := filepath.Join("/tmp", secretName)
	err := os.WriteFile(tokenRequestFile, []byte(tokenRequestRawString), os.FileMode(0o644))
	if err != nil {
		return "", err
	}

	var out string
	verifyTokenCreation := func(g Gomega) {
		// Execute kubectl command to create the token
		cmd := exec.Command("kubectl", "create", "--raw", fmt.Sprintf(
			"/api/v1/namespaces/%s/serviceaccounts/%s/token",
			namespace,
			serviceAccountName,
		), "-f", tokenRequestFile)

		output, err := cmd.CombinedOutput()
		g.Expect(err).NotTo(HaveOccurred())

		// Parse the JSON output to extract the token
		var token tokenRequest
		err = json.Unmarshal(output, &token)
		g.Expect(err).NotTo(HaveOccurred())

		out = token.Status.Token
	}
	Eventually(verifyTokenCreation).Should(Succeed())

	return out, err
}

// getMetricsOutput retrieves and returns the logs from the curl pod used to access the metrics endpoint.
func getMetricsOutput() string {
	By("getting the curl-metrics logs")
	cmd := exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
	metricsOutput, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to retrieve logs from curl pod")
	Expect(metricsOutput).To(ContainSubstring("< HTTP/1.1 200 OK"))
	return metricsOutput
}

func writeTempFile(contents string) string {
	file, err := os.CreateTemp("", "harbor-e2e-*.yaml")
	Expect(err).NotTo(HaveOccurred())
	_, err = file.WriteString(contents)
	Expect(err).NotTo(HaveOccurred())
	Expect(file.Close()).To(Succeed())
	return file.Name()
}

func waitReady(kind, name string) {
	cmd := exec.Command("kubectl", "wait", fmt.Sprintf("%s.harbor.harbor-operator.io/%s", kind, name),
		"-n", namespace, "--for=condition=Ready", "--timeout=2m")
	_, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred())
}

func deleteCR(kind, name string) {
	cmd := exec.Command("kubectl", "delete", fmt.Sprintf("%s.harbor.harbor-operator.io/%s", kind, name),
		"-n", namespace, "--wait=true")
	_, _ = utils.Run(cmd)
}

func waitDeleted(kind, name string) {
	Eventually(func() bool {
		cmd := exec.Command("kubectl", "get", fmt.Sprintf("%s.harbor.harbor-operator.io/%s", kind, name), "-n", namespace)
		_, err := utils.Run(cmd)
		return err != nil
	}, 2*time.Minute, 2*time.Second).Should(BeTrue())
}

func waitConditionReason(kind, name, conditionType, reason string) {
	Eventually(func() string {
		return getConditionReason(kind, name, conditionType)
	}, 2*time.Minute, 2*time.Second).Should(Equal(reason))
}

func getConditionReason(kind, name, conditionType string) string {
	cmd := exec.Command("kubectl", "get", fmt.Sprintf("%s.harbor.harbor-operator.io/%s", kind, name),
		"-n", namespace, "-o", fmt.Sprintf("jsonpath={.status.conditions[?(@.type==\"%s\")].reason}", conditionType))
	out, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred())
	return strings.TrimSpace(out)
}

func getConditionMessage(kind, name, conditionType string) string {
	cmd := exec.Command("kubectl", "get", fmt.Sprintf("%s.harbor.harbor-operator.io/%s", kind, name),
		"-n", namespace, "-o", fmt.Sprintf("jsonpath={.status.conditions[?(@.type==\"%s\")].message}", conditionType))
	out, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred())
	return strings.TrimSpace(out)
}

func harborHasObject(baseURL, user, pass, path, needle string) bool {
	cmd := exec.Command("curl", "-sk", "-u", fmt.Sprintf("%s:%s", user, pass), baseURL+path)
	out, err := utils.Run(cmd)
	if err != nil {
		return false
	}

	key, want, ok := parseNeedle(needle)
	var body any
	if ok && json.Unmarshal([]byte(out), &body) == nil && jsonContainsValue(body, key, want) {
		return true
	}

	var compact bytes.Buffer
	if json.Compact(&compact, []byte(out)) == nil {
		out = compact.String()
	}
	return strings.Contains(out, needle)
}

func parseNeedle(needle string) (string, string, bool) {
	parts := strings.SplitN(needle, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}

	key := strings.Trim(parts[0], "\" ")
	want := strings.Trim(parts[1], "\" ")
	return key, want, key != "" && want != ""
}

func jsonContainsValue(value any, key, want string) bool {
	switch typed := value.(type) {
	case []any:
		for _, item := range typed {
			if jsonContainsValue(item, key, want) {
				return true
			}
		}
	case map[string]any:
		if got, ok := typed[key]; ok && fmt.Sprint(got) == want {
			return true
		}
		for _, item := range typed {
			if jsonContainsValue(item, key, want) {
				return true
			}
		}
	}

	return false
}

func deleteHarborProjectByName(baseURL, user, pass, projectName string) {
	cmd := exec.Command("curl", "-sk", "-u", fmt.Sprintf("%s:%s", user, pass), baseURL+"/api/v2.0/projects?name="+projectName)
	out, err := utils.Run(cmd)
	if err != nil {
		return
	}

	var projects []struct {
		ProjectID int `json:"project_id"`
	}
	if err := json.Unmarshal([]byte(out), &projects); err != nil {
		return
	}
	for _, project := range projects {
		cmd = exec.Command("curl", "-sk", "-u", fmt.Sprintf("%s:%s", user, pass), "-X", "DELETE", baseURL+"/api/v2.0/projects/"+strconv.Itoa(project.ProjectID))
		_, _ = utils.Run(cmd)
	}
}

// tokenRequest is a simplified representation of the Kubernetes TokenRequest API response,
// containing only the token field that we need to extract.
type tokenRequest struct {
	Status struct {
		Token string `json:"token"`
	} `json:"status"`
}
