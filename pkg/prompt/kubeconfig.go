package prompt

import (
	"errors"
	"fmt"
	"log"
	"os"
	common "vresq/pkg/common"

	"gopkg.in/yaml.v2"
)

// ChooseKubeconfigContext prompts the user to choose a context from the provided kubeconfig file.
// It returns the selected context name and any error encountered.
func ChooseKubeconfigContext(kubeconfigPath string, contextLabel string, config *common.Config) (string, error) {
	// Retrieve the list of contexts from the kubeconfig file
	contexts, err := getKubeconfigContexts(kubeconfigPath)
	if err != nil {
		return "", err
	}

	log.Printf("found %d context in %s", len(contexts), contextLabel)

	// If there is more than one context and the corresponding context in the configuration is empty,
	// prompt the user to choose a context.
	if len(contexts) > 1 {
		contextCondition := (contextLabel == "Source" && config.SourceContext == "") ||
			(contextLabel == "Destination" && config.DestinationContext == "")

		if contextCondition {
			return chooseContext(contexts, contextLabel)
		}
	} else if len(contexts) == 1 {
		// If there is only one context available, confirm its usage.
		log.Printf("Only one context available in %s", kubeconfigPath)
		return confirmSingleContext(contexts[0].Name, contextLabel)
	}

	// Return an empty string if no context needs to be chosen.
	return "", nil
}

// ChooseDestinationKubeconfig prompts the user to enter the path of the destination kubeconfig file.
// It returns the selected path and any error encountered.
func ChooseDestinationKubeconfig() (string, error) {
	label := "Destination Kubeconfig path: "
	validate := func(input string) error {
		if input == "" {
			return errors.New("kubeconfig path cannot be empty")
		}

		// Validate the provided kubeconfig file path
		if err := validateKubeconfig(input); err != nil {
			return err
		}

		return nil
	}

	// Prompt for user input with validation
	return promptWithValidate(label, validate, "kubeconfig", "")
}

// getKubeconfigContexts reads the kubeconfig file and returns a list of contexts.
func getKubeconfigContexts(kubeconfigPath string) ([]Context, error) {
	// Read the kubeconfig file
	kubeconfigData, err := os.ReadFile(kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read kubeconfig file: %v", err)
	}

	// Parse the kubeconfig YAML
	var kubeconfig Kubeconfig
	err = yaml.Unmarshal(kubeconfigData, &kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse kubeconfig YAML: %v", err)
	}

	// Return the list of contexts
	return kubeconfig.Contexts, nil
}

// confirmSingleContext prompts the user to confirm using a single context.
func confirmSingleContext(contextName, contextLabel string) (string, error) {
	confirmChoices := []string{ConfirmYes, ConfirmNo}
	promptLabel := fmt.Sprintf("Do you confirm using %s as %s context ?", contextName, contextLabel)
	return promptWithSelect(promptLabel, confirmChoices), nil
}

// validateKubeconfig validates the provided kubeconfig file path.
func validateKubeconfig(input string) error {
	contexts, err := getKubeconfigContexts(input)
	if err != nil {
		return fmt.Errorf("error validating kubeconfig: %v", err)
	}

	if len(contexts) == 0 {
		return errors.New("kubeconfig does not have any contexts")
	}

	return nil
}
