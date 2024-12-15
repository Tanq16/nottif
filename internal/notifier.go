package internal

import (
	"fmt"
	"time"
)

func (n *Notifier) SendRawMessage(message string) error {
	webhook := DiscordWebhook{
		Embeds: []Embed{
			{
				Title:       "RAW MESSAGE",
				Description: message,
				Color:       0x554422,
				Timestamp:   time.Now().Format(time.RFC3339),
				Footer: Footer{
					Text: "Manual alert via NOTIF",
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
	cmdWebhook := DiscordWebhook{
		Embeds: []Embed{
			{
				Title:     "COMMAND EXEC",
				Color:     0x554422,
				Timestamp: time.Now().Format(time.RFC3339),
				Footer: Footer{
					Text: "Command result via NOTIF",
				},
				Fields: []Field{
					{
						Name:   "COMMAND",
						Value:  fmt.Sprintf("```%s```", command),
						Inline: false,
					},
					{
						Name:   "DURATION",
						Value:  formatDuration(duration),
						Inline: true,
					},
				},
			},
		},
	}
	if execType == "cmd" {
		return n.sendToDiscord(cmdWebhook)
	}

	// For output type, chunk the output if needed
	outputParts := chunkString(output, MaxFieldLength)
	// If output is too large, send a message indicating that
	if len(outputParts) > 5 {
		cmdWebhook.Embeds[0].Fields = append(cmdWebhook.Embeds[0].Fields, Field{
			Name:   "OUTPUT",
			Value:  fmt.Sprintf("Output too large to send (%d parts)", len(outputParts)),
			Inline: false,
		})
		return n.sendToDiscord(cmdWebhook)
	}

	// If output fits in a single message
	if len(outputParts) == 1 {
		webhook := DiscordWebhook{
			Embeds: []Embed{
				{
					Title:     "COMMAND EXEC OUTPUT",
					Color:     0x554422,
					Timestamp: time.Now().Format(time.RFC3339),
					Footer: Footer{
						Text: "Command output via NOTIF",
					},
					Fields: []Field{
						{
							Name:   "COMMAND",
							Value:  fmt.Sprintf("```%s```", command),
							Inline: false,
						},
						{
							Name:   "OUTPUT",
							Value:  fmt.Sprintf("```\n%s\n```", outputParts[0]),
							Inline: false,
						},
						{
							Name:   "DURATION",
							Value:  formatDuration(duration),
							Inline: true,
						},
					},
				},
			},
		}
		return n.sendToDiscord(webhook)
	}

	// For multiple parts, send multiple messages
	for i, part := range outputParts {
		webhook := DiscordWebhook{
			Embeds: []Embed{
				{
					Title:     fmt.Sprintf("COMMAND EXEC OUTPUT (Part %d/%d)", i+1, len(outputParts)),
					Color:     0x554422,
					Timestamp: time.Now().Format(time.RFC3339),
					Footer: Footer{
						Text: "Command output via NOTIF",
					},
					Fields: []Field{},
				},
			},
		}
		// First part includes the command
		if i == 0 {
			webhook.Embeds[0].Fields = append(webhook.Embeds[0].Fields, Field{
				Name:   "COMMAND",
				Value:  fmt.Sprintf("```%s```", command),
				Inline: false,
			})
		}
		webhook.Embeds[0].Fields = append(webhook.Embeds[0].Fields, Field{
			Name:   "OUTPUT",
			Value:  fmt.Sprintf("\n%s\n", part),
			Inline: false,
		})
		// Last part includes the duration
		if i == len(outputParts)-1 {
			webhook.Embeds[0].Fields = append(webhook.Embeds[0].Fields, Field{
				Name:   "DURATION",
				Value:  formatDuration(duration),
				Inline: true,
			})
		}
		if err := n.sendToDiscord(webhook); err != nil {
			return err
		}
	}
	return nil
}
