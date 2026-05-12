package logger

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

var (
	greenBold = color.New(color.FgGreen, color.Bold).SprintFunc()
	redBold   = color.New(color.FgRed, color.Bold).SprintFunc()
	yellow    = color.New(color.FgYellow).SprintFunc()
)

func Success(msg string) { fmt.Println(greenBold("✓") + " " + msg) }
func Error(msg string)   { fmt.Fprintln(os.Stderr, redBold("✗")+" "+msg) }
func Warn(msg string)    { fmt.Println(yellow("⚠") + " " + msg) }
func Info(msg string)    { fmt.Println(msg) }
