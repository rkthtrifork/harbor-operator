# harbor-operator

harbor-operator is a Kubernetes operator that manages Harbor registries, projects, users, and members by synchronizing Custom Resources (CRs) with your Harbor instance via its API.

## Description

harbor-operator enables declarative management of Harbor resources in your Kubernetes cluster. You can define your Harbor registries, projects, users, and members using CRDs, and the operator will reconcile these definitions by creating or updating the corresponding entities in Harbor. A HarborConnection resource provides the base URL and optional default credentials for the Harbor API, while individual CRs (such as Registry) reference a HarborConnection for the operator to pull the connection details from.

## Getting Started

### Prerequisites

- **Go:** v1.23.0+
- **Docker:** v17.03+
- **kubectl:** v1.11.3+
- **Access to a Kubernetes v1.11.3+ cluster**
- **Kind (optional):** For local development and testing

### Local Development

For local development with Kind, follow these steps:

1. **Edit Your Hosts File**

   Add the following line to your hosts file (e.g., `/etc/hosts` on Linux or macOS):

   ```sh
   127.0.0.1 core.harbor.domain
   ```

   This entry allows you to resolve Harbor’s ingress locally.

2. **Start a Kind Cluster**

   Run the following command to create a Kind cluster:

   ```sh
   make kind-up
   ```

   This command creates a Kind cluster and deploys Nginx and Harbor.

3. **Deploy harbor-operator in Kind**

   Build a local image and deploy the operator with:

   ```sh
   make kind-deploy
   ```

   This target builds the image for the operator, loads it into the Kind cluster, and deploys it.

4. **Apply Sample Custom Resources**

   To get started with testing the operator, you can apply the sample CRs from the `config/samples/` directory:

   ```sh
   kubectl apply -k config/samples/
   ```

   This creates sample instances of your HarborConnection, Registry, and other resources.

### To Deploy on a Cluster

1. **Build and Push the Image**

   Build and push the operator image to your registry:

   ```sh
   make docker-build docker-push IMG=<your-registry>/harbor-operator:tag
   ```

2. **Install the CRDs**

   Install the Custom Resource Definitions into your cluster:

   ```sh
   make install
   ```

3. **Deploy the Operator**

   Deploy harbor-operator to your cluster using the image built above:

   ```sh
   make deploy IMG=<your-registry>/harbor-operator:tag
   ```

4. **Create Instances of Your CRs**

   Apply the sample CRs (or your custom YAMLs) to create Harbor resources:

   ```sh
   kubectl apply -k config/samples/
   ```

   > **NOTE:** Make sure the sample YAMLs have default values suitable for your environment.

### To Uninstall

1. **Delete Sample Custom Resources**

   ```sh
   kubectl delete -k config/samples/
   ```

2. **Uninstall the CRDs**

   ```sh
   make uninstall
   ```

3. **Undeploy the Operator**

   ```sh
   make undeploy
   ```

## Project Distribution

### By Providing a Bundle with All YAML Files

1. **Build the Installer**

   Build an installer that includes all CRDs and deployment manifests:

   ```sh
   make build-installer IMG=<your-registry>/harbor-operator:tag
   ```

   This command generates an `install.yaml` file in the `dist/` directory.

2. **Deploy Using the Installer**

   Users can install harbor-operator by running:

   ```sh
   kubectl apply -f https://raw.githubusercontent.com/<org>/harbor-operator/<tag-or-branch>/dist/install.yaml
   ```

### By Providing a Helm Chart

1. **Generate the Helm Chart**

   Generate a Helm Chart using the Helm plugin:

   ```sh
   kubebuilder edit --plugins=helm/v1-alpha
   ```

   This creates a chart under the `dist/chart` directory.

2. **Distribute the Chart**

   Users can then install harbor-operator using Helm from the generated chart.

   > **NOTE:** After making changes to the project, update the Helm Chart by re-running the above command (with the `--force` flag if necessary) to synchronize your changes.

## Contributing

Contributions are welcome! To contribute:

- Fork the repository and create a feature branch.
- Ensure your changes pass all tests and linters (`make fmt`, `make vet`, `make lint`).
- Submit a pull request with detailed descriptions of your changes.

For more detailed guidelines, please refer to the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html).

## License

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
