package internal

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

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

func (n *Notifier) sendToDiscord(webhookURL string, webhook DiscordWebhook) error {
	payload, err := json.Marshal(webhook)
	if err != nil {
		return err
	}
	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 204 {
		return fmt.Errorf("discord webhook failed with status: %d", resp.StatusCode)
	}
	return nil
}

func ReadInput() (string, error) {
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		reader := bufio.NewReader(os.Stdin)
		var builder strings.Builder
		for {
			line, err := reader.ReadString('\n')
			if err != nil && err != io.EOF {
				return "", err
			}
			builder.WriteString(line)
			if err == io.EOF {
				break
			}
		}
		return strings.TrimSpace(builder.String()), nil
	}
	return "", nil
}
