package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/brad/gh-pr-pro/pkg/cmd"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func findReadmePath() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		candidate := filepath.Join(dir, "README.md")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("README.md not found in current directory or any parent")
}

func collectCommands(c *cobra.Command, path string) []string {
	var cmds []string
	name := c.Name()
	if name == "help" || name == "completion" {
		return nil
	}

	currentPath := name
	if path != "" {
		currentPath = path + " " + name
	}

	// Only add subcommands under root
	if path != "" {
		cmds = append(cmds, currentPath)
	}

	for _, sub := range c.Commands() {
		cmds = append(cmds, collectCommands(sub, currentPath)...)
	}
	return cmds
}

type flagInfo struct {
	Name      string
	Shorthand string
	Usage     string
	DefValue  string
	Command   string
}

func collectFlags(c *cobra.Command) map[string]flagInfo {
	flags := make(map[string]flagInfo)

	var walk func(cmd *cobra.Command)
	walk = func(cmd *cobra.Command) {
		if cmd.Name() == "help" || cmd.Name() == "completion" {
			return
		}

		addFlag := func(f *pflag.Flag) {
			if f.Name == "help" {
				return
			}
			if _, exists := flags[f.Name]; !exists {
				flags[f.Name] = flagInfo{
					Name:      f.Name,
					Shorthand: f.Shorthand,
					Usage:     f.Usage,
					DefValue:  f.DefValue,
					Command:   cmd.CommandPath(),
				}
			}
		}

		cmd.PersistentFlags().VisitAll(addFlag)
		cmd.Flags().VisitAll(addFlag)

		for _, sub := range cmd.Commands() {
			walk(sub)
		}
	}

	walk(c)
	return flags
}

func main() {
	readmePath, err := findReadmePath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	contentBytes, err := os.ReadFile(readmePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", readmePath, err)
		os.Exit(1)
	}
	readmeText := string(contentBytes)

	fmt.Printf("Auditing README (%s) against CLI API surface...\n\n", readmePath)

	// 1. Audit Commands
	commands := collectCommands(cmd.RootCmd, "")
	sort.Strings(commands)

	var missingCommands []string
	for _, c := range commands {
		// Each subcommand (e.g., "time merge") should be documented in README.
		parts := strings.Fields(c)
		subcommandLeaf := parts[len(parts)-1]
		if !strings.Contains(readmeText, subcommandLeaf) {
			missingCommands = append(missingCommands, c)
		}
	}

	// 2. Audit Flags
	flagMap := collectFlags(cmd.RootCmd)
	var flagNames []string
	for k := range flagMap {
		flagNames = append(flagNames, k)
	}
	sort.Strings(flagNames)

	var missingFlags []flagInfo
	var missingShorthands []flagInfo

	for _, name := range flagNames {
		info := flagMap[name]
		flagPattern := "--" + info.Name
		if !strings.Contains(readmeText, flagPattern) {
			missingFlags = append(missingFlags, info)
		}

		if info.Shorthand != "" {
			shortPattern := "-" + info.Shorthand
			if !strings.Contains(readmeText, shortPattern) {
				missingShorthands = append(missingShorthands, info)
			}
		}
	}

	// 3. Audit Critical Structural Sections
	requiredSections := []string{
		"Flag Scoping & Reference",
		"Command Hierarchy & Reference",
		"Global Flags",
		"Domain / Metric Flags",
		"Cache Flags",
		"time",
		"quality",
		"code",
		"team",
		"overview",
		"export",
		"cache",
	}

	var missingSections []string
	for _, section := range requiredSections {
		if !strings.Contains(readmeText, section) {
			missingSections = append(missingSections, section)
		}
	}

	// 4. Report Results
	hasError := false

	fmt.Println("--- Commands Check ---")
	if len(missingCommands) == 0 {
		fmt.Printf("✓ All %d CLI subcommands are documented in README.md\n", len(commands))
	} else {
		hasError = true
		fmt.Printf("✗ %d CLI subcommands are missing from README.md:\n", len(missingCommands))
		for _, c := range missingCommands {
			fmt.Printf("    - %s\n", c)
		}
	}

	fmt.Println("\n--- Flags Check ---")
	if len(missingFlags) == 0 {
		fmt.Printf("✓ All %d CLI flags are documented in README.md\n", len(flagNames))
	} else {
		hasError = true
		fmt.Printf("✗ %d CLI flags are missing from README.md:\n", len(missingFlags))
		for _, f := range missingFlags {
			fmt.Printf("    - --%s (from command: %s)\n", f.Name, f.Command)
		}
	}

	if len(missingShorthands) > 0 {
		fmt.Printf("\n! Warning: %d flag shorthands not found in README.md:\n", len(missingShorthands))
		for _, f := range missingShorthands {
			fmt.Printf("    - -%s for --%s\n", f.Shorthand, f.Name)
		}
	}

	fmt.Println("\n--- Structural Sections Check ---")
	if len(missingSections) == 0 {
		fmt.Printf("✓ All %d required structural sections are present in README.md\n", len(requiredSections))
	} else {
		hasError = true
		fmt.Printf("✗ %d required structural sections are missing from README.md:\n", len(missingSections))
		for _, s := range missingSections {
			fmt.Printf("    - %s\n", s)
		}
	}

	if hasError {
		fmt.Println("\nResult: FAILED - README.md is out of sync with CLI API surface area.")
		os.Exit(1)
	}

	fmt.Println("\nResult: SUCCESS - README.md is in sync with CLI API surface area.")
}
