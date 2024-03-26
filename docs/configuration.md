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