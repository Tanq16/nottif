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
	"regexp"
	"strings"
	"sync"
	"time"
)

type Notifier struct {
	webhookURL string
}

func NewNotifier(webhookURL string) *Notifier {
	return &Notifier{
		webhookURL: webhookURL,
	}
}

func GetWebhookFromConfig() (string, error) {
	// Check home directory config
	homeDir, err := os.UserHomeDir()
	if err == nil {
		configPath := filepath.Join(homeDir, ".notif.webhook")
		if webhook, err := os.ReadFile(configPath); err == nil {
			return strings.TrimSpace(string(webhook)), nil
		}
	}

	// Check persist directory config
	persistPath := filepath.Join("/persist", ".notif.webhook")
	if webhook, err := os.ReadFile(persistPath); err == nil {
		return strings.TrimSpace(string(webhook)), nil
	}

	return "", fmt.Errorf("webhook URL not found in config files or command line arguments")
}

func (n *Notifier) SendRawMessage(message string) error {
	webhook := DiscordWebhook{
		Embeds: []Embed{
			{
				Title:       "Notif Message",
				Description: message,
				Color:       0x00ff00,
				Timestamp:   time.Now().Format(time.RFC3339),
				Footer: Footer{
					Text: "Manual notification via notif",
				},
			},
		},
	}
	return n.sendToDiscord(webhook)
}

func (n *Notifier) HandleCommand(command, execType string) error {
	if command == "" {
		return fmt.Errorf("command is required when not sending raw message")
	}

	output, err, duration := n.executeCommand(command)
	if err != nil {
		return fmt.Errorf("command execution failed: %v", err)
	}

	webhook := DiscordWebhook{
		Embeds: []Embed{
			{
				Title:     "Command Execution Report",
				Color:     0x00ff00,
				Timestamp: time.Now().Format(time.RFC3339),
				Footer: Footer{
					Text: "Command notification via notif",
				},
			},
		},
	}

	if execType == "cmd" {
		webhook.Embeds[0].Fields = []Field{
			{
				Name:   "Command",
				Value:  fmt.Sprintf("```%s```", command),
				Inline: false,
			},
			{
				Name:   "Duration",
				Value:  formatDuration(duration),
				Inline: true,
			},
			{
				Name:   "Status",
				Value:  "✅ Completed",
				Inline: true,
			},
		}
	} else if execType == "out" {
		webhook.Embeds[0].Fields = []Field{
			{
				Name:   "Command",
				Value:  fmt.Sprintf("```%s```", command),
				Inline: false,
			},
			{
				Name:   "Output",
				Value:  fmt.Sprintf("```%s```", output),
				Inline: false,
			},
			{
				Name:   "Duration",
				Value:  formatDuration(duration),
				Inline: true,
			},
		}
	}

	return n.sendToDiscord(webhook)
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

	// Store the complete output for returning
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

	// Wait for both goroutines to finish
	wg.Wait()

	// Wait for the command to finish and get the error if any
	err = command.Wait()
	duration := time.Since(start)

	// Clean up the output
	cleanOutput := cleanTerminalOutput(outputBuilder.String())

	return cleanOutput, err, duration
}

// New helper function to clean terminal output
func cleanTerminalOutput(output string) string {
	// Remove common terminal escape sequences that might appear due to interactive mode
	// This is a simple cleanup - you might need to add more patterns
	patterns := []string{
		`\x1b\[[0-9;]*[a-zA-Z]`, // ANSI escape sequences
		`\r`,                    // Carriage returns
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		output = re.ReplaceAllString(output, "")
	}

	// Trim any leading/trailing whitespace that might have been added
	return strings.TrimSpace(output)
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
