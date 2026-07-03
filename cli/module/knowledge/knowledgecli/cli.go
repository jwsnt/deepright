package knowledgecli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"knowledge/knowledgecore"
)

type Options struct {
	DefaultAppDir string
}

func Run(args []string, opts Options, stdout, stderr io.Writer) int {
	if len(args) < 1 {
		PrintUsage(stderr)
		return 1
	}

	switch strings.TrimSpace(args[0]) {
	case "ensure":
		return runEnsure(args[1:], opts, stdout, stderr)
	case "get":
		return runGet(args[1:], opts, stdout, stderr)
	case "metadata":
		return runMetadata(args[1:], opts, stdout, stderr)
	case "update-time":
		return runUpdateTime(args[1:], opts, stdout, stderr)
	case "update-commit":
		return runUpdateCommit(args[1:], opts, stdout, stderr)
	case "help", "-h", "--help":
		PrintUsage(stderr)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n\n", args[0])
		PrintUsage(stderr)
		return 1
	}
}

func PrintUsage(w io.Writer) {
	fmt.Fprintln(w, "usage: knowledge <command> [options]")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "commands:")
	fmt.Fprintln(w, "  ensure       ensure knowledge dir and sqlite state exist")
	fmt.Fprintln(w, "  get          print current knowledge state")
	fmt.Fprintln(w, "  metadata     print metadata fragment containing knowledge path")
	fmt.Fprintln(w, "  update-time  update last_update timestamp in shared sqlite")
	fmt.Fprintln(w, "  update-commit update knowledge_commit flag in shared sqlite")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "common option:")
	fmt.Fprintln(w, "  --app-dir    application startup directory (default: current directory)")
	fmt.Fprintln(w, "  --agent-id   agent identity for agent-scoped knowledge runtime")
}

func runEnsure(args []string, opts Options, stdout, stderr io.Writer) int {
	fs, appDir, agentID := newBaseFlagSet("ensure", opts)
	if err := fs.Parse(args); err != nil {
		return writeError(stderr, err)
	}
	data, err := knowledgecore.MarshalStateForAgent(*appDir, *agentID)
	if err != nil {
		return writeError(stderr, err)
	}
	fmt.Fprintln(stdout, string(data))
	return 0
}

func runGet(args []string, opts Options, stdout, stderr io.Writer) int {
	fs, appDir, agentID := newBaseFlagSet("get", opts)
	if err := fs.Parse(args); err != nil {
		return writeError(stderr, err)
	}
	data, err := knowledgecore.MarshalStateForAgent(*appDir, *agentID)
	if err != nil {
		return writeError(stderr, err)
	}
	fmt.Fprintln(stdout, string(data))
	return 0
}

func runMetadata(args []string, opts Options, stdout, stderr io.Writer) int {
	fs, appDir, agentID := newBaseFlagSet("metadata", opts)
	readOnly := fs.Bool("read-only", false, "only output knowledge metadata when it already exists")
	if err := fs.Parse(args); err != nil {
		return writeError(stderr, err)
	}

	var (
		metadata map[string]any
		err      error
	)
	if *readOnly {
		metadata, err = knowledgecore.MetadataIfExistsForAgent(*appDir, *agentID)
	} else {
		metadata, err = knowledgecore.MetadataForAgent(*appDir, *agentID)
	}
	if err != nil {
		return writeError(stderr, err)
	}
	if metadata == nil {
		metadata = map[string]any{}
	}
	out, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return writeError(stderr, err)
	}
	fmt.Fprintln(stdout, string(out))
	return 0
}

func runUpdateTime(args []string, opts Options, stdout, stderr io.Writer) int {
	fs, appDir, agentID := newBaseFlagSet("update-time", opts)
	tsRaw := fs.String("timestamp", "", "unix timestamp in milliseconds (default: now)")
	if err := fs.Parse(args); err != nil {
		return writeError(stderr, err)
	}

	ts := time.Now().UnixMilli()
	if strings.TrimSpace(*tsRaw) == "" {
		if rest := fs.Args(); len(rest) > 0 {
			*tsRaw = rest[0]
		}
	}
	if strings.TrimSpace(*tsRaw) != "" {
		parsed, err := strconv.ParseInt(strings.TrimSpace(*tsRaw), 10, 64)
		if err != nil {
			return writeError(stderr, fmt.Errorf("parse timestamp: %w", err))
		}
		ts = parsed
	}

	db, err := knowledgecore.OpenSharedDB(*appDir)
	if err != nil {
		return writeError(stderr, err)
	}
	if err := knowledgecore.SetLastUpdateForAgent(db, *agentID, ts); err != nil {
		return writeError(stderr, err)
	}

	data, err := knowledgecore.MarshalStateForAgent(*appDir, *agentID)
	if err != nil {
		return writeError(stderr, err)
	}
	fmt.Fprintln(stdout, string(data))
	return 0
}

func runUpdateCommit(args []string, opts Options, stdout, stderr io.Writer) int {
	fs, appDir, agentID := newBaseFlagSet("update-commit", opts)
	valueRaw := fs.String("value", "", "knowledge commit flag (true/false)")
	if err := fs.Parse(args); err != nil {
		return writeError(stderr, err)
	}

	if strings.TrimSpace(*valueRaw) == "" {
		if rest := fs.Args(); len(rest) > 0 {
			*valueRaw = rest[0]
		}
	}
	value, err := strconv.ParseBool(strings.TrimSpace(*valueRaw))
	if err != nil {
		return writeError(stderr, fmt.Errorf("parse commit value: %w", err))
	}

	db, err := knowledgecore.OpenSharedDB(*appDir)
	if err != nil {
		return writeError(stderr, err)
	}
	if err := knowledgecore.SetKnowledgeCommitForAgent(db, *agentID, value); err != nil {
		return writeError(stderr, err)
	}

	data, err := knowledgecore.MarshalStateForAgent(*appDir, *agentID)
	if err != nil {
		return writeError(stderr, err)
	}
	fmt.Fprintln(stdout, string(data))
	return 0
}

func newBaseFlagSet(name string, opts Options) (*flag.FlagSet, *string, *string) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	appDir := fs.String("app-dir", resolveDefaultAppDir(opts), "application startup directory")
	agentID := fs.String("agent-id", "", "agent identity for agent-scoped last_update")
	return fs, appDir, agentID
}

func resolveDefaultAppDir(opts Options) string {
	if appDir := strings.TrimSpace(opts.DefaultAppDir); appDir != "" {
		return appDir
	}
	return "."
}

func writeError(stderr io.Writer, err error) int {
	if err == nil {
		return 0
	}
	fmt.Fprintln(stderr, "error:", err)
	return 1
}
