//go:build integration

package main

import (
	"fmt"
	"os"

	"github.com/loomi-labs/arco/backend/app/types"
	"github.com/loomi-labs/arco/backend/platform"
	"github.com/spf13/cobra"
)

const versionFlag = "version"
const showBorgUrlFlag = "show-borg-url"

// Execute adds all child commands to the root command and sets flags appropriately.
// This is the minimal version used for integration testing.
func Execute() {
	rootCmd := &cobra.Command{
		Use:   "arco",
		Short: "Arco testing CLI for integration tests",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if version flag is set
			if showVersion, _ := cmd.Flags().GetBool(versionFlag); showVersion {
				fmt.Printf("Arco %s\n", types.Version)
				return nil
			}

			// Check if show-borg-url flag is set
			if showBorgUrl, _ := cmd.Flags().GetBool(showBorgUrlFlag); showBorgUrl {
				binary, err := platform.GetLatestBorgBinary(platform.Binaries)
				if err != nil {
					return fmt.Errorf("failed to get borg binary: %w", err)
				}
				fmt.Println(binary.Url)
				return nil
			}

			return nil
		},
	}

	// Add only the essential flags needed for testing
	rootCmd.PersistentFlags().BoolP(versionFlag, "v", false, "print version information and exit")
	rootCmd.PersistentFlags().Bool(showBorgUrlFlag, false, "print borg download URL for current system and exit")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main() {
	Execute()
}
