package prompt

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	common "vresq/pkg/common"
	velero "vresq/pkg/velero"

	"github.com/manifoldco/promptui"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
)

func ChooseRestoreName() (string, error) {
	regex := regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
	validationError := "restore name should match the regex: '^[a-z0-9]([-a-z0-9]*[a-z0-9])?$'"
	label := "Restore name :"
	validate := func(input string) error {
		str := strings.ReplaceAll(input, label, "")
		match := regex.MatchString(str)
		if match && !strings.Contains(str, " ") {
			return nil
		} else {
			return errors.New(validationError)
		}
	}
	now := time.Now()
	t := now.Format("2006-01-02-15-04")

	templates := &promptui.PromptTemplates{
		Prompt:          prompt,
		Valid:           greenCheckmark + toCyan,
		Invalid:         redCross + toRed,
		Success:         toRed,
		ValidationError: validationError,
	}

	prompt := promptui.Prompt{
		Label:     label,
		Validate:  validate,
		Templates: templates,
		Default:   fmt.Sprintf("restore-%s", t),
		AllowEdit: true,
	}
	result, err := prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return "", err
	}
	return result, nil
}

func ChooseNamespaces(dynamiClient *dynamic.DynamicClient, config *common.Config) ([]string, error) {
	backup, err := velero.GetBackup(dynamiClient, config.SourceVeleroNamespace, config.VeleroRestoreOptions.BackupName)
	if err != nil {
		return nil, err
	}
	includedNamespaces, _, err := unstructured.NestedStringSlice(backup.Object, "spec", "includedNamespaces")
	if err != nil {
		return nil, err
	}
	items := []*item{}
	for _, ns := range includedNamespaces {
		items = append(items, &item{ID: ns, IsSelected: false})
	}
	result := []string{}
	selectedItems, err := selectItems(0, items, "Please choose a namespace")
	if err != nil {
		return nil, err
	}
	if len(selectedItems) == 0 {
		return nil, fmt.Errorf("no namespaces were selected")
	}
	for _, selectedNs := range selectedItems {
		result = append(result, selectedNs.ID)
		config.VeleroRestoreOptions.IncludedNamespaces = append(config.VeleroRestoreOptions.IncludedNamespaces, selectedNs.ID)
	}
	return result, nil
}

func ChooseBackup(sourceDynamiClient *dynamic.DynamicClient, config common.Config) (string, error) {
	backups, err := velero.ListBackups(sourceDynamiClient, config.SourceVeleroNamespace)
	if err != nil {
		return "", err
	}
	if len(backups.Items) == 0 {
		return "", err
	}

	// Create a slice to store backup details
	var backupDetails []BackupInfo
	for _, backup := range backups.Items {
		// Populate BackupInfo struct with relevant details
		// Parse the timestamp string
		timestamp, err := time.Parse(time.RFC3339, backup.UnstructuredContent()["status"].(map[string]interface{})["completionTimestamp"].(string))
		if err != nil {
			return "", fmt.Errorf("error parsing timestamp: %v", err)
		}
		completionTimestamp := timestamp.Format("02-01-2006 15:04:05")
		info := BackupInfo{
			Name:                backup.GetName(),
			Status:              backup.UnstructuredContent()["status"].(map[string]interface{})["phase"].(string),
			CompletionTimestamp: completionTimestamp,
			IncludedNamespaces:  backup.UnstructuredContent()["spec"].(map[string]interface{})["includedNamespaces"].([]interface{}),
		}
		backupDetails = append(backupDetails, info)
	}

	// Define a custom template for the prompt to show additional details
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}?",
		Active:   antiClockWiseEmoji + " {{ .Name | cyan }}",
		Inactive: "  {{ .Name | cyan }}",
		Selected: antiClockWiseEmoji + " {{ .Name | green }}",
		Details: `
	--------- Backup ----------
	{{ "Name:" | faint }}	{{ .Name }}
	{{ "Status:" | faint }}	{{ .Status }}
	{{ "CompletionTimestamp:" | faint }}	{{ .CompletionTimestamp }}
	{{ "IncludedNamespaces:" | faint }}	{{ .IncludedNamespaces }}`,
	}

	searcher := func(input string, index int) bool {
		backup := backupDetails[index]
		includesNamespaces := backup.IncludedNamespaces
		input = strings.Replace(strings.ToLower(input), " ", "", -1)
		found := false
		for _, namespace := range includesNamespaces {
			found = strings.Contains(namespace.(string), input)
		}
		return found
	}

	// Prompt user to choose a backup
	prompt := promptui.Select{
		Label:     "Please choose a backup",
		Items:     backupDetails,
		Templates: templates,
		Size:      10,
		Searcher:  searcher,
	}

	// Get the selected BackupInfo struct
	i, _, err := prompt.Run()
	if err != nil {
		return "", fmt.Errorf("prompt failed: %v", err)
	}

	result := backupDetails[i].Name
	return result, nil
}

func chooseContext(contexts []Context, contextLabel string) (string, error) {
	templates := getSelectTemplates()

	searcher := func(input string, index int) bool {
		context := contexts[index]
		return strings.Contains(context.Name, input)
	}

	prompt := promptui.Select{
		Label:     fmt.Sprintf("Please choose a %s Context", contextLabel),
		Items:     contexts,
		Templates: templates,
		Size:      10,
		Searcher:  searcher,
	}

	i, _, err := prompt.Run()
	if err != nil {
		return "", err
	}

	return contexts[i].Name, nil
}
