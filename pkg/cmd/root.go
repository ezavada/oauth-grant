package cmd

import (
	"fmt"
	"os"

	"github.com/rishabhsvats/oauth-grant/pkg/flow"
	"github.com/spf13/cobra"
)

type Config struct {
	Issuer   string
	ClientId string
	Flow     string
}

var config Config

var rootCmd = &cobra.Command{
	Use:   "grant",
	Short: "grant is a command line tool",
	Long:  `grant is a command line tool for testing Oauth grant flow.`,
	Run: func(cmd *cobra.Command, args []string) {
		token, err := flow.OauthFlow(config.Issuer, config.ClientId, config.Flow)
		if err != nil {
			fmt.Printf("Oauth flow execution error: %s\n", err)
			return
		}
		fmt.Printf("Received response: %s\n", token)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {

	rootCmd.PersistentFlags().StringVarP(&config.Issuer, "issuer", "i", "", "oidc issuer url")
	rootCmd.PersistentFlags().StringVarP(&config.ClientId, "client_id", "c", "", "client ID required to test the flow")
	rootCmd.PersistentFlags().StringVarP(&config.Flow, "grant_type", "f", "", "oauth flow to test")
}
