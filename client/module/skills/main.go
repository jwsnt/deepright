package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"skill-scanner/skillscore"
)

type Skill = skillscore.Skill

func GetSkillsOutputJSON(root string, ttl time.Duration) ([]byte, error) {
	return skillscore.GetOutputJSON(root, ttl)
}

func main() {
	args := os.Args[1:]
	if len(args) > 0 {
		switch args[0] {
		case "warning-scan":
			runWarningScan(args[1:])
			return
		case "warning-list":
			runWarningList(args[1:])
			return
		case "help", "--help", "-h":
			printUsage()
			return
		}
	}

	runScan(args)
}

func runScan(args []string) {
	fs := flag.NewFlagSet("scan", flag.ExitOnError)
	cacheMs := fs.Int("skill-cache", 10000, "cache TTL in milliseconds")
	fs.Parse(args)

	rest := fs.Args()
	if len(rest) < 1 {
		fmt.Fprintln(os.Stderr, "usage: skill-scanner [--skill-cache ms] <directory>")
		os.Exit(1)
	}
	root := rest[0]
	ttl := time.Duration(*cacheMs) * time.Millisecond

	data, err := GetSkillsOutputJSON(root, ttl)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}

func runWarningScan(args []string) {
	fs := flag.NewFlagSet("warning-scan", flag.ExitOnError)
	dbPath := fs.String("db", "data", "sqlite database path")
	interval := fs.Duration("interval", time.Minute, "scan interval, 0 means run once")
	fs.Parse(args)

	rest := fs.Args()
	if len(rest) < 1 {
		fmt.Fprintln(os.Stderr, "usage: skill-scanner warning-scan [--db data] [--interval 1m] <directory>")
		os.Exit(1)
	}
	root := rest[0]

	run := func() error {
		warnings, err := skillscore.ScanAndSyncWarnings(root, *dbPath)
		if err != nil {
			return err
		}
		data, err := skillscore.OpenWarningStore(*dbPath)
		if err != nil {
			return err
		}
		out, err := data.ListJSON()
		if err != nil {
			return err
		}
		if len(warnings) == 0 {
			fmt.Println("[]")
			return nil
		}
		fmt.Println(string(out))
		return nil
	}

	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	if *interval <= 0 {
		return
	}

	ticker := time.NewTicker(*interval)
	defer ticker.Stop()
	for range ticker.C {
		if err := run(); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
		}
	}
}

func runWarningList(args []string) {
	fs := flag.NewFlagSet("warning-list", flag.ExitOnError)
	dbPath := fs.String("db", "data", "sqlite database path")
	fs.Parse(args)

	store, err := skillscore.OpenWarningStore(*dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	out, err := store.ListJSON()
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Println(string(out))
}

func printUsage() {
	fmt.Println(strings.TrimSpace(`
usage:
  skill-scanner [--skill-cache ms] <directory>
  skill-scanner warning-scan [--db data] [--interval 1m] <directory>
  skill-scanner warning-list [--db data]
`))
}
