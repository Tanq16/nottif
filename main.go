package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tanq16/notif/internal"
)

func main() {
	var webhookURL, execType, command, rawMessage string

	rootCmd := &cobra.Command{
		Use:   "gonotif",
		Short: "A Discord webhook notification tool for command execution and messages",
		Run: func(cmd *cobra.Command, args []string) {
			// Get webhook URL from config if not provided
			if webhookURL == "" {
				var err error
				webhookURL, err = internal.GetWebhookFromConfig()
				if err != nil {
					fmt.Printf("Error: %v\n", err)
					os.Exit(1)
				}
			}
			notifier := internal.NewNotifier(webhookURL)
			// Handle raw message if provided
			if rawMessage != "" {
				if err := notifier.SendRawMessage(rawMessage); err != nil {
					fmt.Printf("Error sending raw message: %v\n", err)
					os.Exit(1)
				}
				return
			}
			// Handle command execution
			if err := notifier.HandleCommand(command, execType); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	rootCmd.Flags().StringVarP(&webhookURL, "webhook", "w", "", "Discord webhook URL")
	rootCmd.Flags().StringVarP(&execType, "type", "t", "cmd", "Execution type (cmd/out)")
	rootCmd.Flags().StringVarP(&command, "command", "c", "", "Command to execute")
	rootCmd.Flags().StringVarP(&rawMessage, "message", "m", "", "Send a raw message to webhook")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
