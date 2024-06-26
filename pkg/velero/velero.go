package velero

import (
	"context"
	"errors"
	"fmt"
	"log"
	common "vresq/pkg/common"

	helm "github.com/mittwald/go-helm-client"
	"gopkg.in/yaml.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

type VeleroBackupStorageLocationOptions struct {
	AccessMode    string
	Config        map[string]interface{}
	Default       bool
	ObjectStorage map[string]interface{}
	Provider      string
}

// NotFoundError represents an error indicating that a resource was not found.
type NotFoundError struct {
	Err error
}

// Error returns the error message.
func (r NotFoundError) Error() string {
	return r.Err.Error()
}

// Is checks if the given error matches the NotFoundError.
func (r NotFoundError) Is(e error) bool {
	return r.Err.Error() == e.Error()
}

const (
	ConfirmYes     = common.ConfirmYes
	ConfirmNo      = common.ConfirmNo
	veleroApiGroup = common.VeleroApiGroup
	apiVersion     = common.ApiVersion
	configMapName  = common.ConfigMapName
)

// mapToYAML converts a map to YAML format.
func mapToYAML(data map[string]interface{}) (string, error) {
	yamlBytes, err := yaml.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(yamlBytes), nil
}

// getVeleroPod retrieves the Velero pod running in the cluster.
func GetVeleroPod(dynamicClient dynamic.Interface) (unstructured.Unstructured, error) {
	// Create a GVR (Group, Version, Resource) for the Pods resource
	podsGVR := schema.GroupVersionResource{
		Group:    "",
		Version:  apiVersion,
		Resource: "pods",
	}

	// List pods in the "velero" namespace with label selector "name=velero"
	podList, err := dynamicClient.Resource(podsGVR).Namespace("").List(context.TODO(), metav1.ListOptions{
		LabelSelector: "name=velero",
	})
	if err != nil {
		return unstructured.Unstructured{}, err
	}

	if len(podList.Items) == 0 {
		return unstructured.Unstructured{}, NotFoundError{Err: errors.New("velero server not found")}
	}
	return podList.Items[0], nil
}

// getVeleroPodSecretName retrieves the name of the secret associated with the Velero pod.
func getVeleroPodSecretName(veleroPod *unstructured.Unstructured) (string, error) {
	volumes, found, _ := unstructured.NestedSlice(veleroPod.Object, "spec", "volumes")
	if !found {
		return "", fmt.Errorf("could not get velero pod volumes in source cluster")
	}

	for _, volume := range volumes {
		volumeName := volume.(map[string]interface{})["name"].(string)
		if volumeName == "cloud-credentials" {
			return volume.(map[string]interface{})["secret"].(map[string]interface{})["secretName"].(string), nil
		}
	}

	return "", fmt.Errorf("could not find velero pod volume 'cloud-credentials'")
}

// SetupVelero sets up Velero in the destination Kubernetes cluster.
func SetupVelero(
	sourceHelmClient helm.Client,
	sourceDynamicClient dynamic.Interface,
	destinationHelmClient helm.Client,
	destinationDynamicClient dynamic.Interface,
	config *common.Config) {
	if config.SourceVeleroHelmReleaseName == "" {
		release, foundRelease := getHelmReleaseByShortName("velero", sourceHelmClient)
		if !foundRelease {
			log.Fatalf("Error: could not find the velero helm release installed in the source cluster, If It exists please specify it's name.")
		}
		config.SourceVeleroHelmReleaseName = release.Name
	}

	sourceVeleroRelease, err := sourceHelmClient.GetRelease(config.SourceVeleroHelmReleaseName)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	sourceHelmValuesMap, err := sourceHelmClient.GetReleaseValues(sourceVeleroRelease.Name, true)
	if err != nil {
		log.Fatalf("Error: could not get helm release %s values from source cluster. %v", sourceVeleroRelease.Name, err)
	}
	destinationHelmValues := sourceHelmValuesMap
	cloneVeleroHelmChart(destinationHelmClient, destinationHelmValues, *sourceVeleroRelease, config.DestinationVeleroNamespace)
}

// areMapsEqual checks if two maps are equal.
func areMapsEqual(map1, map2 map[string]interface{}) bool {
	if len(map1) != len(map2) {
		return false
	}

	for key, value1 := range map1 {
		if value2, ok := map2[key]; !ok || value1 != value2 {
			return false
		}
	}

	return true
}

// parseOrLabels parses the input map into a slice of map[string]map[string]string suitable for label selector.
func parseOrLabels(inputMap map[string]string) []map[string]map[string]string {
	var result []map[string]map[string]string

	for key, value := range inputMap {
		result = append(result, map[string]map[string]string{"matchLabels": {key: value}})
	}

	return result
}

// createResource creates a Kubernetes resource.
func createResource(dynamicClient dynamic.Interface, namespace string, resource *unstructured.Unstructured, r string) error {
	_, err := dynamicClient.Resource(schema.GroupVersionResource{
		Group:    resource.GroupVersionKind().Group,
		Version:  resource.GroupVersionKind().Version,
		Resource: r,
	}).Namespace(namespace).Create(context.TODO(), resource, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create resource: %v", err)
	}

	return nil
}
