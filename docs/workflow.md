## VresQ Command Workflow

The **VresQ** command orchestrates the restoration of Kubernetes resources from a Velero backup, ensuring a seamless process with configurable options. Below is an expanded workflow outlining how **VresQ** operates:

### 1. Configuration Setup

- **Source and Destination Configuration**: Users specify the source and destination Kubernetes configurations, including kubeconfig paths and contexts. These configurations define where the Velero backup resides and where the resources will be restored.

### 2. Automatic Velero Deployment (Optional, Prompted)

- **Source Velero Helm Release Name**: If Velero was deployed using Helm on the source cluster, users can specify the Helm release name to clone configuration from. Otherwise, the helm release is automatically detected (release name should contain the string "velero")
This information is crucial for cloning the Velero setup to the destination cluster.

- **Automatic Velero Deployment**: If Velero was installed using Helm on the source cluster and no velero server is detected in the destination cluster, **VresQ** can automatically deploy the Velero Helm chart in the destination cluster with the same values used in the source cluster.

- **Manual Resource Creation**: If any resources required by Velero are not handled by Helm (e.g., secrets), users need to create them manually on the destination cluster to support the Velero deployment.

### 3. Backup Selection

- **Backup Listing**: After Velero deployment, **VresQ** automatically lists the available backups on the source cluster. Each backup is accompanied by its status, completion date, and included namespaces.

### 4. Namespace Selection for Restoration

- **Namespace Selection**: Once a backup is selected, **VresQ** prompts the user to choose the namespaces they want to restore from that specific backup. This step provides granular control over the restoration process, allowing users to select only the namespaces they need to restore.

### 5. Namespace Mapping

- **Namespace Mapping**: If namespace mapping is not provided, **VresQ** prompts the user to specify mappings between source and target namespaces for the restore operation. This step ensures that resources are restored to the appropriate namespaces in the destination cluster.

### 6. Velero Restore Configuration

- **Velero Restore Configuration**: Users can configure various aspects of the restore operation using flags and options. This includes specifying the inclusion or exclusion of specific resources, defining the behavior for PersistentVolumes (PVs), preserving NodePorts, and setting resource policies.

### 7. Velero Restore Execution

- **Velero Restore Initialization**: Once all configurations are set, **VresQ** initiates the Velero restore operation in the destination cluster. It leverages Velero's capabilities to create the necessary resources according to the specified parameters.

- **Progress Monitoring**: During the restore process, **VresQ** provides updates on the progress, allowing users to monitor the status of the restoration.

### 8. Confirmation and Feedback

- **Confirmation**: Once the restoration is successful, **VresQ** confirms the completion and provides feedback to the user, indicating that the process was executed without errors.

- **Feedback and Error Handling**: In case of errors or failures during the restoration, **VresQ** provides detailed feedback and error messages.
