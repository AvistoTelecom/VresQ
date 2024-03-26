package velero

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

func EnsureSecret(dynamicClient dynamic.Interface, namespace, secretName string, data map[string]string) error {
	secretsResource := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"}
	secrets, err := dynamicClient.Resource(secretsResource).Namespace(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("could not list secrets: %v", err)
	}
	found := false
	for _, secret := range secrets.Items {
		if secret.GetName() == secretName {
			found = true
			break
		}
	}
	if found {
		return nil
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
	_, err = dynamicClient.Resource(secretsResource).Namespace(namespace).Create(context.TODO(), secretObj, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	return nil
}

func GetSecret(dynamicClient dynamic.Interface, namespace, secretName string) (map[string]string, error) {
	secretsResource := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "secrets"}

	// Retrieve the secret
	secret, err := dynamicClient.Resource(secretsResource).Namespace(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	data, found, err := unstructured.NestedStringMap(secret.Object, "data")
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, fmt.Errorf("data not found in the secret")
	}

	return data, nil
}
