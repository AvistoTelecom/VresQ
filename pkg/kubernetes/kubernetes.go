package kubernetes

import (
	"log"
	"vresq/pkg/common"

	helm "github.com/mittwald/go-helm-client"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type CurrentContext struct {
	SameOrOnlySourceKubeconfig bool
	NoGivenContext             bool
	SameOrOnlySourceContext    bool
}

func SetupSourceAndDestinationClients(sourceHelmClient *helm.Client, sourceDynamicClient *dynamic.DynamicClient, destinationHelmClient *helm.Client, destinationDynamicClient *dynamic.DynamicClient, currentContext *CurrentContext, config *common.Config) {
	sourceKubeconfig := config.SourceKubeconfig
	sourceContext := config.SourceContext
	destinationContext := config.DestinationContext
	destinationKubeconfig := config.DestinationKubeconfig

	if currentContext.NoGivenContext {
		*sourceHelmClient, *sourceDynamicClient = GetKubernetesClients(sourceKubeconfig)
		*destinationHelmClient, *destinationDynamicClient = GetKubernetesClients(destinationKubeconfig)
	} else {
		if currentContext.SameOrOnlySourceContext {
			sourceKubeconfig = destinationKubeconfig
			sourceContext = destinationContext
		}

		*sourceHelmClient, *sourceDynamicClient = GetKubernetesClientsWithContext(sourceKubeconfig, sourceContext)
		*destinationHelmClient, *destinationDynamicClient = GetKubernetesClientsWithContext(destinationKubeconfig, destinationContext)
	}
}

func GetKubernetesClientsWithContext(kubeconfig, contextName string) (helm.Client, dynamic.DynamicClient) {
	// Create Kubernetes client configuration
	config, err := buildConfigWithContextFromFlags(contextName, kubeconfig)
	if err != nil {
		log.Fatalf("Error building Kubernetes config: %v", err)
	}
	opt := &helm.RestConfClientOptions{
		Options: &helm.Options{
			Namespace: "velero", // Change this to the namespace you wish the client to operate in.
			// Debug:            true,
			Linting: false, // Change this to false if you don't want linting.
		},
		RestConfig: config,
	}

	helmClient, err := helm.NewClientFromRestConf(opt)
	if err != nil {
		log.Fatalf("Error creating Helm Kubernetes client: %v", err)
	}
	dynamiClient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Fatalf("Fail to create the k8s dynamic client. Errorf - %v", err)
	}

	return helmClient, *dynamiClient
}

func GetKubernetesClients(kubeconfig string) (helm.Client, dynamic.DynamicClient) {
	// Create Kubernetes client configuration
	config, err := buildConfigWithContextFromFlags("", kubeconfig)
	if err != nil {
		log.Fatalf("Error building Kubernetes config: %v", err)
	}

	opt := &helm.RestConfClientOptions{
		Options: &helm.Options{
			Namespace: "velero", // Change this to the namespace you wish the client to operate in.
			// Debug:            true,
			Linting: false, // Change this to false if you don't want linting.
		},
		RestConfig: config,
	}

	helmClient, err := helm.NewClientFromRestConf(opt)
	if err != nil {
		log.Fatalf("Error creating Helm Kubernetes client: %v", err)
	}

	dynamiClient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Fatalf("Fail to create the k8s dynamic client. Errorf - %v", err)
	}

	return helmClient, *dynamiClient
}

func buildConfigWithContextFromFlags(context string, kubeconfigPath string) (*rest.Config, error) {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{
			CurrentContext: context,
		}).ClientConfig()
}
