package internal

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func NewNotifier(webhookURL string) *Notifier {
	return &Notifier{
		webhookURL: webhookURL,
	}
}

func GetWebhookFromConfig() (string, error) {
	// Check config files for webhook URL
	homeDir, err := os.UserHomeDir()
	if err == nil {
		configPath := filepath.Join(homeDir, ".notif.webhook")
		if webhook, err := os.ReadFile(configPath); err == nil {
			return strings.TrimSpace(string(webhook)), nil
		}
	}
	persistPath := filepath.Join("/persist", ".notif.webhook")
	if webhook, err := os.ReadFile(persistPath); err == nil {
		return strings.TrimSpace(string(webhook)), nil
	}
	return "", fmt.Errorf("webhook URL not found in config files or command line arguments")
}

func (n *Notifier) executeCommand(cmd string) (string, error, time.Duration) {
	start := time.Now()
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	var command *exec.Cmd
	if strings.Contains(shell, "zsh") {
		command = exec.Command(shell, "-i", "-c", cmd)
	} else if strings.Contains(shell, "bash") {
		command = exec.Command(shell, "-ic", cmd)
	} else {
		command = exec.Command(shell, "-c", cmd)
	}
	command.Env = os.Environ()

	// Set up pipes for stdout and stderr
	stdout, err := command.StdoutPipe()
	if err != nil {
		return "", err, 0
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		return "", err, 0
	}
	var outputBuilder strings.Builder

	// Start the command
	if err := command.Start(); err != nil {
		return "", err, 0
	}

	// Create a WaitGroup to wait for both goroutines
	var wg sync.WaitGroup
	wg.Add(2)

	// Handle stdout
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Println(line)
			outputBuilder.WriteString(line + "\n")
		}
	}()

	// Handle stderr
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Fprintln(os.Stderr, line)
			outputBuilder.WriteString(line + "\n")
		}
	}()

	// Wait for the command, stderr, and stdout to finish and get the error if any
	wg.Wait()
	err = command.Wait()
	duration := time.Since(start)
	return outputBuilder.String(), err, duration
}

func (n *Notifier) sendToDiscord(webhook DiscordWebhook) error {
	payload, err := json.Marshal(webhook)
	if err != nil {
		return err
	}
	resp, err := http.Post(n.webhookURL, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 204 {
		return fmt.Errorf("discord webhook failed with status: %d", resp.StatusCode)
	}
	return nil
}

func formatDuration(d time.Duration) string {
	if d.Hours() >= 1 {
		return fmt.Sprintf("%.2f hours", d.Hours())
	} else if d.Minutes() >= 1 {
		return fmt.Sprintf("%.2f minutes", d.Minutes())
	}
	return fmt.Sprintf("%.2f seconds", d.Seconds())
}

func chunkString(s string, chunkSize int) []string {
	var chunks []string
	runes := []rune(s)
	for i := 0; i < len(runes); i += chunkSize {
		end := i + chunkSize
		if end > len(runes) {
			end = len(runes)
		}
		chunks = append(chunks, string(runes[i:end]))
	}
	return chunks
}

// New helper function to clean terminal output
// func cleanTerminalOutput(output string) string {
// 	patterns := []string{
// 		`\x1b\[[0-9;]*[a-zA-Z]`, // ANSI escape sequences
// 		`\r`,                    // Carriage returns
// 	}
// 	for _, pattern := range patterns {
// 		re := regexp.MustCompile(pattern)
// 		output = re.ReplaceAllString(output, "")
// 	}
// 	return strings.TrimSpace(output)
// }
