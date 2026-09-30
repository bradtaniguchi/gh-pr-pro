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
	flagCacheAll           bool
	flagCacheListJSON      bool
	flagCacheStatusJSON    bool
	flagCacheOutdated      bool
	flagCacheCorrupt       bool
	flagCacheMigrateAll    bool
	flagCacheMigrateDryRun bool
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

  # Check cache health, semantic versions, and recommended actions
  gh pr-pro cache status

  # Migrate all outdated cache files to the current package version
  gh pr-pro cache migrate --all

  # Preview migration for a specific repository
  gh pr-pro cache migrate cli/cli --dry-run

  # Clear only outdated cache entries
  gh pr-pro cache clean --outdated

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
				"current_version": cache.CurrentVersion,
				"total_entries":   len(entries),
				"entries":         entries,
			})
		}

		if len(entries) == 0 {
			fmt.Printf("Cache is empty (%s)\n", mgr.GetBaseDir())
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "REPOSITORY\tVERSION\tSTATUS\tPRS\tDISK SIZE\tLAST SYNCED\tFILE PATH")
		fmt.Fprintln(w, "──────────\t───────\t──────\t───\t─────────\t───────────\t─────────")

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
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\t%s\n", e.Repo, e.Version, e.Status, e.PRCount, sizeStr, timeStr, e.FilePath)
		}
		_ = w.Flush()

		fmt.Println()
		fmt.Printf("Summary: %d repository cache(s) | %d total PRs | %s total disk space\n",
			len(entries), totalPRs, formatByteSize(totalSize))
		fmt.Printf("Directory: %s\n", mgr.GetBaseDir())

		return nil
	},
}

var cacheStatusCmd = &cobra.Command{
	Use:     "status",
	Aliases: []string{"check", "doctor"},
	Short:   "Check cache health, schema versions, and migration status",
	Example: `  # Check cache health and migration status
  gh pr-pro cache status

  # Output cache status in JSON format
  gh pr-pro cache status --json`,
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := cache.NewCacheManager()
		if err != nil {
			return fmt.Errorf("failed to access cache manager: %w", err)
		}

		entries, err := mgr.ListEntries()
		if err != nil {
			return fmt.Errorf("failed to check cache status: %w", err)
		}

		type StatusItem struct {
			Repo      string            `json:"repo"`
			Version   string            `json:"version"`
			Status    cache.CacheStatus `json:"status"`
			PRCount   int               `json:"pr_count"`
			SizeBytes int64             `json:"size_bytes"`
			Action    string            `json:"action"`
		}

		var items []StatusItem
		var currentCount, outdatedCount, futureCount, corruptCount int

		for _, e := range entries {
			action := "Up to date"
			switch e.Status {
			case cache.StatusCurrent:
				currentCount++
			case cache.StatusOutdated:
				outdatedCount++
				if e.IsMigratable {
					action = fmt.Sprintf("Run 'gh pr-pro cache migrate %s'", e.Repo)
				} else {
					action = fmt.Sprintf("Run 'gh pr-pro cache clean %s' to re-sync", e.Repo)
				}
			case cache.StatusFuture:
				futureCount++
				action = "Upgrade gh-pr-pro extension"
			case cache.StatusCorrupt:
				corruptCount++
				action = "Run 'gh pr-pro cache clean --corrupt'"
			}

			items = append(items, StatusItem{
				Repo:      e.Repo,
				Version:   e.Version,
				Status:    e.Status,
				PRCount:   e.PRCount,
				SizeBytes: e.SizeBytes,
				Action:    action,
			})
		}

		if flagCacheStatusJSON {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(map[string]interface{}{
				"cache_directory":  mgr.GetBaseDir(),
				"target_version":   cache.CurrentVersion,
				"total_entries":    len(entries),
				"current_entries":  currentCount,
				"outdated_entries": outdatedCount,
				"future_entries":   futureCount,
				"corrupt_entries":  corruptCount,
				"items":            items,
			})
		}

		if len(entries) == 0 {
			fmt.Printf("Cache is empty (%s). All future fetches will create version %s caches.\n", mgr.GetBaseDir(), cache.CurrentVersion)
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
		fmt.Fprintln(w, "REPOSITORY\tVERSION\tSTATUS\tPRS\tDISK SIZE\tACTION")
		fmt.Fprintln(w, "──────────\t───────\t──────\t───\t─────────\t──────")

		for _, item := range items {
			sizeStr := formatByteSize(item.SizeBytes)
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\n", item.Repo, item.Version, item.Status, item.PRCount, sizeStr, item.Action)
		}
		_ = w.Flush()

		fmt.Println()
		fmt.Printf("Health: %d total (%d current, %d outdated, %d future, %d corrupt) | Target Package Version: %s\n",
			len(entries), currentCount, outdatedCount, futureCount, corruptCount, cache.CurrentVersion)

		return nil
	},
}

var cacheMigrateCmd = &cobra.Command{
	Use:     "migrate [owner/repo]",
	Aliases: []string{"upgrade"},
	Short:   "Migrate outdated cache files to the current schema version",
	Example: `  # Migrate cache for a specific repository
  gh pr-pro cache migrate cli/cli

  # Migrate all outdated cached repositories
  gh pr-pro cache migrate --all

  # Preview migration without writing to disk
  gh pr-pro cache migrate --all --dry-run`,
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := cache.NewCacheManager()
		if err != nil {
			return fmt.Errorf("failed to access cache manager: %w", err)
		}

		if flagCacheMigrateAll {
			count, err := mgr.MigrateAll(flagCacheMigrateDryRun)
			if err != nil {
				return fmt.Errorf("migration failed: %w", err)
			}
			if flagCacheMigrateDryRun {
				fmt.Printf("ℹ️  [dry-run] Found %d outdated cache file(s) eligible for migration to version %s\n", count, cache.CurrentVersion)
			} else {
				fmt.Printf("✅ Successfully migrated %d cache file(s) to version %s\n", count, cache.CurrentVersion)
			}
			return nil
		}

		targetRepo := ""
		if len(args) > 0 {
			targetRepo = args[0]
		} else if flagRepo != "" {
			targetRepo = flagRepo
		}

		if targetRepo == "" {
			return fmt.Errorf("specify a repository (e.g. 'gh pr-pro cache migrate owner/repo') or use '--all' to migrate everything")
		}

		fromVer, toVer, err := mgr.Migrate(targetRepo, flagCacheMigrateDryRun)
		if err != nil {
			return fmt.Errorf("failed to migrate cache for %s: %w", targetRepo, err)
		}

		if fromVer == toVer {
			fmt.Printf("ℹ️  Cache for %s is already up to date (version %s)\n", targetRepo, toVer)
			return nil
		}

		if flagCacheMigrateDryRun {
			fmt.Printf("ℹ️  [dry-run] Cache for %s would be migrated from version %s to %s\n", targetRepo, fromVer, toVer)
		} else {
			fmt.Printf("✅ Successfully migrated cache for %s from version %s to %s\n", targetRepo, fromVer, toVer)
		}

		return nil
	},
}

var cacheCleanCmd = &cobra.Command{
	Use:     "clean [owner/repo]",
	Aliases: []string{"clear", "delete", "rm"},
	Short:   "Delete cached PR records for a specific repository or all repositories",
	Example: `  # Delete cache for a specific repository
  gh pr-pro cache clean cli/cli

  # Delete only outdated cache entries
  gh pr-pro cache clean --outdated

  # Delete only corrupted cache entries
  gh pr-pro cache clean --corrupt

  # Delete all cached repositories
  gh pr-pro cache clean --all`,
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := cache.NewCacheManager()
		if err != nil {
			return fmt.Errorf("failed to access cache manager: %w", err)
		}

		if flagCacheOutdated {
			deleted, err := mgr.ClearOutdated()
			if err != nil {
				return fmt.Errorf("failed to clear outdated cache entries: %w", err)
			}
			fmt.Printf("✅ Successfully cleared %d outdated cache file(s) from %s\n", deleted, mgr.GetBaseDir())
			return nil
		}

		if flagCacheCorrupt {
			deleted, err := mgr.ClearCorrupt()
			if err != nil {
				return fmt.Errorf("failed to clear corrupted cache entries: %w", err)
			}
			fmt.Printf("✅ Successfully cleared %d corrupted cache file(s) from %s\n", deleted, mgr.GetBaseDir())
			return nil
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
			return fmt.Errorf("specify a repository (e.g. 'gh pr-pro cache clean owner/repo'), use '--outdated', '--corrupt', or use '--all' to clear everything")
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
	cacheStatusCmd.Flags().BoolVar(&flagCacheStatusJSON, "json", false, "Output cache status in JSON format")
	cacheMigrateCmd.Flags().BoolVar(&flagCacheMigrateAll, "all", false, "Migrate all outdated cached repository records")
	cacheMigrateCmd.Flags().BoolVar(&flagCacheMigrateDryRun, "dry-run", false, "Preview cache migration without modifying files on disk")
	cacheCleanCmd.Flags().BoolVar(&flagCacheAll, "all", false, "Clear all cached repository records")
	cacheCleanCmd.Flags().BoolVar(&flagCacheOutdated, "outdated", false, "Clear only outdated cache entries requiring refresh")
	cacheCleanCmd.Flags().BoolVar(&flagCacheCorrupt, "corrupt", false, "Clear only corrupted cache files with invalid JSON")

	cacheCmd.AddCommand(cacheListCmd)
	cacheCmd.AddCommand(cacheStatusCmd)
	cacheCmd.AddCommand(cacheMigrateCmd)
	cacheCmd.AddCommand(cacheCleanCmd)
	cacheCmd.AddCommand(cachePathCmd)
}
