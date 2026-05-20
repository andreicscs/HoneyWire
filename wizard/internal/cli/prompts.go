package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

func ReadPasswordMasked(prompt string) (string, error) {
	fmt.Print(prompt)

	if term.IsTerminal(int(os.Stdin.Fd())) {
		pw, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err != nil {
			return "", fmt.Errorf("failed to read password: %w", err)
		}
		return string(pw), nil
	}

	reader := bufio.NewReader(os.Stdin)
	pw, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read password from stdin: %w", err)
	}
	return strings.TrimSpace(pw), nil
}

func ResolveDashboardPassword() (string, error) {
	if envPW := os.Getenv("HW_DASHBOARD_PASSWORD"); envPW != "" {
		return envPW, nil
	}

	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return "", fmt.Errorf("dashboard password required but stdin is not a terminal. Set HW_DASHBOARD_PASSWORD environment variable")
	}

	return ReadPasswordMasked("Enter dashboard password: ")
}

func ConfirmAction(prompt string) bool {
	fmt.Printf("    %s? [y/N]: ", prompt)
	input, err := PromptInput("")
	if err != nil {
		return false
	}
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes"
}

func PromptInput(prompt string) (string, error) {
	if prompt != "" {
		fmt.Print(prompt)
	}

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(input), nil
}

func IsTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}
