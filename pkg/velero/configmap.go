package velero

import (
	"context"
	"log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

func SetupVeleroConfigmap(sourceDynamicClient dynamic.Interface, destinationDynamicClient dynamic.Interface, namespace string) {
	configMaps, err := destinationDynamicClient.Resource(schema.GroupVersionResource{
		Group:    "",
		Version:  apiVersion,
		Resource: "configmaps",
	}).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Error: could not retrieve the list of config maps %v", err)
	}

	if !configMapExists(configMaps.Items) {
		sourceStorageClasses, err := sourceDynamicClient.Resource(schema.GroupVersionResource{
			Group:    "storage.k8s.io",
			Version:  apiVersion,
			Resource: "storageclasses",
		}).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			log.Fatalf("Error: could not list storage classes in source cluster. %v", err)
		}

		destinationStorageClasses, err := destinationDynamicClient.Resource(schema.GroupVersionResource{
			Group:    "storage.k8s.io",
			Version:  apiVersion,
			Resource: "storageclasses",
		}).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			log.Fatalf("Error: could not list storage classes in destination cluster. %v", err)
		}

		destinationDefaultStorageClass := getDestinationDefaultStorageClass(destinationStorageClasses.Items)

		oldStorageClasses := getSourceStorageClassNames(sourceStorageClasses.Items)

		_, err = createStorageClassConfigMap(destinationDynamicClient, configMapName, oldStorageClasses, destinationDefaultStorageClass, namespace)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func configMapExists(configMaps []unstructured.Unstructured) bool {
	for _, configMap := range configMaps {
		if configMap.GetName() == configMapName {
			return true
		}
	}
	return false
}

func createStorageClassConfigMap(dynamicClient dynamic.Interface, configMapName string, oldStorageClasses []string, newStorageClass string, namespace string) (map[string]string, error) {
	data := map[string]string{}

	// Populate ConfigMap data
	for _, oldStorageClass := range oldStorageClasses {
		data[oldStorageClass] = newStorageClass
	}

	// Create ConfigMap object
	configMap := unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      configMapName,
				"namespace": namespace,
			},
			"data": data,
		},
	}

	err := createResource(dynamicClient, namespace, &configMap, "configmaps")
	if err != nil {
		return nil, err
	} else {
		log.Println("Successfully configured Velero to map all storage classes in old cluster to default storage class of destination.")
	}

	return data, nil
}
