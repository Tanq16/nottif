package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

const (
	DefaultAvatarURL = "https://raw.githubusercontent.com/tanq16/nottif/main/.github/assets/logo.png"
)

// Notifier handles sending messages to Discord.
type Notifier struct {
	mu         sync.RWMutex
	webhookURL string
}

func New(webhookURL string) *Notifier {
	return &Notifier{
		webhookURL: webhookURL,
	}
}

func (n *Notifier) SetWebhookURL(url string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.webhookURL = url
}

// SendMessage sends a message to the configured Discord webhook
func (n *Notifier) SendMessage(message, username, avatarURL string) error {
	n.mu.RLock()
	url := n.webhookURL
	n.mu.RUnlock()
	if url == "" {
		return fmt.Errorf("webhook URL is not configured")
	}
	if username == "" {
		username = "Nottif Notification"
	}
	if avatarURL == "" {
		avatarURL = DefaultAvatarURL
	}
	webhook := DiscordWebhook{
		Username:  username,
		AvatarURL: avatarURL,
		Embeds: []Embed{
			{
				Description: message,
				Color:       0x89b4fa, // Catppuccin Blue
				Timestamp:   time.Now().Format(time.RFC3339),
				Footer:      Footer{Text: "via Nottif"},
			},
		},
	}

	payload, err := json.Marshal(webhook)
	if err != nil {
		return err
	}
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("discord webhook failed with status: %s", resp.Status)
	}
	return nil
}

type DiscordWebhook struct {
	Content   string  `json:"content,omitempty"`
	Username  string  `json:"username,omitempty"`
	AvatarURL string  `json:"avatar_url,omitempty"`
	Embeds    []Embed `json:"embeds,omitempty"`
}

type Embed struct {
	Description string `json:"description"`
	Color       int    `json:"color"`
	Footer      Footer `json:"footer"`
	Timestamp   string `json:"timestamp"`
}

type Footer struct {
	Text string `json:"text"`
}
