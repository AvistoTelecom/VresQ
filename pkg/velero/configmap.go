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

func SetupVeleroConfigmap(sourceDynamicClient dynamic.Interface, destinationDynamicClient dynamic.Interface, namespace string) {
	configMaps, err := destinationDynamicClient.Resource(configmapGVR).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Error: could not retrieve the list of config maps %v", err)
	}
	sourceStorageClasses, err := sourceDynamicClient.Resource(storageClassGVR).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Error: could not list storage classes in source cluster. %v", err)
	}
	destinationStorageClasses, err := destinationDynamicClient.Resource(storageClassGVR).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Fatalf("Error: could not list storage classes in destination cluster. %v", err)
	}
	destinationDefaultStorageClass := getDestinationDefaultStorageClass(destinationStorageClasses.Items)
	oldStorageClasses := getSourceStorageClassNames(sourceStorageClasses.Items)
	configMap, found := configMapExists(configMaps.Items)
	if !found {
		_, err = createStorageClassConfigMap(destinationDynamicClient, configMapName, oldStorageClasses, destinationDefaultStorageClass, namespace)
		if err != nil {
			log.Fatal(err)
		}
	} else {
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

func configMapExists(configMaps []unstructured.Unstructured) (*unstructured.Unstructured, bool) {
	for _, configMap := range configMaps {
		if configMap.GetName() == configMapName {
			return &configMap, true
		}
	}
	return nil, false
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
				"labels": map[string]interface{}{
					"velero.io/plugin-config":        "",
					"velero.io/change-storage-class": "RestoreItemAction",
				},
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
