package common

import (
	"time"

	"k8s.io/client-go/dynamic"
)

const (
	ConfirmYes     = "Yes"
	ConfirmNo      = "No"
	VeleroApiGroup = "velero.io"
	ApiVersion     = "v1"
	ConfigMapName  = "change-storage-class-config"
)

type DynamicClientInterface interface {
	Resource(resource string, namespace string) dynamic.ResourceInterface
	// Add other methods as needed
}

// Config holds configuration parameters
type Config struct {
	SourceContext               string `mapstructure:"source-context"`
	DestinationContext          string `mapstructure:"destination-context"`
	SourceKubeconfig            string `mapstructure:"source-kubeconfig"`
	DestinationKubeconfig       string `mapstructure:"destination-kubeconfig"`
	RestoreName                 string `mapstructure:"restore-name"`
	SourceVeleroHelmReleaseName string `mapstructure:"source-velero-helm-release-name"`
	SourceVeleroNamespace       string `mapstructure:"source-velero-namespace"`
	DestinationVeleroNamespace  string `mapstructure:"destination-velero-namespace"`
	VeleroRestoreOptions        VeleroRestoreOptions
}

type VeleroRestoreOptions struct {
	BackupName              string            `mapstructure:"backup-name"`
	ScheduleName            string            `mapstructure:"schedule-name"`
	ItemOperationTimeout    time.Duration     `mapstructure:"item-operation-timeout"`
	IncludedNamespaces      []string          `mapstructure:"included-namespaces"`
	ExcludedNamespaces      []string          `mapstructure:"excluded-namespaces"`
	IncludedResources       []string          `mapstructure:"included-resources"`
	ExcludedResources       []string          `mapstructure:"excluded-resources"`
	IncludeClusterResources bool              `mapstructure:"include-cluster-resources"`
	LabelSelector           map[string]string `mapstructure:"label-selector"`
	OrLabelSelectors        map[string]string `mapstructure:"or-label-selectors"`
	NamespaceMapping        map[string]string `mapstructure:"namespace-mapping"`
	RestorePVs              bool              `mapstructure:"restore-pvs"`
	PreserveNodePorts       bool              `mapstructure:"preserve-node-ports"`
	ExistingResourcePolicy  string            `mapstructure:"existing-resource-policy"`
}
