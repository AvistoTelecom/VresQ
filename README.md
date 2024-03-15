# VresQ - Easily restore Kubernetes workloads from Velero backups
![image](images/vresq-logo.png)

# Overview
VresQ is an open-source command-line tool designed to simplify the restoration of Kubernetes resources from Velero backups. Whether you need to recover workloads on the same cluster or migrate them to a different one, VresQ provides configurable options to streamline the restoration process.

## Why Use VresQ
- 💸 **Free and Open-Source:** VresQ is free to use.
- 🎮 **Interactive Mode:** VresQ offers an interactive mode to guide you through the restoration process step by step.
- 🌐 **Cluster Flexibility:** Restore workloads on the same cluster or migrate them to a different one.
- 🚀 **No Dependencies:** As a self-contained binary, VresQ has no dependencies, making it easy to run.
- 🖥️ **OS Agnostic:** VresQ is designed to be platform-agnostic, providing seamless support across a variety of operating systems.
- ⚙️ **Flexible Configuration:** Easily configure the restoration process with various options, such as source and destination kubeconfig paths, backup names, namespace mappings, and more.

## Supported Platforms

| Operating System | Architecture | Support Status |
| ----------------- | ------------ | -------------- |
| Linux             | amd64        | ✅ Supported   |
| Windows           | amd64        | ✅ Supported   |
| macOS             | arm64        | 🚧 Coming Soon  |

# Usage
Example usage:
```shell
$ vresq \
--source-kubeconfig=<source-path> \
--source-context=<source-context> \
--destination-kubeconfig=<destination-path> \
--destination-context=<destination-context> \
--backup-name=<backup-name> \
--namespace-mapping=<source-namespace-1>=<target-namespace-1>,<source-namespace-2>=<target-namespace-2> \
--restore-name=<restore-name>
```
For interactive mode:
```shell
$ vresq
```
# Prerequisites
- Velero must be installed on the source cluster.
- The source cluster must have an existing Velero backup.

# Installation
## Linux
```shell
$ VRESQ_VERSION="main" && \
wget "https://storage.googleapis.com/vresq/vresq_linux_amd64_${VRESQ_VERSION}.tar.gz" && \
tar -zxvf vresq_linux_amd64_${VRESQ_VERSION}.tar.gz && \
chmod +x vresq_linux_amd64 && \
sudo mv vresq_linux_amd64 /usr/local/bin/vresq && \
rm vresq_linux_amd64_${VRESQ_VERSION}.tar.gz
```

# Configuration
VresQ uses a configuration file to manage settings. You can also set configuration options via command-line flags or environment variables.

Precedence order is:
**command-line flags --> environment variables --> configuration file**

| Argument                          | Environment Variable               | Config File Field               | Default Value     |
|-----------------------------------|------------------------------------|---------------------------------|-------------------|
| --source-context, -s              | VRESQ_SOURCE_CONTEXT               | source-context                  | ""                |
| --destination-context, -d         | VRESQ_DESTINATION_CONTEXT          | destination-context             | ""                |
| --source-kubeconfig, -k           | VRESQ_SOURCE_KUBECONFIG            | source-kubeconfig               | ""                |
| --destination-kubeconfig, -f      | VRESQ_DESTINATION_KUBECONFIG       | destination-kubeconfig          | ""                |
| --source-velero-helm-release-name, -r | VRESQ_SOURCE_VELERO_HELM_RELEASE_NAME | source-velero-helm-release-name | ""                |
| --source-velero-namespace         | VRESQ_SOURCE_VELERO_NAMESPACE      | source-velero-namespace         | ""                |
| --destination-velero-namespace    | VRESQ_DESTINATION_VELERO_NAMESPACE | destination-velero-namespace    | ""                |
| --restore-name, -o                | VRESQ_RESTORE_NAME                 | restore-name                    | ""                |
| --backup-name, -b                 | VRESQ_BACKUP_NAME                  | backup-name                     | ""                |
| --schedule-name, -c               | VRESQ_SCHEDULE_NAME                | schedule-name                   | ""                |
| --item-operation-timeout, -t      | VRESQ_ITEM_OPERATION_TIMEOUT       | item-operation-timeout          | 4h                |
| --included-namespaces, -i         | VRESQ_INCLUDED_NAMESPACES          | included-namespaces             | []                |
| --excluded-namespaces, -e         | VRESQ_EXCLUDED_NAMESPACES          | excluded-namespaces             | []                |
| --included-resources, -l          | VRESQ_INCLUDED_RESOURCES           | included-resources              | ["*"]             |
| --excluded-resources, -x          | VRESQ_EXCLUDED_RESOURCES           | excluded-resources              | []                |
| --include-cluster-resources, -C   | VRESQ_INCLUDE_CLUSTER_RESOURCES    | include-cluster-resources       | false             |
| --label-selector, -L              | VRESQ_LABEL_SELECTOR               | label-selector                  | {}                |
| --or-label-selectors, -O          | VRESQ_OR_LABEL_SELECTORS           | or-label-selectors              | {}                |
| --namespace-mapping, -M           | VRESQ_NAMESPACE_MAPPING            | namespace-mapping               | {}                |
| --restore-pvs, -p                 | VRESQ_RESTORE_PVS                  | restore-pvs                     | true              |
| --preserve-node-ports, -P         | VRESQ_PRESERVE_NODE_PORTS          | preserve-node-ports             | true              |
| --existing-resource-policy, -E    | VRESQ_EXISTING_RESOURCE_POLICY     | existing-resource-policy        | "none"            |


# License
This project is licensed under the X License.

# Contributing
We welcome contributions! If you find any issues or have suggestions, please open an issue or submit a pull request.

# Acknowledgments
Special thanks to the Velero project for providing a robust backup and restore solution for Kubernetes.