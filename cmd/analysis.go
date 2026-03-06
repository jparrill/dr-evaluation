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

var analysisCmd = &cobra.Command{
	Use:   "analysis",
	Short: "Generate a backup/restore analysis report",
	Long: `Fetches Velero backups and restores from the cluster and generates a Markdown report.

Date flags are optional:
  - If neither --start nor --end is provided, the last N backups (per --sample) are reported.
  - If only --start is provided, the end date defaults to now().
  - If only --end is provided, the start date defaults to the earliest available data.`,
	Example: `  # Last 10 backups/restores (no dates needed)
  dr-evaluation analysis --kubeconfig /path/to/kubeconfig --sample 10

  # From a start date until now
  dr-evaluation analysis --start "2026-03-05T00:00:00Z" --kubeconfig /path/to/kubeconfig

  # Explicit date range
  dr-evaluation analysis --start "2026-03-05T00:00:00Z" --end "2026-03-06T23:59:59Z" --kubeconfig /path/to/kubeconfig`,
	RunE: runAnalysis,
}

var (
	analysisStart      string
	analysisEnd        string
	analysisKubeconfig string
	analysisNamespace  string
	analysisSample     int
	analysisOutput     string
)

func init() {
	analysisCmd.Flags().StringVar(&analysisStart, "start", "", "Start date in ISO8601 format (optional)")
	analysisCmd.Flags().StringVar(&analysisEnd, "end", "", "End date in ISO8601 format (default: now)")
	analysisCmd.Flags().StringVar(&analysisKubeconfig, "kubeconfig", "", "Path to kubeconfig file (required)")
	analysisCmd.Flags().StringVar(&analysisNamespace, "namespace", "openshift-adp", "Velero namespace")
	analysisCmd.Flags().IntVar(&analysisSample, "sample", 5, "Number of backups/restores to sample per category")
	analysisCmd.Flags().StringVar(&analysisOutput, "output", "", "Output .md file path (default: reports/analysis_<timestamp>.md)")

	_ = analysisCmd.MarkFlagRequired("kubeconfig")

	rootCmd.AddCommand(analysisCmd)
}

func runAnalysis(cmd *cobra.Command, args []string) error {
	hasStart := analysisStart != ""
	hasEnd := analysisEnd != ""

	var startDate, endDate time.Time
	var err error

	if hasEnd {
		endDate, err = time.Parse(time.RFC3339, analysisEnd)
		if err != nil {
			return fmt.Errorf("parsing --end: %w", err)
		}
	} else {
		endDate = time.Now().UTC()
	}

	if hasStart {
		startDate, err = time.Parse(time.RFC3339, analysisStart)
		if err != nil {
			return fmt.Errorf("parsing --start: %w", err)
		}
	}
	// If !hasStart, startDate stays zero-value — report pkg will skip date filtering

	if _, err := os.Stat(analysisKubeconfig); os.IsNotExist(err) {
		return fmt.Errorf("kubeconfig file not found: %s", analysisKubeconfig)
	}

	output := analysisOutput
	if output == "" {
		if err := ensureReportsDir(); err != nil {
			return err
		}
		output = defaultReportPath("analysis")
	}

	fmt.Fprintf(os.Stderr, "Connecting to cluster using kubeconfig: %s\n", analysisKubeconfig)
	client, err := velero.NewClient(analysisKubeconfig, analysisNamespace)
	if err != nil {
		return fmt.Errorf("creating velero client: %w", err)
	}

	ctx := context.Background()
	fmt.Fprintf(os.Stderr, "Fetching backups and restores from namespace %s...\n", analysisNamespace)

	backups, err := client.FetchBackups(ctx)
	if err != nil {
		return fmt.Errorf("fetching backups: %w", err)
	}

	restores, err := client.FetchRestores(ctx)
	if err != nil {
		return fmt.Errorf("fetching restores: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Found %d backups and %d restores total\n", len(backups), len(restores))

	content := report.GenerateAnalysis(report.AnalysisInput{
		StartDate:    startDate,
		EndDate:      endDate,
		HasStartDate: hasStart,
		Namespace:    analysisNamespace,
		Sample:       analysisSample,
		Backups:      backups,
		Restores:     restores,
	})

	if err := writeReport(output, content); err != nil {
		return fmt.Errorf("writing report: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Report written to: %s\n", output)
	return nil
}
