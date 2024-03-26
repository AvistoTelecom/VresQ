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

// SetupSourceAndDestinationKubernetesClients sets up dynamic Kubernetes clients based on the provided configurations.
func SetupSourceAndDestinationKubernetesClients(sourceDynamicClient *dynamic.DynamicClient, destinationDynamicClient *dynamic.DynamicClient, currentContext *CurrentContext, config *common.Config) {
	sourceKubeconfig := config.SourceKubeconfig
	sourceContext := config.SourceContext
	destinationContext := config.DestinationContext
	destinationKubeconfig := config.DestinationKubeconfig

	if currentContext.NoGivenContext {
		*sourceDynamicClient = GetKubernetesClient(sourceKubeconfig)
		*destinationDynamicClient = GetKubernetesClient(destinationKubeconfig)
	} else {
		if currentContext.SameOrOnlySourceContext {
			sourceKubeconfig = destinationKubeconfig
			sourceContext = destinationContext
		}

		*sourceDynamicClient = GetKubernetesClientWithContext(sourceKubeconfig, sourceContext)
		*destinationDynamicClient = GetKubernetesClientWithContext(destinationKubeconfig, destinationContext)
	}
}

// SetupSourceAndDestinationHelmClients sets up Helm clients based on the provided configurations.
func SetupSourceAndDestinationHelmClients(sourceHelmClient *helm.Client, destinationHelmClient *helm.Client, currentContext *CurrentContext, config *common.Config) {
	sourceKubeconfig := config.SourceKubeconfig
	sourceContext := config.SourceContext
	destinationContext := config.DestinationContext
	destinationKubeconfig := config.DestinationKubeconfig

	if currentContext.NoGivenContext {
		*sourceHelmClient = GetHelmClient(sourceKubeconfig, config.SourceVeleroNamespace)
		*destinationHelmClient = GetHelmClient(destinationKubeconfig, config.DestinationVeleroNamespace)
	} else {
		if currentContext.SameOrOnlySourceContext {
			sourceKubeconfig = destinationKubeconfig
			sourceContext = destinationContext
		}

		*sourceHelmClient = GetHelmClientWithContext(sourceKubeconfig, sourceContext, config.SourceVeleroNamespace)
		*destinationHelmClient = GetHelmClientWithContext(destinationKubeconfig, destinationContext, config.DestinationVeleroNamespace)
	}
}

// GetKubernetesClientWithContext returns a dynamic Kubernetes client based on the provided kubeconfig and context.
func GetKubernetesClientWithContext(kubeconfig, contextName string) dynamic.DynamicClient {
	config, err := buildConfigWithContextFromFlags(contextName, kubeconfig)
	if err != nil {
		log.Fatalf("Error building Kubernetes config: %v", err)
	}
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Fatalf("Fail to create the k8s dynamic client. Error: %v", err)
	}
	return *dynamicClient
}

// GetHelmClientWithContext returns a Helm client based on the provided kubeconfig, context, and namespace.
func GetHelmClientWithContext(kubeconfig, contextName, namespace string) helm.Client {
	config, err := buildConfigWithContextFromFlags(contextName, kubeconfig)
	if err != nil {
		log.Fatalf("Error building Kubernetes config: %v", err)
	}
	options := &helm.RestConfClientOptions{
		Options: &helm.Options{
			Namespace: namespace,
			Linting:   false,
		},
		RestConfig: config,
	}

	helmClient, err := helm.NewClientFromRestConf(options)
	if err != nil {
		log.Fatalf("Error creating Helm Kubernetes client: %v", err)
	}
	return helmClient
}

// GetKubernetesClient returns a dynamic Kubernetes client based on the provided kubeconfig.
func GetKubernetesClient(kubeconfig string) dynamic.DynamicClient {
	config, err := buildConfigWithContextFromFlags("", kubeconfig)
	if err != nil {
		log.Fatalf("Error building Kubernetes config: %v", err)
	}
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Fatalf("Fail to create the k8s dynamic client. Error: %v", err)
	}
	return *dynamicClient
}

// GetHelmClient returns a Helm client based on the provided kubeconfig and namespace.
func GetHelmClient(kubeconfig, namespace string) helm.Client {
	config, err := buildConfigWithContextFromFlags("", kubeconfig)
	if err != nil {
		log.Fatalf("Error building Kubernetes config: %v", err)
	}

	options := &helm.RestConfClientOptions{
		Options: &helm.Options{
			Namespace: namespace,
			Linting:   false,
		},
		RestConfig: config,
	}

	helmClient, err := helm.NewClientFromRestConf(options)
	if err != nil {
		log.Fatalf("Error creating Helm Kubernetes client: %v", err)
	}
	return helmClient
}

// buildConfigWithContextFromFlags constructs a Kubernetes client configuration with the provided context and kubeconfig.
func buildConfigWithContextFromFlags(context string, kubeconfigPath string) (*rest.Config, error) {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{
			CurrentContext: context,
		}).ClientConfig()
}
