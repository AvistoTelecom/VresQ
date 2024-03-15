package prompt

import (
	"errors"
	"fmt"
	"log"
	"os"
	common "vresq/pkg/common"

	"gopkg.in/yaml.v2"
)

func ChooseKubeconfigContext(kubeconfigPath string, contextLabel string, config *common.Config) (string, error) {
	contexts, err := getKubeconfigContexts(kubeconfigPath)
	if err != nil {
		return "", err
	}

	log.Printf("found %d context in %s", len(contexts), contextLabel)

	if len(contexts) > 1 {
		contextCondition := (contextLabel == "Source" && config.SourceContext == "") ||
			(contextLabel == "Destination" && config.DestinationContext == "")

		if contextCondition {
			return chooseContext(contexts, contextLabel)
		}
	} else if len(contexts) == 1 {
		log.Printf("Only one context available in %s", kubeconfigPath)
		return confirmSingleContext(contexts[0].Name, contextLabel)
	}

	return "", nil
}

func ChooseDestinationKubeconfig() (string, error) {
	label := "Destination Kubeconfig path: "
	validate := func(input string) error {
		if input == "" {
			return errors.New("kubeconfig path cannot be empty")
		}

		if err := validateKubeconfig(input); err != nil {
			return err
		}

		return nil
	}

	return promptWithValidate(label, validate, "kubeconfig", "")
}

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

func confirmSingleContext(contextName, contextLabel string) (string, error) {
	confirmChoices := []string{ConfirmYes, ConfirmNo}
	promptLabel := fmt.Sprintf("Do you confirm using %s as %s context ?", contextName, contextLabel)
	return promptWithSelect(promptLabel, confirmChoices), nil
}

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
