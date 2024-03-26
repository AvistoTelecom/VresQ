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

var (
	backupLocationGVR = schema.GroupVersionResource{
		Group:    veleroApiGroup,
		Version:  apiVersion,
		Resource: "backupstoragelocations",
	}
)

func SetupVeleroBackupLocation(sourceDynamicClient dynamic.Interface, destinationDynamicClient dynamic.Interface, config *common.Config) string {
	// Retrieve the backup from the source cluster
	backup, err := GetBackup(sourceDynamicClient, config.SourceVeleroNamespace, config.VeleroRestoreOptions.BackupName)
	if err != nil {
		log.Fatalf("Error: could not get backup, %v", err)
	}
	// Extract the storage location name from the backup
	backupStorageLocationName, _, err := unstructured.NestedString(backup.Object, "spec", "storageLocation")
	if err != nil {
		log.Fatalf("Error: could not read source backup storage location Name. %v", err)
	}
	// Get the source backup storage location
	sourceBackupLocation, err := getBackupStorageLocation(sourceDynamicClient, config.SourceVeleroNamespace, backupStorageLocationName)
	if err != nil {
		log.Fatalf("Error: could not get source backup location in source namespace, %v", err)
	}
	// List destination backup storage locations
	destinationBackupLocations, err := listBackupStorageLocations(destinationDynamicClient, config.DestinationVeleroNamespace)
	if err != nil {
		log.Fatalf("Error: could not list Backup storage locations in destination namespace, %v", err)
	}
	// Check if the destination backup storage location exists
	foundStorageLocation := findDestinationStorageLocation(&sourceBackupLocation, destinationBackupLocations.Items)
	if !foundStorageLocation {
		// If not found, create a new backup storage location in the destination cluster
		log.Printf("Did not find any backup storage location in destination cluster with source BackupStorageLocation: %s, creating one ...", sourceBackupLocation.GetName())
		sourceBucketName := sourceBackupLocation.UnstructuredContent()["spec"].(map[string]interface{})["objectStorage"].(map[string]interface{})["bucket"].(string)
		SetupDestinationBackupLocationSecret(sourceDynamicClient, destinationDynamicClient, &sourceBackupLocation, sourceBucketName, config)
		createVeleroBackupStorageLocation(destinationDynamicClient, config.DestinationVeleroNamespace, fmt.Sprintf("%s-readonly", sourceBucketName), sourceBackupLocation.UnstructuredContent()["spec"].(map[string]interface{}))
		groupVersionResource := schema.GroupVersionResource{
			Group:    veleroApiGroup,
			Version:  apiVersion,
			Resource: "backups",
		}
		// Wait for the backup to be available in the destination cluster
		backupReady, err := watchBackupWithTimeout(destinationDynamicClient, config.DestinationVeleroNamespace, config.VeleroRestoreOptions.BackupName, groupVersionResource, 5*time.Minute)
		if err != nil {
			log.Fatalf("Error waiting for backup %s to be available on destination cluster. %v", config.VeleroRestoreOptions.BackupName, err)
		}
		if !backupReady {
			log.Fatalf("Error: could not get backup %s on destination cluster", config.VeleroRestoreOptions.BackupName)
		}
		return fmt.Sprintf("%s-readonly", sourceBucketName)
	}
	return ""
}

// SetupDestinationBackupLocationSecret sets up the secret for the destination backup location.
func SetupDestinationBackupLocationSecret(sourceDynamicClient dynamic.Interface, destinationDynamicClient dynamic.Interface, sourceBackupLocation *unstructured.Unstructured, sourceBucketName string, config *common.Config) {
	destinationBackupLocationName := fmt.Sprintf("%s-readonly-credentials", sourceBucketName)
	sourceCreds, foundCreds, _ := unstructured.NestedMap(sourceBackupLocation.Object, "spec", "credential")
	if foundCreds {
		handleExistingCredentials(sourceDynamicClient, destinationDynamicClient, config, sourceBackupLocation, sourceCreds, destinationBackupLocationName)
	} else {
		handleNoCredentials(sourceDynamicClient, destinationDynamicClient, config, sourceBackupLocation, destinationBackupLocationName)
	}
}

// handleExistingCredentials handles the case where credentials are set on the source BackupStorageLocation level. It will clone creds as a secret in destination
func handleExistingCredentials(sourceDynamicClient dynamic.Interface, destinationDynamicClient dynamic.Interface, config *common.Config, sourceBackupLocation *unstructured.Unstructured, sourceCreds map[string]interface{}, destinationBackupLocationName string) {
	// Extract the name of the secret containing credentials from the source backup location
	secretName, foundSecretName, _ := unstructured.NestedString(sourceCreds, "name")
	if !foundSecretName {
		log.Fatalf("Error: could not read secret name in source backup location.")
	}
	// Retrieve the secret from the source cluster
	secret, err := GetSecret(sourceDynamicClient, sourceBackupLocation.GetNamespace(), secretName)
	if err != nil {
		log.Fatalf("Error: could not read source BackupStorageLocation secret. %v", err)
	}
	// Ensure the secret exists in the destination cluster
	err = EnsureSecret(destinationDynamicClient, config.DestinationVeleroNamespace, destinationBackupLocationName, secret)
	if err != nil {
		log.Fatalf("Error: Could not create secret for BackupStorageLocation in destination cluster. %v", err)
	}
	// Extract the key from the source credentials
	secretKey, _, _ := unstructured.NestedString(sourceCreds, "key")
	credentialSpec := map[string]string{
		"name": destinationBackupLocationName,
		"key":  secretKey,
	}
	unstructured.SetNestedField(sourceBackupLocation.Object, credentialSpec, "spec", "credential")
}

// handleNoCredentials handles the case where credentials are NOT set on the source BackupStorageLocation level. It will clone global creds set at velero instance as a secret in destination
func handleNoCredentials(sourceDynamicClient dynamic.Interface, destinationDynamicClient dynamic.Interface, config *common.Config, sourceBackupLocation *unstructured.Unstructured, destinationBackupLocationName string) {
	// Retrieve the Velero pod in the source cluster
	veleroPod, err := GetVeleroPod(sourceDynamicClient)
	if err != nil {
		log.Fatalf("Error: could not get velero pod in source cluster. %v", err)
	}
	// Get the name of the secret used by Velero
	veleroSecretName, err := getVeleroPodSecretName(&veleroPod)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
	// Retrieve the secret from the source cluster
	secret, err := GetSecret(sourceDynamicClient, veleroPod.GetNamespace(), veleroSecretName)
	if err != nil {
		log.Fatalf("Error: could not retrieve velero Pod secret in source cluster. %v", err)
	}
	// Ensure the secret exists in the destination cluster
	err = EnsureSecret(destinationDynamicClient, config.DestinationVeleroNamespace, destinationBackupLocationName, secret)
	if err != nil {
		log.Fatalf("Error: Could not create secret for BackupStorageLocation in destination cluster. %v", err)
	}
	// Set up the credential specification for the destination backup location
	credentialSpec := map[string]interface{}{
		"name": destinationBackupLocationName,
		"key":  "cloud",
	}
	unstructured.SetNestedField(sourceBackupLocation.Object, credentialSpec, "spec", "credential")
}

// findDestinationStorageLocation checks if there is a BackupStorageLocation with the same specs as the source one in destination cluster.
func findDestinationStorageLocation(sourceBackupLocation *unstructured.Unstructured, destinationBackupLocations []unstructured.Unstructured) bool {
	foundStorageLocation := false
	for _, destinationBackupLocation := range destinationBackupLocations {
		// Check if configurations match
		configCheck := areMapsEqual(
			destinationBackupLocation.UnstructuredContent()["spec"].(map[string]interface{})["config"].(map[string]interface{}),
			sourceBackupLocation.UnstructuredContent()["spec"].(map[string]interface{})["config"].(map[string]interface{}),
		)
		// Check if object storage match
		objectStorageCheck := areMapsEqual(
			destinationBackupLocation.UnstructuredContent()["spec"].(map[string]interface{})["objectStorage"].(map[string]interface{}),
			sourceBackupLocation.UnstructuredContent()["spec"].(map[string]interface{})["objectStorage"].(map[string]interface{}),
		)
		// Check if the status phase is "Available"
		availabilityCheck := sourceBackupLocation.UnstructuredContent()["status"].(map[string]interface{})["phase"].(string) == "Available"
		// If all checks pass, set foundStorageLocation to true
		if configCheck && objectStorageCheck && availabilityCheck {
			foundStorageLocation = true
		}
	}
	return foundStorageLocation
}

// getBackupStorageLocation retrieves the backup storage location.
func getBackupStorageLocation(dynamicClient dynamic.Interface, namespace string, name string) (unstructured.Unstructured, error) {
	groupVersionResource := backupLocationGVR
	backup, err := dynamicClient.Resource(groupVersionResource).Namespace(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return unstructured.Unstructured{}, err
	}
	return *backup, nil
}

// listBackupStorageLocations lists all backup storage locations.
func listBackupStorageLocations(dynamicClient dynamic.Interface, namespace string) (*unstructured.UnstructuredList, error) {
	groupVersionResource := backupLocationGVR
	backupStorageLocations, err := dynamicClient.Resource(groupVersionResource).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return backupStorageLocations, nil
}

// createVeleroBackupStorageLocation creates a new Velero backup storage location.
func createVeleroBackupStorageLocation(dynamicClient dynamic.Interface, namespace string, name string, spec map[string]interface{}) error {
	backupStorageLocation := unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": fmt.Sprintf("%s/%s", veleroApiGroup, apiVersion),
			"kind":       "BackupStorageLocation",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"spec": spec,
		},
	}
	//Every BackupStorageLocation created by the program, should be readonly and not default.
	spec["default"] = false         //defalut: true is not needed when making a restore
	spec["accessMode"] = "ReadOnly" //make sure backups cannot be altered on the source cluster by mistake
	err := createResource(dynamicClient, namespace, &backupStorageLocation, "backupstoragelocations")
	if err != nil {
		return err
	} else {
		log.Println("BackupStorageLocation created successfully")
	}

	return nil
}
