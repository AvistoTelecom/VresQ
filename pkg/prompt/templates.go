package prompt

import "github.com/manifoldco/promptui"

func getSelectTemplates() *promptui.SelectTemplates {
	return &promptui.SelectTemplates{
		Label:    "{{ . }}?",
		Active:   "→ ✔ {{ .Name | blue }}",
		Inactive: "{{ .Name | cyan }}",
		Selected: " {{ .Name | green }}",
		Details: `
		--------- CONTEXT ----------
		{{ "Name:" | faint }}	{{ .Name }}
		{{ "Cluster:" | faint }}	{{ .Context.Cluster }}
		{{ "User:" | faint }}	{{ .Context.User }}`,
	}
}

func getPromptTemplates() *promptui.PromptTemplates {
	return &promptui.PromptTemplates{
		Prompt:          prompt,
		Valid:           greenCheckmark + toCyan,
		Invalid:         redCross + toRed,
		Success:         toRed,
		ValidationError: "{{ . | red }}",
	}
}
