package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/tanq16/nottif/internal"
)

var (
	webhookURL string
	message    string
)

var NottifVersion = "dev"

var rootCmd = &cobra.Command{
	Use:     "nottif [message]",
	Short:   "A Discord webhook notification tool for sending markdown messages",
	Version: NottifVersion,
	Args:    cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		var webhooks []string
		var err error
		if len(args) > 0 {
			message = args[0]
		}
		if message == "" {
			// Try to read from pipe first
			message, err = internal.ReadInput()
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

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&webhookURL, "webhook", "w", "", "Discord webhook URL")
}
