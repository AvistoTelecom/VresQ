package velero

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

func getDestinationDefaultStorageClass(storageClasses []unstructured.Unstructured) string {
	for _, storageClass := range storageClasses {
		if storageClass.GetAnnotations()["storageclass.kubernetes.io/is-default-class"] == "true" {
			return storageClass.GetName()
		}
	}
	return ""
}

func getSourceStorageClassNames(storageClasses []unstructured.Unstructured) []string {
	var storageClassNames []string
	for _, ssc := range storageClasses {
		storageClassNames = append(storageClassNames, ssc.GetName())
	}
	return storageClassNames
}
