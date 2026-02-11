// Package outputs provides GitHub Actions output and logging utilities.
package outputs

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// SetOutput writes a value to GITHUB_OUTPUT.
func SetOutput(name, value string) {
	outputFile := os.Getenv("GITHUB_OUTPUT")
	if outputFile != "" {
		f, err := os.OpenFile(outputFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			fmt.Printf("::set-output name=%s::%s\n", name, value)
			return
		}
		defer f.Close()

		if strings.Contains(value, "\n") {
			delimiter := fmt.Sprintf("ghadelimiter_%d", time.Now().UnixNano())
			fmt.Fprintf(f, "%s<<%s\n%s\n%s\n", name, delimiter, value, delimiter)
		} else {
			fmt.Fprintf(f, "%s=%s\n", name, value)
		}
	} else {
		fmt.Printf("::set-output name=%s::%s\n", name, value)
	}
}

// LogInfo prints an info message.
func LogInfo(msg string) {
	fmt.Println(msg)
}

// LogWarning prints a warning message.
func LogWarning(msg string) {
	fmt.Printf("::warning::%s\n", msg)
}

// LogError prints an error message.
func LogError(msg string) {
	fmt.Printf("::error::%s\n", msg)
}

// LogGroup starts a collapsible group.
func LogGroup(title string) {
	fmt.Printf("::group::%s\n", title)
}

// LogEndGroup ends a collapsible group.
func LogEndGroup() {
	fmt.Println("::endgroup::")
}
