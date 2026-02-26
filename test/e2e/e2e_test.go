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
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/rkthtrifork/harbor-operator/test/utils"
)

// namespace where the project is deployed in
const namespace = "harbor-operator-system"

// serviceAccountName created for the project
const serviceAccountName = "harbor-operator-controller-manager"

// metricsServiceName is the name of the metrics service of the project
const metricsServiceName = "harbor-operator-controller-manager-metrics-service"

// metricsRoleBindingName is the name of the RBAC that will be created to allow get the metrics data
const metricsRoleBindingName = "harbor-operator-metrics-binding"

var _ = Describe("Manager", Ordered, func() {
	var controllerPodName string

	// Before running the tests, set up the environment by creating the namespace,
	// enforce the restricted security policy to the namespace, installing CRDs,
	// and deploying the controller.
	BeforeAll(func() {
		By("creating manager namespace")
		cmd := exec.Command("kubectl", "create", "ns", namespace)
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create namespace")

		By("labeling the namespace to enforce the restricted security policy")
		cmd = exec.Command("kubectl", "label", "--overwrite", "ns", namespace,
			"pod-security.kubernetes.io/enforce=restricted")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to label namespace with restricted policy")

		By("installing CRDs")
		cmd = exec.Command("make", "install")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to install CRDs")

		By("deploying the controller-manager")
		cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", projectImage))
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to deploy the controller-manager")
	})

	// After all tests have been executed, clean up by undeploying the controller, uninstalling CRDs,
	// and deleting the namespace.
	AfterAll(func() {
		By("cleaning up the curl pod for metrics")
		cmd := exec.Command("kubectl", "delete", "pod", "curl-metrics", "-n", namespace)
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
			By("Fetching controller manager pod logs")
			cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
			controllerLogs, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Controller logs:\n %s", controllerLogs)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Controller logs: %s", err)
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

			By("Fetching controller manager pod description")
			cmd = exec.Command("kubectl", "describe", "pod", controllerPodName, "-n", namespace)
			podDescription, err := utils.Run(cmd)
			if err == nil {
				fmt.Println("Pod description:\n", podDescription)
			} else {
				fmt.Println("Failed to describe controller pod")
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
					"pods", "-l", "control-plane=controller-manager",
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
				g.Expect(controllerPodName).To(ContainSubstring("controller-manager"))

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
			cmd := exec.Command("kubectl", "create", "clusterrolebinding", metricsRoleBindingName,
				"--clusterrole=harbor-operator-metrics-reader",
				fmt.Sprintf("--serviceaccount=%s:%s", namespace, serviceAccountName),
			)
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
			Expect(metricsOutput).To(ContainSubstring(
				"controller_runtime_reconcile_total",
			))
		})

		Context("Harbor Flow", func() {
			It("should reconcile Harbor resources end-to-end", func() {
				baseURL := os.Getenv("HARBOR_BASE_URL")
				adminUser := os.Getenv("HARBOR_ADMIN_USER")
				adminPass := os.Getenv("HARBOR_ADMIN_PASSWORD")
				if baseURL == "" || adminUser == "" || adminPass == "" {
					Skip("HARBOR_BASE_URL, HARBOR_ADMIN_USER, and HARBOR_ADMIN_PASSWORD must be set to run Harbor flow")
				}

				suffix := fmt.Sprintf("%d", time.Now().UnixNano())
				registryName := "e2e-registry-" + suffix
				projectName := "e2e-project-" + suffix
				retentionName := "e2e-retention-" + suffix
				userName := "e2e-user-" + suffix
				secretName := "harbor-e2e-pass-" + suffix
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
  harborConnectionRef: %s
  type: docker-registry
  name: %s
  url: https://registry-1.docker.io
  insecure: false
---
apiVersion: harbor.harbor-operator.io/v1alpha1
kind: Project
metadata:
  name: %s
  namespace: %s
spec:
  harborConnectionRef: %s
  public: true
  metadata:
    public: "true"
  registryName: %s
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
  harborConnectionRef: %s
  username: %s
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
  harborConnectionRef: %s
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
`, secretName, namespace, adminPass, connName, namespace, baseURL, adminUser, secretName,
					registryName, namespace, connName, registryName,
					projectName, namespace, connName, registryName,
					secretName, namespace, userName, namespace, connName, userName, userName, secretName,
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
				Expect(harborHasObject(baseURL, adminUser, adminPass, "/api/v2.0/registries", fmt.Sprintf(`\"name\":\"%s\"`, registryName))).To(BeTrue())
				Expect(harborHasObject(baseURL, adminUser, adminPass, "/api/v2.0/projects", fmt.Sprintf(`\"name\":\"%s\"`, projectName))).To(BeTrue())
				Expect(harborHasObject(baseURL, adminUser, adminPass, "/api/v2.0/users", fmt.Sprintf(`\"username\":\"%s\"`, userName))).To(BeTrue())

				By("deleting Harbor CRs in dependency order")
				deleteCR("retentionpolicy", retentionName)
				deleteCR("project", projectName)
				deleteCR("registry", registryName)
				deleteCR("user", userName)
				deleteCR("harborconnection", connName)
				deleteCR("secret", secretName)

				By("verifying Harbor objects are gone")
				Eventually(func() bool {
					return !harborHasObject(baseURL, adminUser, adminPass, "/api/v2.0/registries", fmt.Sprintf(`\"name\":\"%s\"`, registryName))
				}, 2*time.Minute, 5*time.Second).Should(BeTrue())
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

func harborHasObject(baseURL, user, pass, path, needle string) bool {
	cmd := exec.Command("curl", "-sk", "-u", fmt.Sprintf("%s:%s", user, pass), baseURL+path)
	out, err := utils.Run(cmd)
	if err != nil {
		return false
	}
	return strings.Contains(out, needle)
}

// tokenRequest is a simplified representation of the Kubernetes TokenRequest API response,
// containing only the token field that we need to extract.
type tokenRequest struct {
	Status struct {
		Token string `json:"token"`
	} `json:"status"`
}
