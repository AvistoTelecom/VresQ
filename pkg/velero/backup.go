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

func watchBackupWithTimeout(dynamicClient dynamic.Interface, namespace, backupName string, veleroBackupGVR schema.GroupVersionResource, timeout time.Duration) (bool, error) {
	log.Printf("Watching backup '%s' in namespace '%s' with timeout %v\n", backupName, namespace, timeout)

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Set up the list interface
	listOptions := metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", backupName),
	}

	// Check if the desired backup is already present
	backups, err := dynamicClient.Resource(veleroBackupGVR).Namespace(namespace).List(ctx, listOptions)
	if err != nil {
		return false, fmt.Errorf("failed to list backups: %v", err)
	}

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

func GetBackup(dynamicClient dynamic.Interface, namespace string, name string) (unstructured.Unstructured, error) {
	// Create a GVR which represents an Istio Virtual Service.
	groupVersionResource := schema.GroupVersionResource{
		Group:    veleroApiGroup,
		Version:  apiVersion,
		Resource: "backups",
	}

	// List all of the Virtual Services.
	backup, err := dynamicClient.Resource(groupVersionResource).Namespace(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return unstructured.Unstructured{}, err
	}

	return *backup, nil
}

func ListBackups(dynamicClient dynamic.Interface, namespace string) (*unstructured.UnstructuredList, error) {

	// Create a GVR which represents an Istio Virtual Service.
	groupVersionResource := schema.GroupVersionResource{
		Group:    veleroApiGroup,
		Version:  apiVersion,
		Resource: "backups",
	}

	// List all of the Virtual Services.
	backups, err := dynamicClient.Resource(groupVersionResource).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	return backups, nil

}
