package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/tanq16/notif/internal"
)

func readInput() (string, error) {
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

func main() {
	var webhookURL string
	var message string

	rootCmd := &cobra.Command{
		Use:   "notif [message]",
		Short: "A Discord webhook notification tool for sending markdown messages",
		Run: func(cmd *cobra.Command, args []string) {
			var webhooks []string
			var err error
			if message == "" {
				// Try to read from pipe first
				message, err = readInput()
				if err != nil {
					fmt.Printf("Error reading input: %v\n", err)
					os.Exit(1)
				}
				if message == "" {
					fmt.Println("Error: No message provided. Either pipe input or provide a message argument")
					os.Exit(1)
				}
			}
			// Get webhook URLs
			if webhookURL != "" {
				webhooks = []string{webhookURL}
			} else {
				webhooks, err = internal.GetWebhooksFromConfig()
				if err != nil {
					fmt.Printf("Error: %v\n", err)
					os.Exit(1)
				}
			}
			notifier := internal.NewNotifier(webhooks)
			if err := notifier.SendMessage(message); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	rootCmd.Flags().StringVarP(&webhookURL, "webhook", "w", "", "Discord webhook URL")
	rootCmd.Flags().StringVarP(&message, "message", "m", "", "Message to send")
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
