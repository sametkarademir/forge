package ui

import (
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/olekukonko/tablewriter"
	"golang.org/x/term"
)

// IsInteractive reports whether stdin is an interactive terminal.
func IsInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// Confirm shows an interactive yes/no prompt defaulting to No.
func Confirm(question string) (bool, error) {
	return ConfirmDefault(question, false)
}

// ConfirmDefault shows an interactive yes/no prompt with an explicit default.
func ConfirmDefault(question string, defaultYes bool) (bool, error) {
	var answer bool
	prompt := &survey.Confirm{Message: question, Default: defaultYes}
	err := survey.AskOne(prompt, &answer)
	return answer, err
}

// Text shows a single-line input prompt. An empty submission returns defaultVal.
// validate is optional; survey re-prompts automatically until it returns nil.
func Text(label, defaultVal string, validate func(string) error) (string, error) {
	var answer string
	opts := []survey.AskOpt{}
	if validate != nil {
		opts = append(opts, survey.WithValidator(func(ans interface{}) error {
			s, _ := ans.(string)
			if s == "" {
				s = defaultVal
			}
			return validate(s)
		}))
	}
	err := survey.AskOne(&survey.Input{Message: label, Default: defaultVal}, &answer, opts...)
	if err != nil {
		return "", err
	}
	if answer == "" {
		answer = defaultVal
	}
	return answer, nil
}

// Password shows a masked input prompt. validate is optional; survey re-prompts
// automatically until it returns nil.
func Password(label string, validate func(string) error) (string, error) {
	var answer string
	opts := []survey.AskOpt{}
	if validate != nil {
		opts = append(opts, survey.WithValidator(func(ans interface{}) error {
			s, _ := ans.(string)
			return validate(s)
		}))
	}
	err := survey.AskOne(&survey.Password{Message: label}, &answer, opts...)
	return answer, err
}

// Select shows a single-choice picker. defaultVal should be one of options;
// if not found, the first option is used as the default.
func Select(label string, options []string, defaultVal string) (string, error) {
	var answer string
	err := survey.AskOne(&survey.Select{
		Message: label,
		Options: options,
		Default: defaultVal,
	}, &answer)
	return answer, err
}

// RenderTable prints a bordered-less table to stdout.
func RenderTable(headers []string, rows [][]string) {
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader(headers)
	table.SetBorder(false)
	table.SetHeaderLine(false)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("  ")
	table.SetRowSeparator("")
	table.SetTablePadding("  ")
	table.SetNoWhiteSpace(true)
	for _, row := range rows {
		table.Append(row)
	}
	table.Render()
}
