package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/rishabhsvats/oauth-grant/pkg/flow"
	"github.com/spf13/cobra"
)

var cloneCmd = &cobra.Command{
	Use:   "clone [repository-url]",
	Short: "Clone a GitHub repository without history",
	Long:  `Clone a GitHub repository without history into a temporary directory using the GitHub access token for authentication.`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repoURL := args[0]

		// Get the access token from the OAuth flow
		token, err := flow.GetAccessToken(config.ClientId)
		if err != nil {
			fmt.Printf("Failed to get access token: %s\n", err)
			return
		}

		// Create a temporary directory
		tmpDir, err := os.MkdirTemp("", "github-clone-*")
		if err != nil {
			fmt.Printf("Failed to create temporary directory: %s\n", err)
			return
		}
		defer os.RemoveAll(tmpDir)

		// Modify the repository URL to include the token
		// Convert https://github.com/owner/repo.git to https://<token>@github.com/owner/repo.git
		repoURL = strings.Replace(repoURL, "https://", fmt.Sprintf("https://%s@", token), 1)

		// Clone the repository without history
		cloneCmd := exec.Command("git", "clone", "--verbose", "--depth", "1", repoURL, tmpDir)
		output, err := cloneCmd.CombinedOutput()
		if err != nil {
			fmt.Printf("Failed to clone repository: %s\n", err)
			if config.Verbose {
				fmt.Printf("Git command output: %s\n", string(output))
				fmt.Printf("Command was: git clone --depth 1 %s %s\n", repoURL, tmpDir)
			}
			return
		}

		fmt.Printf("Repository cloned successfully to: %s\n", tmpDir)
	},
}

func init() {
	rootCmd.AddCommand(cloneCmd)
}
