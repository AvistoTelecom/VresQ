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

func GetKubernetesClientWithContext(kubeconfig, contextName string) dynamic.DynamicClient {
	config, err := buildConfigWithContextFromFlags(contextName, kubeconfig)
	if err != nil {
		log.Fatalf("Error building Kubernetes config: %v", err)
	}
	dynamiClient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Fatalf("Fail to create the k8s dynamic client. Errorf - %v", err)
	}
	return *dynamiClient
}

func GetHelmClientWithContext(kubeconfig, contextName, namespace string) helm.Client {
	config, err := buildConfigWithContextFromFlags(contextName, kubeconfig)
	if err != nil {
		log.Fatalf("Error building Kubernetes config: %v", err)
	}
	opt := &helm.RestConfClientOptions{
		Options: &helm.Options{
			Namespace: namespace,
			// Debug:            true,
			Linting: false,
		},
		RestConfig: config,
	}

	helmClient, err := helm.NewClientFromRestConf(opt)
	if err != nil {
		log.Fatalf("Error creating Helm Kubernetes client: %v", err)
	}
	return helmClient
}

func GetKubernetesClient(kubeconfig string) dynamic.DynamicClient {
	config, err := buildConfigWithContextFromFlags("", kubeconfig)
	if err != nil {
		log.Fatalf("Error building Kubernetes config: %v", err)
	}
	dynamiClient, err := dynamic.NewForConfig(config)
	if err != nil {
		log.Fatalf("Fail to create the k8s dynamic client. Errorf - %v", err)
	}
	return *dynamiClient
}

func GetHelmClient(kubeconfig, namespace string) helm.Client {
	config, err := buildConfigWithContextFromFlags("", kubeconfig)
	if err != nil {
		log.Fatalf("Error building Kubernetes config: %v", err)
	}

	opt := &helm.RestConfClientOptions{
		Options: &helm.Options{
			Namespace: namespace,
			// Debug:            true,
			Linting: false,
		},
		RestConfig: config,
	}

	helmClient, err := helm.NewClientFromRestConf(opt)
	if err != nil {
		log.Fatalf("Error creating Helm Kubernetes client: %v", err)
	}
	return helmClient
}

func buildConfigWithContextFromFlags(context string, kubeconfigPath string) (*rest.Config, error) {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
		&clientcmd.ConfigOverrides{
			CurrentContext: context,
		}).ClientConfig()
}
