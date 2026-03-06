package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "dr-evaluation",
	Short: "Velero backup/restore evaluation tool",
	Long:  "Evaluates Velero backup and restore operations with analysis and comparison reports in Markdown format.",
}

func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return err
	}
	return nil
}
