package velero

import (
	"context"
	"fmt"
	"log"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

var (
	configmapGVR = schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "configmaps",
	}
)

// SetupVeleroConfigmap sets up Velero configuration for mapping old storage classes to the default storage class of the destination cluster.
func SetupVeleroConfigmap(sourceDynamicClient dynamic.Interface, destinationDynamicClient dynamic.Interface, namespace string) {
	// Retrieve the list of config maps in the destination namespace
	configMaps, err := destinationDynamicClient.Resource(configmapGVR).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Error: could not retrieve the list of config maps %v", err)
	}

	// List storage classes in the source and destination clusters
	sourceStorageClasses, err := sourceDynamicClient.Resource(storageClassGVR).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Error: could not list storage classes in the source cluster. %v", err)
	}
	destinationStorageClasses, err := destinationDynamicClient.Resource(storageClassGVR).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Error: could not list storage classes in the destination cluster. %v", err)
	}

	// Find the default storage class of the destination cluster
	destinationDefaultStorageClass := getDestinationDefaultStorageClass(destinationStorageClasses.Items)

	// Get the names of storage classes in the source cluster
	oldStorageClasses := getSourceStorageClassNames(sourceStorageClasses.Items)

	// Check if the Velero config map already exists in the destination cluster
	configMap, found := configMapExists(configMaps.Items)

	// If the config map does not exist, create it
	if !found {
		_, err = createStorageClassConfigMap(destinationDynamicClient, configMapName, oldStorageClasses, destinationDefaultStorageClass, namespace)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		// Update the existing config map with the new storage class mappings
		data, _, _ := unstructured.NestedStringMap(configMap.Object, "data")
		for _, oldStorageClass := range oldStorageClasses {
			data[oldStorageClass] = destinationDefaultStorageClass
		}
		configMap.Object["data"] = data
		_, err = destinationDynamicClient.Resource(configmapGVR).Namespace(namespace).Update(context.TODO(), configMap, metav1.UpdateOptions{})
		if err != nil {
			panic(err.Error())
		}
		fmt.Printf("ConfigMap %s in namespace %s updated successfully\n", configMapName, namespace)
	}
}

// configMapExists checks if the Velero config map already exists in the destination cluster.
func configMapExists(configMaps []unstructured.Unstructured) (*unstructured.Unstructured, bool) {
	for _, configMap := range configMaps {
		if configMap.GetName() == configMapName {
			return &configMap, true
		}
	}
	return nil, false
}

// createStorageClassConfigMap creates a new Velero config map with the specified storage class mappings.
func createStorageClassConfigMap(dynamicClient dynamic.Interface, configMapName string, oldStorageClasses []string, newStorageClass string, namespace string) (map[string]string, error) {
	data := map[string]string{}

	// Populate ConfigMap data with storage class mappings
	for _, oldStorageClass := range oldStorageClasses {
		data[oldStorageClass] = newStorageClass
	}

	// Create the ConfigMap object
	configMap := unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "ConfigMap",
			"metadata": map[string]interface{}{
				"name":      configMapName,
				"namespace": namespace,
				"labels": map[string]interface{}{
					"velero.io/plugin-config":        "",
					"velero.io/change-storage-class": "RestoreItemAction",
				},
			},
			"data": data,
		},
	}

	// Create the ConfigMap resource in the cluster
	err := createResource(dynamicClient, namespace, &configMap, "configmaps")
	if err != nil {
		return nil, err
	} else {
		log.Println("Successfully configured Velero to map all storage classes in the old cluster to the default storage class of the destination.")
	}

	return data, nil
}
