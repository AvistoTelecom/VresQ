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
$ VRESQ_VERSION="1.0.0-alpha" && \
wget "https://github.com/AvistoTelecom/VresQ/releases/download/$VRESQ_VERSION/VresQ_Linux_x86_64.tar.gz" && \
tar -zxvf VresQ_Linux_x86_64.tar.gz && \
chmod +x VresQ && \
sudo mv VresQ /usr/local/bin/vresq && \
rm VresQ_Linux_x86_64.tar.gz
```

# Documentation
[Docs](./docs/)

# License
This project is licensed under the Apache-2.0 license.

# Contributing
We welcome contributions! If you find any issues or have suggestions, please open an issue or submit a pull request.

# Acknowledgments
Special thanks to the Velero project for providing a robust backup and restore solution for Kubernetes.