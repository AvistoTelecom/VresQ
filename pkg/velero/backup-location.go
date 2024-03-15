package velero

import (
	"context"
	"fmt"
	"log"
	"time"
	common "vresq/pkg/common"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

func SetupVeleroBackupLocation(sourceDynamicClient dynamic.Interface, destinationDynamicClient dynamic.Interface, config *common.Config) {
	backup, err := GetBackup(sourceDynamicClient, config.SourceVeleroNamespace, config.VeleroRestoreOptions.BackupName)
	if err != nil {
		log.Fatalf("Error: could not get backup, %v", err)
	}
	backupStorageLocationName, _, err := unstructured.NestedString(backup.Object, "spec", "storageLocation")
	if err != nil {
		log.Fatalf("Error: could not read source backup storage location Name. %v", err)
	}
	sourceBackupLocation, err := getBackupStorageLocation(sourceDynamicClient, config.SourceVeleroNamespace, backupStorageLocationName)
	if err != nil {
		log.Fatalf("Error: could not get source backup location in source namespace, %v", err)
	}
	destinationBackupLocations, err := listBackupStorageLocations(destinationDynamicClient, config.DestinationVeleroNamespace)
	if err != nil {
		log.Fatalf("Error: could not list Backup storage locations in destination namespace, %v", err)
	}
	foundStorageLocation := findDestinationStorageLocation(&sourceBackupLocation, destinationBackupLocations.Items)
	if !foundStorageLocation {
		log.Printf("Did not find any backup storage location in destination cluster with source BackupStorageLocation: %s, creating one ...", sourceBackupLocation.GetName())
		sourceBucketName := sourceBackupLocation.UnstructuredContent()["spec"].(map[string]interface{})["objectStorage"].(map[string]interface{})["bucket"].(string)
		SetupDestinationBackupLocationSecret(sourceDynamicClient, destinationDynamicClient, &sourceBackupLocation, sourceBucketName, config)
		createVeleroBackupStorageLocation(destinationDynamicClient, config.DestinationVeleroNamespace, fmt.Sprintf("%s-readonly", sourceBucketName), sourceBackupLocation.UnstructuredContent()["spec"].(map[string]interface{}))
		groupVersionResource := schema.GroupVersionResource{
			Group:    veleroApiGroup,
			Version:  apiVersion,
			Resource: "backups",
		}
		backupReady, err := watchBackupWithTimeout(destinationDynamicClient, config.DestinationVeleroNamespace, config.VeleroRestoreOptions.BackupName, groupVersionResource, 5*time.Minute)
		if err != nil {
			log.Fatalf("Error waiting for backup %s to be available on destination cluster. %v", config.VeleroRestoreOptions.BackupName, err)
		}
		if !backupReady {
			log.Fatalf("Error: could not get backup %s on destination cluster", config.VeleroRestoreOptions.BackupName)
		}
	}
}

func SetupDestinationBackupLocationSecret(sourceDynamicClient dynamic.Interface, destinationDynamicClient dynamic.Interface, sourceBackupLocation *unstructured.Unstructured, sourceBucketName string, config *common.Config) {
	destinationBackupLocationName := fmt.Sprintf("%s-readonly-credentials", sourceBucketName)
	setSourceBackupLocationFields(sourceBackupLocation, destinationBackupLocationName)

	sourceCreds, foundCreds, _ := unstructured.NestedMap(sourceBackupLocation.Object, "spec", "credential")
	if foundCreds {
		handleExistingCredentials(sourceDynamicClient, destinationDynamicClient, config, sourceBackupLocation, sourceCreds, destinationBackupLocationName)
	} else {
		handleNoCredentials(sourceDynamicClient, destinationDynamicClient, config, sourceBackupLocation, destinationBackupLocationName)
	}
}

func handleExistingCredentials(sourceDynamicClient dynamic.Interface, destinationDynamicClient dynamic.Interface, config *common.Config, sourceBackupLocation *unstructured.Unstructured, sourceCreds map[string]interface{}, destinationBackupLocationName string) {
	secretName, foundSecretName, _ := unstructured.NestedString(sourceCreds, "name")
	if !foundSecretName {
		log.Fatalf("Error: could not read secret name in source backup location.")
	}

	secret, err := GetSecret(sourceDynamicClient, sourceBackupLocation.GetNamespace(), secretName)
	if err != nil {
		log.Fatalf("Error: could not read source BackupStorageLocation secret. %v", err)
	}

	err = CreateSecret(destinationDynamicClient, config.DestinationVeleroNamespace, destinationBackupLocationName, secret)
	if err != nil {
		log.Fatalf("Error: Could not create secret for BackupStorageLocation in destination cluster. %v", err)
	}

	secretKey, _, _ := unstructured.NestedString(sourceCreds, "key")
	credentialSpec := map[string]string{
		"name": destinationBackupLocationName,
		"key":  secretKey,
	}
	unstructured.SetNestedField(sourceBackupLocation.Object, credentialSpec, "spec", "credential")
}

func handleNoCredentials(sourceDynamicClient dynamic.Interface, destinationDynamicClient dynamic.Interface, config *common.Config, sourceBackupLocation *unstructured.Unstructured, destinationBackupLocationName string) {
	veleroPod, err := GetVeleroPod(sourceDynamicClient)
	if err != nil {
		log.Fatalf("Error: could not get velero pod in source cluster. %v", err)
	}

	veleroSecretName, err := getVeleroPodSecretName(&veleroPod)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	secret, err := GetSecret(sourceDynamicClient, veleroPod.GetNamespace(), veleroSecretName)
	if err != nil {
		log.Fatalf("Error: could not retrieve velero Pod secret in source cluster. %v", err)
	}

	err = CreateSecret(destinationDynamicClient, config.DestinationVeleroNamespace, destinationBackupLocationName, secret)
	if err != nil {
		log.Fatalf("Error: Could not create secret for BackupStorageLocation in destination cluster. %v", err)
	}
	credentialSpec := map[string]string{
		"name": destinationBackupLocationName,
		"key":  "cloud",
	}
	unstructured.SetNestedField(sourceBackupLocation.Object, credentialSpec, "spec", "credential")
}

func setSourceBackupLocationFields(sourceBackupLocation *unstructured.Unstructured, destinationBackupLocationName string) {
	unstructured.SetNestedField(sourceBackupLocation.Object, "ReadOnly", "spec", "accessMode")
	unstructured.SetNestedField(sourceBackupLocation.Object, true, "spec", "default")
}

func findDestinationStorageLocation(sourceBackupLocation *unstructured.Unstructured, destinationBackupLocations []unstructured.Unstructured) bool {
	foundStorageLocation := false
	for _, destinationBackupLocation := range destinationBackupLocations {
		configCheck := areMapsEqual(
			destinationBackupLocation.UnstructuredContent()["spec"].(map[string]interface{})["config"].(map[string]interface{}),
			sourceBackupLocation.UnstructuredContent()["spec"].(map[string]interface{})["config"].(map[string]interface{}),
		)
		objectStorageCheck := areMapsEqual(
			destinationBackupLocation.UnstructuredContent()["spec"].(map[string]interface{})["objectStorage"].(map[string]interface{}),
			sourceBackupLocation.UnstructuredContent()["spec"].(map[string]interface{})["objectStorage"].(map[string]interface{}),
		)
		availabilityCheck := sourceBackupLocation.UnstructuredContent()["status"].(map[string]interface{})["phase"].(string) == "Available"
		if configCheck && objectStorageCheck && availabilityCheck {
			foundStorageLocation = true
		}
	}
	return foundStorageLocation
}

func getBackupStorageLocation(dynamicClient dynamic.Interface, namespace string, name string) (unstructured.Unstructured, error) {
	// Create a GVR which represents an Istio Virtual Service.
	groupVersionResource := schema.GroupVersionResource{
		Group:    veleroApiGroup,
		Version:  apiVersion,
		Resource: "backupstoragelocations",
	}

	// List all of the Virtual Services.
	backup, err := dynamicClient.Resource(groupVersionResource).Namespace(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return unstructured.Unstructured{}, err
	}

	return *backup, nil
}

func listBackupStorageLocations(dynamicClient dynamic.Interface, namespace string) (*unstructured.UnstructuredList, error) {

	// Create a GVR which represents an Istio Virtual Service.
	groupVersionResource := schema.GroupVersionResource{
		Group:    veleroApiGroup,
		Version:  apiVersion,
		Resource: "backupstoragelocations",
	}

	// List all of the Virtual Services.
	backupStorageLocations, err := dynamicClient.Resource(groupVersionResource).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return backupStorageLocations, nil
}

func createVeleroBackupStorageLocation(dynamicClient dynamic.Interface, namespace string, name string, spec map[string]interface{}) error {
	backupStorageLocation := unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "velero.io/v1",
			"kind":       "BackupStorageLocation",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"spec": spec,
		},
	}

	err := createResource(dynamicClient, namespace, &backupStorageLocation, "backupstoragelocations")
	if err != nil {
		return err
	} else {
		log.Println("BackupStorageLocation created successfully")
	}

	return nil
}
