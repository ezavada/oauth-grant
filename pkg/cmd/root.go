package cmd

import (
	"fmt"
	"os"

	"github.com/rishabhsvats/oauth-grant/pkg/flow"
	"github.com/spf13/cobra"
)

type Config struct {
	ClientId string
	Verbose  bool
}

var config Config

var rootCmd = &cobra.Command{
	Use:   "grant",
	Short: "grant is a command line tool",
	Long:  `grant is a command line tool for testing Oauth grant flow.`,
	Run: func(cmd *cobra.Command, args []string) {
		token, err := flow.OauthFlow(config.ClientId, config.Verbose)
		if err != nil {
			fmt.Printf("Oauth flow execution error: %s\n", err)
			return
		}
		if config.Verbose {
			fmt.Printf("Received response: %s\n", token)
		}
		fmt.Printf("Authorization successful!\n")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&config.ClientId, "client_id", "c", "", "client ID required to test the flow")
	rootCmd.PersistentFlags().BoolVarP(&config.Verbose, "verbose", "v", false, "enable verbose output")
}
