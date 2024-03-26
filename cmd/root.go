/*
VresQ command is used to facilitate the restoration of Kubernetes resources from a Velero backup
to a different cluster or to the same one. It provides various options for configuring the restore process.
Example usage:
$ vresq --source-kubeconfig=<source-path> --source-context=<source-context> --destination-context=<destination-context> --backup-name=<backup-name> --namespace-mapping=<source-namespace>=<target-namespace> --namespace=<namespace> --label-selector "<label-selector>" --restore-name=<restore-name>
*/
package cmd

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	common "vresq/pkg/common"
	kube "vresq/pkg/kubernetes"
	prompt "vresq/pkg/prompt"
	velero "vresq/pkg/velero"

	"github.com/go-logr/logr"
	"github.com/manifoldco/promptui"
	helm "github.com/mittwald/go-helm-client"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/client-go/dynamic"
	controller_logger "sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	defaultSourceKubeconfig string
	config                  common.Config
	sourceHelmClient        helm.Client
	sourceDynamiClient      dynamic.DynamicClient
	destinationHelmClient   helm.Client
	destinationDynamiClient dynamic.DynamicClient
)

const (
	// The name of our config file, without the file extension because viper supports many different config file languages.
	defaultConfigFilename = "vresq"

	// The environment variable prefix of all environment variables bound to our command line flags.
	// For example, --number is bound to VRESQ_NUMBER.
	envPrefix                  = "VRESQ"
	ReplaceHyphenWithCamelCase = false
	// Replace hyphenated flag names with camelCase in the config file
	ConfirmYes = common.ConfirmYes
	ConfirmNo  = common.ConfirmNo
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "vresq",
	Version: "main",
	Short:   "VresQ facilitates the restoration of Kubernetes resources from a Velero backup to the same or a different cluster with configurable options.",
	Long: `The "vresq" command facilitates the restoration of Kubernetes resources from a Velero backup
to the same or a different cluster. It supports various options for configuring the restoration process, including specifying
source and destination kubeconfig paths, source and destination contexts, backup name, namespace mappings, and more.

Example usage:
You can call vresq without specifying any arguments, in this case velero runs in interactive mode.
OR run using arguments:
  $ vresq --source-kubeconfig=<source-path> --source-context=<source-context> --destination-context=<destination-context> --backup-name=<backup-name> --namespace-mapping=<source-namespace>=<target-namespace> ==--restore-name=<restore-name>
`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return initConfig(cmd)
	},
	Run: func(cmd *cobra.Command, args []string) {
		if config.SourceKubeconfig == "" {
			var err error
			log.Printf("No source kubeconfig given, parsing contexts in default kubeconfig in %s ...", defaultSourceKubeconfig)
			config.SourceKubeconfig = defaultSourceKubeconfig
			config.SourceContext, err = prompt.ChooseKubeconfigContext(defaultSourceKubeconfig, "Source", &config)
			if err != nil {
				log.Fatalf("Error selecting source context: %v", err)
			}
		}
		if config.DestinationKubeconfig == "" {
			label := "No destination kubeconfig given, do you want to use the source kubeconfig as a destination "
			selected := prompt.ConfirmUserChoice(label)
			var defaultDestinationKubeconfig string
			var err error
			if selected {
				defaultDestinationKubeconfig = config.SourceKubeconfig
			} else {
				defaultDestinationKubeconfig, err = prompt.ChooseDestinationKubeconfig()
				if err != nil {
					log.Fatalf("%v", err)
				}
			}
			config.DestinationKubeconfig = defaultDestinationKubeconfig
			config.DestinationContext, err = prompt.ChooseKubeconfigContext(defaultDestinationKubeconfig, "Destination", &config)
			if err != nil {
				log.Fatalf("Error selecting destination context: %v", err)
			}
		}

		currentContext := kube.CurrentContext{
			SameOrOnlySourceKubeconfig: config.DestinationKubeconfig == "" || config.DestinationKubeconfig == config.SourceKubeconfig,
			SameOrOnlySourceContext:    config.DestinationContext == "" || config.DestinationContext == config.SourceContext,
			NoGivenContext:             config.DestinationContext == "" && config.SourceContext == "",
		}
		kube.SetupSourceAndDestinationKubernetesClients(&sourceDynamiClient, &destinationDynamiClient, &currentContext, &config)
		if config.SourceVeleroNamespace == "" {
			var err error
			veleroPod, err := velero.GetVeleroPod(&sourceDynamiClient)
			if err != nil {
				log.Fatalf("Error: could not discover source velero namespace. Please specify one. %v", err)
			}
			config.SourceVeleroNamespace = veleroPod.GetNamespace()
		}
		if config.DestinationVeleroNamespace == "" {
			var err error
			veleroPod, err := velero.GetVeleroPod(&destinationDynamiClient)
			config.DestinationVeleroNamespace = veleroPod.GetNamespace()
			if err != nil {
				if _, ok := err.(velero.NotFoundError); ok {
					log.Printf("Info: could not discover velero namespace in destination cluster.")
					confirmChoices := []string{ConfirmYes, ConfirmNo}
					promptSelect := promptui.Select{
						Label: "No Velero helm chart was discoverd in destination cluster, do you want to clone it from source cluster ?",
						Items: confirmChoices,
						Size:  2,
					}
					_, selected, err := promptSelect.Run()
					if err != nil {
						log.Fatalf("Error: %v", err)
					}
					if !currentContext.SameOrOnlySourceKubeconfig || !currentContext.SameOrOnlySourceContext {
						if config.DestinationVeleroNamespace == "" {
							label := "Namespace for velero installation in destination cluster:"
							validationError := "namespace name should match the regex: '[a-z0-9]([-a-z0-9]*[a-z0-9])?'"
							regex := regexp.MustCompile(`[a-z0-9]([-a-z0-9]*[a-z0-9])?`)
							chosenNamespace, err := prompt.UserInput(regex, validationError, label, "velero")
							if err != nil {
								log.Fatalf("Error: could not construct namespaces mapping, %v", err)
							}
							config.DestinationVeleroNamespace = chosenNamespace
						}
						if selected == ConfirmYes {
							log.Println("Cloning Velero helm release from source cluster...")
							kube.SetupSourceAndDestinationHelmClients(&sourceHelmClient, &destinationHelmClient, &currentContext, &config)
							velero.SetupVelero(sourceHelmClient, &sourceDynamiClient, destinationHelmClient, &destinationDynamiClient, &config)
						} else {
							log.Println("Skipping Velero helm release Cloning from source cluster...")
						}
					}
				} else {
					log.Fatal(err)
				}
			}
		}
		if config.RestoreName == "" {
			restoreName, err := prompt.ChooseRestoreName()
			if err != nil {
				log.Fatalf("Could not get Restore name, %v", err)
			}
			config.RestoreName = restoreName
		}
		if config.VeleroRestoreOptions.BackupName == "" {
			selected_backup, err := prompt.ChooseBackup(&sourceDynamiClient, config)
			config.VeleroRestoreOptions.BackupName = selected_backup
			if err != nil {
				log.Fatal(err)
			}
		}
		for _, ns := range config.VeleroRestoreOptions.IncludedNamespaces {
			log.Printf("---> %s", ns)
		}
		if len(config.VeleroRestoreOptions.IncludedNamespaces) == 0 {
			namespaces, err := prompt.ChooseNamespaces(&sourceDynamiClient, &config)
			if err != nil {
				log.Fatalf("Error: could not get namespaces that should be included in restore, %v", err)
			}
			config.VeleroRestoreOptions.IncludedNamespaces = namespaces
		}
		if len(config.VeleroRestoreOptions.NamespaceMapping) == 0 {
			var selected = false
			for {
				for _, namespace := range config.VeleroRestoreOptions.IncludedNamespaces {
					label := fmt.Sprintf("Destination namespace for namespace '%s' restoration :", namespace)
					validationError := "namespace name should match the regex: '[a-z0-9]([-a-z0-9]*[a-z0-9])?'"
					regex := regexp.MustCompile(`[a-z0-9]([-a-z0-9]*[a-z0-9])?`)
					chosenNamespace, err := prompt.UserInput(regex, validationError, label, fmt.Sprintf("%s-%s", config.RestoreName, namespace))
					if err != nil {
						log.Fatalf("Error: could not construct namespaces mapping, %v", err)
					}
					config.VeleroRestoreOptions.NamespaceMapping[namespace] = chosenNamespace
				}
				var labelParts []string
				labelParts = append(labelParts, "Do you confirm the following choice : ")
				for snamespace, dnamespace := range config.VeleroRestoreOptions.NamespaceMapping {
					labelParts = append(labelParts, fmt.Sprintf("%s ==> %s", snamespace, dnamespace))
				}
				label := strings.Join(labelParts, ", ")
				selected = prompt.ConfirmUserChoice(label)
				if selected {
					break
				}
			}
		}
		velero.SetupVeleroBackupLocation(&sourceDynamiClient, &destinationDynamiClient, &config)
		velero.SetupVeleroConfigmap(&sourceDynamiClient, &destinationDynamiClient, config.DestinationVeleroNamespace)
		err := velero.CreateVeleroRestore(&destinationDynamiClient, config.DestinationVeleroNamespace, config.RestoreName, config.VeleroRestoreOptions)
		if err != nil {
			log.Fatalf("Error creating Velero Restore: %v", err)
		}
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&config.SourceContext, "source-context", "s", viper.GetString("SOURCE_CONTEXT"), "name of the source context in kubeconfig")
	rootCmd.PersistentFlags().StringVarP(&config.DestinationContext, "destination-context", "d", viper.GetString("DESTINATION_CONTEXT"), "name of the destination context in kubeconfig")
	rootCmd.PersistentFlags().StringVarP(&config.SourceKubeconfig, "source-kubeconfig", "k", viper.GetString("SOURCE_KUBECONFIG"), "absolute path to the source kubeconfig file")
	rootCmd.PersistentFlags().StringVarP(&config.DestinationKubeconfig, "destination-kubeconfig", "f", viper.GetString("DESTINATION_KUBECONFIG"), "absolute path to the source kubeconfig file")
	rootCmd.PersistentFlags().StringVarP(&config.SourceVeleroHelmReleaseName, "source-velero-helm-release-name", "r", viper.GetString("SOURCE_VELERO_HELM_RELEASE_NAME"), "velero Helm release name in the source cluster.")
	rootCmd.PersistentFlags().StringVarP(&config.SourceVeleroNamespace, "source-velero-namespace", "", viper.GetString("SOURCE_VELERO_NAMESPACE"), "source Velero namespace")
	rootCmd.PersistentFlags().StringVarP(&config.DestinationVeleroNamespace, "destination-velero-namespace", "", viper.GetString("DESTINATION_VELERO_NAMESPACE"), "destination Velero namespace")
	rootCmd.PersistentFlags().StringVarP(&config.RestoreName, "restore-name", "o", viper.GetString("RESTORE_NAME"), "name for Velero Restore")
	rootCmd.PersistentFlags().StringVarP(&config.VeleroRestoreOptions.BackupName, "backup-name", "b", viper.GetString("BACKUP_NAME"), "Velero backup name")
	rootCmd.PersistentFlags().StringVarP(&config.VeleroRestoreOptions.ScheduleName, "schedule-name", "c", viper.GetString("SCHEDULE_NAME"), "Velero schedule name")
	rootCmd.PersistentFlags().DurationVarP(&config.VeleroRestoreOptions.ItemOperationTimeout, "item-operation-timeout", "t", viper.GetDuration("ITEM_OPERATION_TIMEOUT"), "Time used to wait for asynchronous BackupItemAction operations")
	rootCmd.PersistentFlags().StringSliceVarP(&config.VeleroRestoreOptions.IncludedNamespaces, "included-namespaces", "i", viper.GetStringSlice("INCLUDED_NAMESPACES"), "Array of namespaces to include in the restore")
	rootCmd.PersistentFlags().StringSliceVarP(&config.VeleroRestoreOptions.ExcludedNamespaces, "excluded-namespaces", "e", viper.GetStringSlice("EXCLUDED_NAMESPACES"), "Array of namespaces to exclude from the restore")
	rootCmd.PersistentFlags().StringSliceVarP(&config.VeleroRestoreOptions.IncludedResources, "included-resources", "l", viper.GetStringSlice("INCLUDED_RESOURCES"), "Array of resources to include in the restore")
	rootCmd.PersistentFlags().StringSliceVarP(&config.VeleroRestoreOptions.ExcludedResources, "excluded-resources", "x", viper.GetStringSlice("EXCLUDED_RESOURCES"), "Array of resources to exclude from the restore")
	rootCmd.PersistentFlags().BoolVarP(&config.VeleroRestoreOptions.IncludeClusterResources, "include-cluster-resources", "C", viper.GetBool("INCLUDE_CLUSTER_RESOURCES"), "Whether or not to include cluster-scoped resources")
	rootCmd.PersistentFlags().StringToStringVarP(&config.VeleroRestoreOptions.LabelSelector, "label-selector", "L", viper.GetStringMapString("LABEL_SELECTOR"), "Individual objects must match this label selector to be included in the restore")
	rootCmd.PersistentFlags().StringToStringVarP(&config.VeleroRestoreOptions.OrLabelSelectors, "or-label-selectors", "O", viper.GetStringMapString("OR_LABEL_SELECTORS"), "Individual object when matched with any of the label selectors specified in the set are to be included in the restore")
	rootCmd.PersistentFlags().StringToStringVarP(&config.VeleroRestoreOptions.NamespaceMapping, "namespace-mapping", "M", viper.GetStringMapString("NAMESPACE_MAPPING"), "Map of source namespace names to target namespace names to restore into")
	rootCmd.PersistentFlags().BoolVarP(&config.VeleroRestoreOptions.RestorePVs, "restore-pvs", "p", viper.GetBool("RESTORE_PVS"), "Whether to restore all included PVs from snapshot")
	rootCmd.PersistentFlags().BoolVarP(&config.VeleroRestoreOptions.PreserveNodePorts, "preserve-node-ports", "P", viper.GetBool("PRESERVE-NODE-PORTS"), "Whether to restore old nodePorts from backup")
	rootCmd.PersistentFlags().StringVarP(&config.VeleroRestoreOptions.ExistingResourcePolicy, "existing-resource-policy", "E", viper.GetString("EXISTING_RESOURCE_POLICY"), "Restore behavior for the Kubernetes resource to be restored")
	setDefaultSourceKubeconfig()
	controller_logger.SetLogger(logr.Logger{})
}
