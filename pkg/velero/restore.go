package velero

import (
	"context"
	"fmt"
	"log"
	"strings"
	common "vresq/pkg/common"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/dynamic"
)

func CreateVeleroRestore(dynamicClient dynamic.Interface, namespace string, name string, options common.VeleroRestoreOptions) error {
	restore := unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "velero.io/v1",
			"kind":       "Restore",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"backupName":              options.BackupName,
				"scheduleName":            options.ScheduleName,
				"itemOperationTimeout":    options.ItemOperationTimeout.String(),
				"includedNamespaces":      options.IncludedNamespaces,
				"excludedNamespaces":      options.ExcludedNamespaces,
				"includedResources":       options.IncludedResources,
				"excludedResources":       options.ExcludedResources,
				"includeClusterResources": options.IncludeClusterResources,
				"labelSelector": map[string]interface{}{
					"matchLabels": options.LabelSelector,
				},
				"orLabelSelectors":       parseOrLabels(options.OrLabelSelectors),
				"namespaceMapping":       options.NamespaceMapping,
				"restorePVs":             options.RestorePVs,
				"preserveNodePorts":      options.PreserveNodePorts,
				"existingResourcePolicy": options.ExistingResourcePolicy,
			},
		},
	}

	groupVersionResource := schema.GroupVersionResource{
		Group:    veleroApiGroup,
		Version:  apiVersion,
		Resource: "restores",
	}

	err := createResource(dynamicClient, namespace, &restore, "restores")
	if err != nil {
		log.Fatalf("Error creating Velero Resource: %v", err)
		return err
	} else {
		log.Println("Velero Restore created successfully")
	}

	// Watch the restore until it's completed or fails
	err = watchRestore(dynamicClient, namespace, name, groupVersionResource)
	if err != nil {
		log.Fatalf("Error watching restore: %v", err)
	}

	return nil
}

// watchRestore watches the restore with the given name in the specified namespace
// until its status is completed or fails. It returns an error if the restore fails.
func watchRestore(dynamicClient dynamic.Interface, namespace, restoreName string, veleroRestoreGVR schema.GroupVersionResource) error {
	log.Printf("Watching restore '%s' in namespace '%s'\n", restoreName, namespace)

	// Set up the watch interface
	watcher, err := dynamicClient.Resource(veleroRestoreGVR).Namespace(namespace).Watch(context.TODO(), metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", restoreName),
	})
	if err != nil {
		return fmt.Errorf("failed to watch restore: %v", err)
	}
	defer watcher.Stop()

	// Define a channel to signal the end of watching
	stopCh := make(chan struct{})

	// Watch for changes in a separate goroutine
	go watchChanges(dynamicClient, namespace, restoreName, veleroRestoreGVR, stopCh, watcher)

	// Wait for the watch to be stopped (when status is completed or failed)
	<-stopCh

	// Check the final status of the restore
	return checkFinalStatus(dynamicClient, namespace, restoreName, veleroRestoreGVR)
}

func watchChanges(dynamicClient dynamic.Interface, namespace, restoreName string, veleroRestoreGVR schema.GroupVersionResource, stopCh chan struct{}, watcher watch.Interface) {
	for event := range watcher.ResultChan() {
		restore, isUnstructured := event.Object.(*unstructured.Unstructured)
		if !isUnstructured {
			log.Println("Unexpected object type received from watch event")
			continue
		}

		// Extract the status phase from the restore
		statusPhase, found, err := unstructured.NestedString(restore.Object, "status", "phase")
		if err != nil || !found {
			log.Println("Failed to get restore status phase")
			continue
		}

		log.Printf("Restore status: %s\n", statusPhase)

		// Check if the status is completed or failed
		if statusPhase == "Completed" || strings.Contains(strings.ToLower(statusPhase), "failed") {
			close(stopCh)
			return
		}
	}
}

func checkFinalStatus(dynamicClient dynamic.Interface, namespace, restoreName string, veleroRestoreGVR schema.GroupVersionResource) error {
	// Check the final status of the restore
	finalRestore, err := dynamicClient.Resource(veleroRestoreGVR).Namespace(namespace).Get(context.TODO(), restoreName, metav1.GetOptions{})
	if err != nil {
		log.Printf("Failed to get final restore status: %v\n", err)
		return fmt.Errorf("failed to get final restore status: %v", err)
	}

	finalStatusPhase, found, err := unstructured.NestedString(finalRestore.Object, "status", "phase")
	if err != nil || !found {
		log.Println("Failed to get final restore status phase")
		return fmt.Errorf("failed to get final restore status phase")
	}

	log.Printf("Final restore status: %s\n", finalStatusPhase)

	// Check if the status is completed or failed
	if finalStatusPhase == "Completed" {
		return nil
	} else if finalStatusPhase == "Failed" {
		log.Printf("Restore failed\n")
		return fmt.Errorf("restore failed")
	}

	log.Printf("Restore status is not completed or failed\n")
	return fmt.Errorf("restore status is not completed or failed")
}
