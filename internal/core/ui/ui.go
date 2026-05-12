package ui

import (
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/olekukonko/tablewriter"
)

// Confirm shows an interactive yes/no prompt. Returns false on non-TTY or interrupt.
func Confirm(question string) (bool, error) {
	var answer bool
	prompt := &survey.Confirm{Message: question}
	err := survey.AskOne(prompt, &answer)
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
