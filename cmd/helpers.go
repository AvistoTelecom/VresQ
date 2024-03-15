package cmd

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

func setDefaultSourceKubeconfig() {
	// Get the user's home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Error getting user's home directory: %v\n", err)
		return
	}

	// Construct the defaultSourceKubeconfig based on the user's home directory and OS
	switch runtime.GOOS {
	case "windows":
		defaultSourceKubeconfig = homeDir + "\\.kube\\config"
	case "linux":
		defaultSourceKubeconfig = homeDir + "/.kube/config"
	default:
		fmt.Printf("Unsupported operating system: %s\n", runtime.GOOS)
	}
}

func bindFlags(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		// Determine the naming convention of the flags when represented in the config file
		configName := f.Name
		// If using camelCase in the config file, replace hyphens with a camelCased string.
		// Since viper does case-insensitive comparisons, we don't need to bother fixing the case, and only need to remove the hyphens.
		if ReplaceHyphenWithCamelCase {
			configName = strings.ReplaceAll(f.Name, "-", "_")
		}
		// Apply the viper config value to the flag when the flag is not set and viper has a value
		if !f.Changed && v.IsSet(configName) {
			val := v.Get(configName)
			cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
		}
	})
}

// initConfig reads in config file and ENV variables if set.
func initConfig(cmd *cobra.Command) error {
	// Initialize viper configuration
	v := viper.New()
	v.SetConfigName(defaultConfigFilename)
	v.SetConfigType("yaml")

	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	// Set v configuration file search paths and name
	v.AddConfigPath("/etc/restore/")
	v.AddConfigPath("$HOME/.restore")
	v.AddConfigPath(".")
	v.SetConfigName("config")

	// Read configuration file
	if err := v.ReadInConfig(); err != nil {
		// It's okay if there isn't a config file
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}
	v.SetDefault("source-context", "")
	v.SetDefault("destination-context", "")
	v.SetDefault("namespace", "velero")
	v.SetDefault("destination-kubeconfig", "")
	v.SetDefault("source-velero-helm-release-name", "")
	v.SetDefault("source-velero-namespace", "")
	v.SetDefault("destination-velero-namespace", "")
	v.SetDefault("backup-name", "")
	v.SetDefault("schedule-name", "")
	v.SetDefault("item-operation-timeout", 4*time.Hour)
	v.SetDefault("included-namespaces", "")
	v.SetDefault("excluded-namespaces", "")
	v.SetDefault("included-resources", "*")
	v.SetDefault("excluded-resources", "")
	v.SetDefault("include-cluster-resources", false)
	v.SetDefault("label-selector", "")
	v.SetDefault("or-label-selectors", "")
	v.SetDefault("namespace-mapping", "")
	v.SetDefault("restore-pvs", true)
	v.SetDefault("preserve-node-ports", true)
	v.SetDefault("existing-resource-policy", "none")
	v.SetEnvPrefix(envPrefix)
	// Bind environment variables
	v.AutomaticEnv()
	bindFlags(cmd, v)
	err := v.Unmarshal(&config)
	if err != nil {
		log.Fatalf("Error: could not read configuration, %v", err)
	}
	return nil
}
