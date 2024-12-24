package internal

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func NewNotifier(webhookURLs []string) *Notifier {
	return &Notifier{
		webhookURLs: webhookURLs,
	}
}

func GetWebhooksFromConfig() ([]string, error) {
	var webhooks []string

	// Check home directory config
	homeDir, err := os.UserHomeDir()
	if err == nil {
		configPath := filepath.Join(homeDir, ".notif.webhook")
		if content, err := os.ReadFile(configPath); err == nil {
			scanner := bufio.NewScanner(bytes.NewReader(content))
			for scanner.Scan() {
				if url := strings.TrimSpace(scanner.Text()); url != "" {
					webhooks = append(webhooks, url)
				}
			}
		}
	}

	// Check persist directory config
	persistPath := filepath.Join("/persist", ".notif.webhook")
	if content, err := os.ReadFile(persistPath); err == nil {
		scanner := bufio.NewScanner(bytes.NewReader(content))
		for scanner.Scan() {
			if url := strings.TrimSpace(scanner.Text()); url != "" {
				webhooks = append(webhooks, url)
			}
		}
	}
	if len(webhooks) == 0 {
		return nil, fmt.Errorf("no webhook URLs found in config files")
	}
	return webhooks, nil
}

func (n *Notifier) SendMessage(message string) error {
	// Split message if needed
	parts := chunkString(message, MaxFieldLength)
	if len(parts) > 5 {
		return fmt.Errorf("message too large (would require %d parts)", len(parts))
	}
	// Send to all configured webhooks
	for _, webhookURL := range n.webhookURLs {
		for i, part := range parts {
			webhook := DiscordWebhook{
				Username:  "Notif",
				AvatarURL: AvatarURL,
				Embeds: []Embed{
					{
						Description: part,
						Color:       0x554422,
						Timestamp:   time.Now().Format(time.RFC3339),
						Footer: Footer{
							Text: func() string {
								if len(parts) > 1 {
									return fmt.Sprintf("Message via NOTIF (Part %d/%d)", i+1, len(parts))
								}
								return "Message via NOTIF"
							}(),
						},
					},
				},
			}
			if err := n.sendToDiscord(webhookURL, webhook); err != nil {
				return fmt.Errorf("failed to send to webhook: %v", err)
			}
		}
	}
	return nil
}
