package main

import (
	"fmt"
	"io"
	"strings"
)

const remoteHelpHeader = `Usage:
  remote <command> [options]

Commands:
  name        print plugin metadata key and display name
  param       print plugin metadata params
  command     print supported plugin command names
  scope       print supported plugin scopes ([])
  create      create or reuse one managed SSH connection
  shutdown    stop one managed SSH connection
  list        list live managed SSH connections
  get         read one managed SSH connection
  exec        execute one remote command through a managed SSH connection
  ssh         pass through to the local ssh binary
  scp         transfer files through the cached SSH session selected by --session
  help        show this manual

Managed SSH Workflow:
  ./remote create --agentId Agent-A --chatId Chat-001 --remote ubuntu@1.2.3.4 --password secret --port 10086
  ./remote create --agentId Agent-A --chatId Chat-002 --remote ubuntu@1.2.3.4 --certificate /path/to/id.pem --port 22
  ./remote create --agentId Agent-A --chatId Chat-001 --remote ubuntu@1.2.3.5 --password secret --port 22
  ./remote get --agentId agent-a --chatId chat-001 --remote ubuntu@1.2.3.4
  ./remote exec --session agent-a@chat-001 --remote ubuntu@1.2.3.4 "uname -a"
  ./remote scp ./local.txt ubuntu@1.2.3.4:/tmp/ --session agent-a@chat-001
  ./remote scp ubuntu@1.2.3.4:/tmp/local.txt . --session agent-a@chat-001
  ./remote shutdown --agentId agent-a --chatId chat-001 --remote ubuntu@1.2.3.4
  ./remote shutdown --agentId agent-a --chatId chat-001

Contract Notes:
  name returns {"key":"remote","name":"远程"}
  param returns [{"exec_timeout":"选填。SSH执行超时","scp_timeout":"选填。SCP执行超时"}]
  command returns the plugin capability list required by the plugin contract
  agentId and chatId are normalized to lowercase before matching or persisting
  managed session cache keys are scoped by agentId + chatId + remote
  remote.json is always written beside the remote binary
  remote.log is always written beside the remote binary
  each managed session is validated by daemon pid, daemon socket ownership, and binary fingerprint
  create, list, get, shutdown, and exec are served through the started manager daemon
  get and exec can target a specific cached host via --remote; when one agentId/chatId maps to multiple hosts, --remote is required
  exec reuses the cached SSH connection instead of opening a new login per request
  delegated ssh passthrough stays fixed on the local ssh binary and does not use fallback candidates
  scp reuses the cached SSH master connection selected by --session, auto-detects the remote endpoint from the scp source/target, and still follows system scp argument semantics
  create supports password mode and certificate mode; certificate mode maps to ssh -i <pem>

Examples:
  ./remote name
  ./remote param
  ./remote list
  ./remote ssh -V
  ./remote ssh ubuntu@1.2.3.4 "ls -la"
  ./remote scp ./artifact.txt ubuntu@1.2.3.4:/srv/tmp/ --session agent-a@chat-001
`

func printHelp(w io.Writer) {
	_, _ = io.WriteString(w, remoteHelpHeader)
	sshHelp := strings.TrimSpace(sshHelpText())
	if sshHelp == "" {
		return
	}
	fmt.Fprintln(w, "Delegated SSH Manual:")
	fmt.Fprintln(w, sshHelp)
}
