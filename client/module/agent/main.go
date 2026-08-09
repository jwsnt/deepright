package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"agent-scanner/agentcore"
)

type Skill = agentcore.Skill
type Agent = agentcore.Agent
type AgentOutput = agentcore.Output

func detectGit() string {
	return agentcore.DetectGit()
}

func detectPlugins(appPath string) []string {
	return agentcore.DetectPlugins(appPath)
}

func GetAgentOutput(root string, deviceID string, ttl time.Duration) ([]byte, error) {
	return agentcore.GetOutputJSON(root, deviceID, ttl)
}

func GetAgentOutputForApp(root string, appDir string, deviceID string, ttl time.Duration) ([]byte, error) {
	return agentcore.GetOutputJSONForApp(root, appDir, deviceID, ttl)
}

func GetAgentOutputForAppAndChat(root string, appDir string, deviceID string, ttl time.Duration, chatID string) ([]byte, error) {
	return agentcore.GetOutputJSONForAppAndChat(root, appDir, deviceID, ttl, chatID)
}

func GetAgentIDs(root string, deviceID string, ttl time.Duration) ([]string, error) {
	return agentcore.GetAgentIDs(root, deviceID, ttl)
}

func GetAgentByID(root string, deviceID string, ttl time.Duration, agentID string) (*Agent, error) {
	return agentcore.GetAgentByID(root, deviceID, ttl, agentID)
}

func GetSkillNames(root string, deviceID string, ttl time.Duration, agentID string) ([]string, error) {
	return agentcore.GetSkillNames(root, deviceID, ttl, agentID)
}

func FlushCache() {
	agentcore.FlushCache()
}

// ── CLI entry point ─────────────────────────────────────────────────────────

func main() {
	cacheMs := flag.Int("agent-cache", 10000, "cache TTL in milliseconds (default 10000ms = 10s)")
	deviceFlag := flag.String("device", "", "device ID (default: auto-generated from hardware)")
	appDirFlag := flag.String("app-dir", "", "application startup directory used to resolve optional knowledge metadata")
	chatIDFlag := flag.String("chatId", "", "chat ID used to resolve per-session sandbox metadata")
	listIDs := flag.Bool("list", false, "list all agent IDs")
	getID := flag.String("get", "", "get agent metadata by agentId")
	skillsID := flag.String("skills", "", "list skill names by agentId")
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: agent-scanner [options] <directory>")
		fmt.Fprintln(os.Stderr, "  --list          list all agent IDs")
		fmt.Fprintln(os.Stderr, "  --get <id>      get agent metadata by agentId")
		fmt.Fprintln(os.Stderr, "  --skills <id>   list skill names by agentId")
		fmt.Fprintln(os.Stderr, "  --chatId <id>   resolve per-session sandbox metadata for that ChatId")
		fmt.Fprintln(os.Stderr, "  (no flag)       output full agent metadata")
		os.Exit(1)
	}

	root := args[0]
	ttl := time.Duration(*cacheMs) * time.Millisecond

	if *listIDs {
		ids, err := agentcore.GetAgentIDs(root, *deviceFlag, ttl)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		out, _ := json.MarshalIndent(ids, "", "  ")
		fmt.Println(string(out))
		return
	}

	if *getID != "" {
		output, err := agentcore.GetOutputForAppAndChat(root, *appDirFlag, *deviceFlag, ttl, *chatIDFlag)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		var agent *Agent
		for i := range output.Agents {
			if output.Agents[i].AgentID == *getID {
				cp := output.Agents[i]
				agent = &cp
				break
			}
		}
		if agent == nil {
			fmt.Fprintf(os.Stderr, "agent %q not found\n", *getID)
			os.Exit(1)
		}
		out, _ := json.MarshalIndent(agent, "", "  ")
		fmt.Println(string(out))
		return
	}

	if *skillsID != "" {
		names, err := agentcore.GetSkillNames(root, *deviceFlag, ttl, *skillsID)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
		out, _ := json.MarshalIndent(names, "", "  ")
		fmt.Println(string(out))
		return
	}

	data, err := GetAgentOutputForAppAndChat(root, *appDirFlag, *deviceFlag, ttl, *chatIDFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}
