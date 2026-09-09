package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/brad/gh-pr-pro/pkg/cache"
	"github.com/spf13/cobra"
)

var (
	flagCacheAll      bool
	flagCacheListJSON bool
)

// cacheCmd represents the parent command for inspecting and managing the local disk cache.
var cacheCmd = &cobra.Command{
	Use:   "cache [command]",
	Short: "Inspect, list, clean, and manage local PR disk cache",
	Long: `Inspect and manage the local persistent PR cache stored under ~/.cache/gh-pr-pro/.

The cache stores immutable historical PR records and incremental sync state to accelerate
subsequent queries and prevent GitHub GraphQL API rate limit exhaustion.`,
	Example: `  # List all cached repositories and disk space usage
  gh pr-pro cache list

  # List cached entries in JSON format
  gh pr-pro cache list --json

  # Clear cached data for a specific repository
  gh pr-pro cache clean cli/cli

  # Clear all cached repositories
  gh pr-pro cache clean --all

  # Print the local cache directory path
  gh pr-pro cache path`,
}

var cacheListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all cached repositories, record counts, and disk usage",
	Example: `  # View cached repos in tabular format
  gh pr-pro cache list

  # View cached repos in JSON format
  gh pr-pro cache list --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := cache.NewCacheManager()
		if err != nil {
			return fmt.Errorf("failed to access cache manager: %w", err)
		}

		entries, err := mgr.ListEntries()
		if err != nil {
			return fmt.Errorf("failed to list cache entries: %w", err)
		}

		if flagCacheListJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(map[string]interface{}{
				"cache_directory": mgr.GetBaseDir(),
				"total_entries":   len(entries),
				"entries":         entries,
			})
		}

		if len(entries) == 0 {
			fmt.Printf("Cache is empty (%s)\n", mgr.GetBaseDir())
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "REPOSITORY\tPRS\tDISK SIZE\tLAST SYNCED\tFILE PATH")
		fmt.Fprintln(w, "──────────\t───\t─────────\t───────────\t─────────")

		var totalSize int64
		var totalPRs int
		for _, e := range entries {
			totalSize += e.SizeBytes
			totalPRs += e.PRCount
			sizeStr := formatByteSize(e.SizeBytes)
			timeStr := e.LastFetched.Format("2006-01-02 15:04:05")
			if e.LastFetched.IsZero() {
				timeStr = "unknown"
			}
			fmt.Fprintf(w, "%s\t%d\t%s\t%s\t%s\n", e.Repo, e.PRCount, sizeStr, timeStr, e.FilePath)
		}
		_ = w.Flush()

		fmt.Println()
		fmt.Printf("Summary: %d repository cache(s) | %d total PRs | %s total disk space\n",
			len(entries), totalPRs, formatByteSize(totalSize))
		fmt.Printf("Directory: %s\n", mgr.GetBaseDir())

		return nil
	},
}

var cacheCleanCmd = &cobra.Command{
	Use:     "clean [owner/repo]",
	Aliases: []string{"clear", "delete", "rm"},
	Short:   "Delete cached PR records for a specific repository or all repositories",
	Example: `  # Delete cache for a specific repository
  gh pr-pro cache clean cli/cli

  # Delete all cached repositories
  gh pr-pro cache clean --all`,
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := cache.NewCacheManager()
		if err != nil {
			return fmt.Errorf("failed to access cache manager: %w", err)
		}

		if flagCacheAll {
			deleted, err := mgr.ClearAll()
			if err != nil {
				return fmt.Errorf("failed to clear cache: %w", err)
			}
			fmt.Printf("✅ Successfully cleared all cache entries (%d file(s) removed from %s)\n", deleted, mgr.GetBaseDir())
			return nil
		}

		targetRepo := ""
		if len(args) > 0 {
			targetRepo = args[0]
		} else if flagRepo != "" {
			targetRepo = flagRepo
		}

		if targetRepo == "" {
			return fmt.Errorf("specify a repository (e.g. 'gh pr-pro cache clean owner/repo') or use '--all' to clear everything")
		}

		deleted, err := mgr.Delete(targetRepo)
		if err != nil {
			return fmt.Errorf("failed to delete cache for %s: %w", targetRepo, err)
		}

		if !deleted {
			fmt.Printf("ℹ️  No cached data found for %s\n", targetRepo)
		} else {
			fmt.Printf("✅ Successfully deleted cache for %s\n", targetRepo)
		}

		return nil
	},
}

var cachePathCmd = &cobra.Command{
	Use:     "path",
	Aliases: []string{"dir"},
	Short:   "Print the local cache filesystem directory path",
	Example: `  gh pr-pro cache path`,
	Run: func(cmd *cobra.Command, args []string) {
		mgr, err := cache.NewCacheManager()
		if err != nil {
			fmt.Println("~/.cache/gh-pr-pro")
			return
		}
		fmt.Println(mgr.GetBaseDir())
	},
}

func formatByteSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func init() {
	cacheListCmd.Flags().BoolVar(&flagCacheListJSON, "json", false, "Output cached entries in JSON format")
	cacheCleanCmd.Flags().BoolVar(&flagCacheAll, "all", false, "Clear all cached repository records")

	cacheCmd.AddCommand(cacheListCmd)
	cacheCmd.AddCommand(cacheCleanCmd)
	cacheCmd.AddCommand(cachePathCmd)
}
