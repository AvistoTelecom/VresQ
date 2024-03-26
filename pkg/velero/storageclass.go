package velero

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	storageClassGVR = schema.GroupVersionResource{
		Group:    "storage.k8s.io",
		Version:  apiVersion,
		Resource: "storageclasses",
	}
)

// getDestinationDefaultStorageClass retrieves the name of the default storage class from the given list of unstructured storage classes.
// It returns an empty string if no default storage class is found.
func getDestinationDefaultStorageClass(storageClasses []unstructured.Unstructured) string {
	for _, storageClass := range storageClasses {
		// Check if the storage class is marked as default
		if storageClass.GetAnnotations()["storageclass.kubernetes.io/is-default-class"] == "true" {
			return storageClass.GetName()
		}
	}
	// No default storage class found
	return ""
}

// getSourceStorageClassNames retrieves the names of storage classes from the given list of unstructured storage classes.
func getSourceStorageClassNames(storageClasses []unstructured.Unstructured) []string {
	var storageClassNames []string
	for _, ssc := range storageClasses {
		// Append the name of each storage class to the list
		storageClassNames = append(storageClassNames, ssc.GetName())
	}
	return storageClassNames
}
