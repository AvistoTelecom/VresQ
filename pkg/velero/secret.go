package velero

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

var (
	secretGVR = schema.GroupVersionResource{
		Group:    "",
		Version:  "v1",
		Resource: "secrets",
	}
)

// EnsureSecret ensures that a Secret with the specified name and data exists in the given namespace.
// It creates the Secret if it doesn't already exist.
func EnsureSecret(dynamicClient dynamic.Interface, namespace, secretName string, data map[string]string) error {
	// List existing secrets in the namespace
	secrets, err := dynamicClient.Resource(secretGVR).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("could not list secrets: %v", err)
	}

	// Check if the secret already exists
	for _, secret := range secrets.Items {
		if secret.GetName() == secretName {
			// Secret already exists, no action needed
			return nil
		}
	}

	// Create the Secret object
	secretObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Secret",
			"metadata": map[string]interface{}{
				"name":      secretName,
				"namespace": namespace,
			},
			"data": data,
		},
	}

	// Create the Secret in the cluster
	_, err = dynamicClient.Resource(secretGVR).Namespace(namespace).Create(context.TODO(), secretObj, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}

// GetSecret retrieves the data of a Secret with the specified name in the given namespace.
// It returns the data map[string]string of the Secret.
func GetSecret(dynamicClient dynamic.Interface, namespace, secretName string) (map[string]string, error) {
	// Retrieve the secret
	secret, err := dynamicClient.Resource(secretGVR).Namespace(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	// Extract the data from the secret
	data, found, err := unstructured.NestedStringMap(secret.Object, "data")
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("data not found in the secret")
	}

	return data, nil
}
