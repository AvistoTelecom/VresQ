package prompt

import (
	"errors"
	"fmt"
	"log"
	"regexp"
	"strings"
	common "vresq/pkg/common"

	"github.com/manifoldco/promptui"
)

const (
	ConfirmYes         = common.ConfirmYes
	ConfirmNo          = common.ConfirmNo
	redCross           = "\u2717"
	greenCheckmark     = "\u2714"
	antiClockWiseEmoji = "\U0001F504"
	prompt             = "{{ . }} "
	toCyan             = " {{ . | cyan }} "
	toRed              = " {{ . | red }} "
	toBold             = "{{ . | bold }} "
)

type Context struct {
	Context struct {
		Cluster string `yaml:"cluster"`
		User    string `yaml:"user"`
	} `yaml:"context"`
	Name string `yaml:"name"`
}

type Kubeconfig struct {
	Contexts []Context `yaml:"contexts"`
}

type BackupInfo struct {
	Name                string
	Status              string
	CompletionTimestamp string
	IncludedNamespaces  []interface{}
}

type item struct {
	ID         string
	IsSelected bool
}

func ConfirmUserChoice(label string) bool {
	return promptWithSelect(fmt.Sprintf("%s?", label), []string{ConfirmYes, ConfirmNo}) == ConfirmYes
}

func promptWithSelect(label string, choices []string) string {
	prompt := promptui.Select{
		Label: label,
		Items: choices,
		Size:  2,
	}

	_, selected, err := prompt.Run()
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	return selected
}

func promptWithValidate(label string, validate func(string) error, entity string, defaultValue string) (string, error) {
	templates := getPromptTemplates()

	prompt := promptui.Prompt{
		Label:     label,
		Validate:  validate,
		Templates: templates,
		Default:   defaultValue,
		AllowEdit: true,
	}

	result, err := prompt.Run()
	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return "", err
	}

	return result, nil
}

func UserInput(regex *regexp.Regexp, validationError string, label string, defaultValue string) (string, error) {
	// regex := regexp.MustCompile(`[a-z0-9]([-a-z0-9]*[a-z0-9])?`)
	// validationError := "namespace name should match the regex: '[a-z0-9]([-a-z0-9]*[a-z0-9])?'"
	validate := func(input string) error {
		str := strings.ReplaceAll(input, label, "")
		match := regex.MatchString(str)
		if match && !strings.Contains(str, " ") {
			return nil
		} else {
			return errors.New(validationError)
		}
	}

	templates := &promptui.PromptTemplates{
		Prompt:          prompt,
		Valid:           greenCheckmark + toCyan,
		Invalid:         redCross + toRed,
		Success:         toRed,
		ValidationError: validationError,
	}

	prompt := promptui.Prompt{
		Label:     label,
		Templates: templates,
		Validate:  validate,
		Default:   defaultValue,
		AllowEdit: true,
	}

	result, err := prompt.Run()
	if err != nil {
		return "", err
	}
	return result, nil
}

func selectItems(selectedPos int, allItems []*item, label string) ([]*item, error) {
	// Always prepend a "Done" item to the slice if it doesn't
	// already exist.
	const doneID = "Done"
	if len(allItems) > 0 && allItems[0].ID != doneID {
		var items = []*item{
			{
				ID: doneID,
			},
		}
		allItems = append(items, allItems...)
	}

	// Define promptui template
	templates := &promptui.SelectTemplates{
		Label: `{{if .IsSelected}}
                    ✔
                {{end}} {{ .ID }} - label`,
		Active:   "→ {{if .IsSelected}}✔ {{end}}{{ .ID | cyan }}",
		Inactive: "{{if .IsSelected}}✔ {{end}}{{ .ID | cyan }}",
	}

	prompt := promptui.Select{
		Label:     label,
		Items:     allItems,
		Templates: templates,
		Size:      5,
		// Start the cursor at the currently selected index
		CursorPos:    selectedPos,
		HideSelected: true,
	}

	selectionIdx, _, err := prompt.Run()
	if err != nil {
		return nil, err
	}

	chosenItem := allItems[selectionIdx]

	if chosenItem.ID != doneID {
		// If the user selected something other than "Done",
		// toggle selection on this item and run the function again.
		chosenItem.IsSelected = !chosenItem.IsSelected
		return selectItems(selectionIdx, allItems, label)
	}

	// If the user selected the "Done" item, return
	// all selected items.
	var selectedItems []*item
	for _, i := range allItems {
		if i.IsSelected {
			selectedItems = append(selectedItems, i)
		}
	}
	return selectedItems, nil
}
