package velero

import (
	"context"
	"fmt"
	"log"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

var (
	backupGVR = schema.GroupVersionResource{
		Group:    veleroApiGroup,
		Version:  apiVersion,
		Resource: "backups",
	}
)

// watchBackupWithTimeout watches a Velero backup with a timeout period.
func watchBackupWithTimeout(dynamicClient dynamic.Interface, namespace, backupName string, veleroBackupGVR schema.GroupVersionResource, timeout time.Duration) (bool, error) {
	log.Printf("Watching backup '%s' in namespace '%s' with timeout %v\n", backupName, namespace, timeout)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Set up list options to filter backups by name
	listOptions := metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", backupName),
	}

	// Check if the desired backup already exists
	backups, err := dynamicClient.Resource(veleroBackupGVR).Namespace(namespace).List(ctx, listOptions)
	if err != nil {
		return false, fmt.Errorf("failed to list backups: %v", err)
	}

	// If the backup exists, return true immediately
	if backupExists(backups.Items, backupName) {
		log.Printf("Backup '%s' found without waiting\n", backupName)
		return true, nil
	}

	// Poll for changes with a specified interval
	pollInterval := 5 * time.Second
	timeoutTimer := time.NewTimer(timeout)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Timeout reached (%v).", timeout)
			return false, nil
		case <-timeoutTimer.C:
			log.Printf("Timeout reached (%v).", timeout)
			return false, nil
		default:
			// List available backups
			backups, err := dynamicClient.Resource(veleroBackupGVR).Namespace(namespace).List(ctx, listOptions)
			if err != nil {
				return false, fmt.Errorf("failed to list backups: %v", err)
			}

			// Check if the desired backup is now present
			if backupExists(backups.Items, backupName) {
				log.Printf("Backup '%s' found within the timeout\n", backupName)
				return true, nil
			}

			// Sleep before polling again
			time.Sleep(pollInterval)
		}
	}
}

// backupExists checks if the desired backup is present in the list.
func backupExists(backups []unstructured.Unstructured, backupName string) bool {
	for _, backup := range backups {
		if backup.GetName() == backupName {
			return true
		}
	}
	return false
}

// GetBackup retrieves a Velero backup by name from the specified namespace.
func GetBackup(dynamicClient dynamic.Interface, namespace string, name string) (unstructured.Unstructured, error) {
	backup, err := dynamicClient.Resource(backupGVR).Namespace(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return unstructured.Unstructured{}, err
	}

	return *backup, nil
}

// ListBackups lists all Velero backups in the specified namespace.
func ListBackups(dynamicClient dynamic.Interface, namespace string) (*unstructured.UnstructuredList, error) {
	backups, err := dynamicClient.Resource(backupGVR).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return backups, nil

}
