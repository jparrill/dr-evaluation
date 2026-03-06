package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"dr-evaluation/pkg/report"
	"dr-evaluation/pkg/velero"
)

var comparisonCmd = &cobra.Command{
	Use:   "comparison",
	Short: "Compare backup/restore performance before and after a cutoff date",
	Long: `Splits all Velero backups and restores into pre/post periods based on a cutoff date,
and generates a Markdown comparison report with performance deltas, trends, and takeaways.`,
	Example: `  dr-evaluation comparison --date "2026-03-05T00:00:00Z" --kubeconfig /path/to/kubeconfig
  dr-evaluation comparison --date "2026-03-05T00:00:00Z" --kubeconfig /path/to/kubeconfig --sample 10
  dr-evaluation comparison --date "2026-03-05T00:00:00Z" --kubeconfig /path/to/kubeconfig --output comparison.md`,
	RunE: runComparison,
}

var (
	comparisonDate       string
	comparisonKubeconfig string
	comparisonNamespace  string
	comparisonSample     int
	comparisonOutput     string
)

func init() {
	comparisonCmd.Flags().StringVar(&comparisonDate, "date", "", "Cutoff date in ISO8601 format - separates pre/post periods (required)")
	comparisonCmd.Flags().StringVar(&comparisonKubeconfig, "kubeconfig", "", "Path to kubeconfig file (required)")
	comparisonCmd.Flags().StringVar(&comparisonNamespace, "namespace", "openshift-adp", "Velero namespace")
	comparisonCmd.Flags().IntVar(&comparisonSample, "sample", 5, "Number of samples for duration trend")
	comparisonCmd.Flags().StringVar(&comparisonOutput, "output", "", "Output .md file path (default: reports/comparison_<timestamp>.md)")

	_ = comparisonCmd.MarkFlagRequired("date")
	_ = comparisonCmd.MarkFlagRequired("kubeconfig")

	rootCmd.AddCommand(comparisonCmd)
}

func runComparison(cmd *cobra.Command, args []string) error {
	cutoffDate, err := time.Parse(time.RFC3339, comparisonDate)
	if err != nil {
		return fmt.Errorf("parsing --date: %w", err)
	}

	if _, err := os.Stat(comparisonKubeconfig); os.IsNotExist(err) {
		return fmt.Errorf("kubeconfig file not found: %s", comparisonKubeconfig)
	}

	output := comparisonOutput
	if output == "" {
		if err := ensureReportsDir(); err != nil {
			return err
		}
		output = defaultReportPath("comparison")
	}

	fmt.Fprintf(os.Stderr, "Connecting to cluster using kubeconfig: %s\n", comparisonKubeconfig)
	client, err := velero.NewClient(comparisonKubeconfig, comparisonNamespace)
	if err != nil {
		return fmt.Errorf("creating velero client: %w", err)
	}

	ctx := context.Background()
	fmt.Fprintf(os.Stderr, "Fetching backups and restores from namespace %s...\n", comparisonNamespace)

	backups, err := client.FetchBackups(ctx)
	if err != nil {
		return fmt.Errorf("fetching backups: %w", err)
	}

	restores, err := client.FetchRestores(ctx)
	if err != nil {
		return fmt.Errorf("fetching restores: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Found %d backups and %d restores total\n", len(backups), len(restores))

	content := report.GenerateComparison(report.ComparisonInput{
		CutoffDate: cutoffDate,
		Namespace:  comparisonNamespace,
		Sample:     comparisonSample,
		Backups:    backups,
		Restores:   restores,
	})

	if err := writeReport(output, content); err != nil {
		return fmt.Errorf("writing report: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Report written to: %s\n", output)
	return nil
}
