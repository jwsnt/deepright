package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"connect/sharedutil"
)

type integrationStringSliceFlag []string

func (s *integrationStringSliceFlag) String() string {
	if s == nil {
		return ""
	}
	return strings.Join(*s, ",")
}

func (s *integrationStringSliceFlag) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	*s = append(*s, value)
	return nil
}

func integrationAPIRepeatedTaskValue(values []string, index, taskCount int, fallback, flagName string) (string, error) {
	switch len(values) {
	case 0:
		return fallback, nil
	case 1:
		return strings.TrimSpace(values[0]), nil
	case taskCount:
		return strings.TrimSpace(values[index]), nil
	default:
		return "", fmt.Errorf("%s must be provided once or once for every --textPath", flagName)
	}
}

type integrationAPIQueryFlag struct {
	QueryKey string
	Names    []string
	Usage    string
}

type integrationAPIGenericRequestSpec struct {
	Command       string
	Usage         string
	Description   string
	Method        string
	Path          string
	QueryFlags    []integrationAPIQueryFlag
	PathFlag      string
	PathUsage     string
	AllowOutput   bool
	RawOutput     bool
	RequireOutput bool
}

func printIntegrationAPIHelp() {
	fmt.Println("Usage:")
	fmt.Println("  integration api <command> [options]")
	fmt.Println("  integration service <command> [options]")
	fmt.Println("")
	fmt.Println("Description:")
	fmt.Println("  统一把 integration HTTP 接口收口成 CLI 命令。")
	fmt.Println("  `integration service ...` 是 `integration api ...` 的兼容别名。")
	fmt.Println("  其中已存在的本地 CLI 会优先复用；其余命令通过本机 integration HTTP 服务调用。")
	fmt.Println("")
	fmt.Println("Common options:")
	fmt.Println("  --addr URL        Integration HTTP base address, default http://127.0.0.1:<runtime port>")
	fmt.Println("  --port INT        Integration HTTP port; defaults to config/config.json port, then 8080")
	fmt.Println("  --output PATH     Save response body to file instead of stdout (for download/knowledge etc.)")
	fmt.Println("")
	fmt.Println("Commands:")
	fmt.Println("  chat-completions      Call POST /v1/chat/completions")
	fmt.Println("  heartbeat             Call GET /api/heartbeat")
	fmt.Println("  log-cleanup-status    Call GET /api/log_cleanup_status")
	fmt.Println("  agent-id              Call GET /api/agentId")
	fmt.Println("  swarm-agent           Call GET /api/swarm_agent")
	fmt.Println("  device-id             Call GET /api/deviceId")
	fmt.Println("  folder                Call GET /api/folder")
	fmt.Println("  skills                Call GET /api/skills")
	fmt.Println("  skill-state           Call POST /api/skill_state")
	fmt.Println("  files                 Call GET /api/files")
	fmt.Println("  data                  Call GET /api/data")
	fmt.Println("  workspace             Call GET /api/workspace")
	fmt.Println("  url-preview-probe     Call GET /api/url_preview_probe")
	fmt.Println("  edit                  Call POST /api/edit")
	fmt.Println("  del                   Call GET /api/del")
	fmt.Println("  raw                   Call GET /api/raw")
	fmt.Println("  file-last-update      Call GET /file/lastUpdate")
	fmt.Println("  host                  Reuse the service-address CLI for /api/host")
	fmt.Println("  standalone            Reuse existing runtime standalone CLI for /api/standalone")
	fmt.Println("  site-access           Call GET /api/site/access")
	fmt.Println("  agent                 Agent-related endpoints")
	fmt.Println("  upload                Call POST /api/upload")
	fmt.Println("  config                Call POST /api/config")
	fmt.Println("  token                 GET/POST /api/token")
	fmt.Println("  consume               Call GET /api/consume")
	fmt.Println("  message-insert        Message insert endpoints")
	fmt.Println("  sandbox               Reuse existing sandbox CLI for /api/sandbox*")
	fmt.Println("  connect               Connect endpoints")
	fmt.Println("  cron                  Cron endpoints")
	fmt.Println("  cancel                Call POST /api/cancel")
	fmt.Println("  restore               Call POST /api/restore")
	fmt.Println("  chat-session-log      Call GET /api/chat_session_log")
	fmt.Println("  download              Call GET /api/download")
	fmt.Println("  cmd                   Call POST /api/cmd")
	fmt.Println("  kill                  Call POST /api/kill")
	fmt.Println("  plugins               Plugin endpoints")
	fmt.Println("  skills-warning        Call GET /skills_warning")
	fmt.Println("  install-app           Call GET /install_app")
	fmt.Println("  log-round             Reuse existing /log_round CLI")
	fmt.Println("  log-skill             Call GET /log_skill")
	fmt.Println("  log-skill-status      Call GET /log_skill_status")
	fmt.Println("  knowledge             Call GET /knowledge[/path]")
	fmt.Println("  knowledge-last-update Call GET /knowledge_lastUpdate")
	fmt.Println("  knowledge-path        Call GET /knowledge_path")
	fmt.Println("  whisper               Whisper audio transcription queue endpoints")
	fmt.Println("  rembg                 Image subject extraction queue endpoints")
	fmt.Println("  voxcpm                VoxCPM text-to-speech queue endpoints")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  integration api heartbeat")
	fmt.Println("  integration api log-cleanup-status")
	fmt.Println("  integration api agent-id")
	fmt.Println("  integration api edit --agentId demo --path USER.md --content '# hello'")
	fmt.Println("  integration api token get")
	fmt.Println("  integration api token set --body-file ./token.json")
	fmt.Println("  integration api cron create --help")
	fmt.Println("  integration api whisper create --agentId demo-agent --path audios/meeting.mp3 --scenario chinese_meeting")
	fmt.Println("  integration api rembg create --agentId demo-agent --path images/product.jpg --model u2net --alpha-matting")
	fmt.Println("  integration api voxcpm create --agentId demo-agent --textPath scripts/intro.txt --outputName intro.wav")
	fmt.Println("  integration service cancel --chat chat-001")
}

func resolveIntegrationAPIBase(addr string, port int) (string, error) {
	if err := validateOptionalIntegrationServicePort(port); err != nil {
		return "", err
	}
	return resolveIntegrationHostAPIBase(addr, port), nil
}

func newIntegrationAPIClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	return &http.Client{Timeout: timeout}
}

func buildIntegrationAPIURL(base, rawPath string, query url.Values) (string, error) {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return "", fmt.Errorf("api base address is required")
	}
	ref, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	pathValue := strings.TrimSpace(rawPath)
	if pathValue == "" {
		pathValue = "/"
	}
	ref.Path = strings.TrimRight(ref.Path, "/") + pathValue
	ref.RawPath = ""
	if len(query) > 0 {
		ref.RawQuery = query.Encode()
	}
	return ref.String(), nil
}

func appendEscapedURLPath(basePath, relPath string) string {
	basePath = strings.TrimRight(strings.TrimSpace(basePath), "/")
	relPath = strings.TrimSpace(relPath)
	if relPath == "" || relPath == "." || relPath == "/" {
		if basePath == "" {
			return "/"
		}
		return basePath
	}
	relPath = strings.TrimPrefix(relPath, "/")
	parts := strings.Split(relPath, "/")
	escaped := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		escaped = append(escaped, url.PathEscape(part))
	}
	if len(escaped) == 0 {
		if basePath == "" {
			return "/"
		}
		return basePath
	}
	if basePath == "" {
		return "/" + strings.Join(escaped, "/")
	}
	return basePath + "/" + strings.Join(escaped, "/")
}

func integrationAPIRequest(ctx context.Context, client *http.Client, method, base, rawPath string, query url.Values, body io.Reader, contentType string, headers map[string]string) (*http.Response, error) {
	target, err := buildIntegrationAPIURL(base, rawPath, query)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(contentType) != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for key, value := range headers {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		req.Header.Set(key, value)
	}
	return client.Do(req)
}

func integrationAPIPrettyBody(raw []byte, contentType string, pretty bool) []byte {
	if !pretty {
		return raw
	}
	if !strings.Contains(strings.ToLower(contentType), "json") {
		trimmed := bytes.TrimSpace(raw)
		if len(trimmed) == 0 || (trimmed[0] != '{' && trimmed[0] != '[') {
			return raw
		}
	}
	var buf bytes.Buffer
	if err := json.Indent(&buf, bytes.TrimSpace(raw), "", "  "); err != nil {
		return raw
	}
	buf.WriteByte('\n')
	return buf.Bytes()
}

func integrationAPIWriteHTTPResponse(stdout io.Writer, resp *http.Response, outputPath string, pretty bool, rawOutput bool) error {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if strings.TrimSpace(outputPath) != "" {
		if err := os.WriteFile(outputPath, body, 0o644); err != nil {
			return err
		}
		return nil
	}
	if !rawOutput {
		body = integrationAPIPrettyBody(body, resp.Header.Get("Content-Type"), pretty)
	}
	if len(body) == 0 {
		return nil
	}
	_, err = stdout.Write(body)
	return err
}

func integrationAPIHandleHTTPResult(stdout, stderr io.Writer, resp *http.Response, err error, outputPath string, pretty bool, rawOutput bool) int {
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		_ = integrationAPIWriteHTTPResponse(stderr, resp, "", pretty, rawOutput)
		if resp.StatusCode >= 400 {
			return 1
		}
		return 0
	}
	if err := integrationAPIWriteHTTPResponse(stdout, resp, outputPath, pretty, rawOutput); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	if strings.TrimSpace(outputPath) != "" {
		fmt.Fprintf(stdout, "%s\n", outputPath)
	}
	return 0
}

func newIntegrationAPICommonFlagSet(name, usage, description string, stderr io.Writer) (*flag.FlagSet, *string, *int, *string, *bool) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	addr := fs.String("addr", "", "integration HTTP base address")
	port := fs.Int("port", 0, "integration HTTP port")
	output := fs.String("output", "", "write response body to file")
	pretty := fs.Bool("pretty", true, "pretty-print JSON responses")
	fs.Usage = func() {
		fmt.Fprintln(fs.Output(), "Usage:")
		fmt.Fprintf(fs.Output(), "  %s\n", usage)
		if strings.TrimSpace(description) != "" {
			fmt.Fprintln(fs.Output(), "")
			fmt.Fprintln(fs.Output(), description)
		}
		fmt.Fprintln(fs.Output(), "")
		fs.PrintDefaults()
	}
	return fs, addr, port, output, pretty
}

func addIntegrationAPIStringFlags(fs *flag.FlagSet, specs []integrationAPIQueryFlag) map[string]map[string]*string {
	values := make(map[string]map[string]*string, len(specs))
	for _, spec := range specs {
		if len(spec.Names) == 0 {
			continue
		}
		current := make(map[string]*string, len(spec.Names))
		for _, name := range spec.Names {
			ptr := fs.String(name, "", spec.Usage)
			current[name] = ptr
		}
		values[spec.QueryKey] = current
	}
	return values
}

func buildIntegrationAPIQueryValues(specs []integrationAPIQueryFlag, values map[string]map[string]*string) url.Values {
	query := url.Values{}
	for _, spec := range specs {
		candidates := values[spec.QueryKey]
		for _, name := range spec.Names {
			ptr := candidates[name]
			if ptr == nil {
				continue
			}
			value := strings.TrimSpace(*ptr)
			if value == "" {
				continue
			}
			query.Set(spec.QueryKey, value)
			break
		}
	}
	return query
}

func runIntegrationAPIGenericRequestCLI(spec integrationAPIGenericRequestSpec, args []string, stdout, stderr io.Writer) int {
	fs, addr, port, output, pretty := newIntegrationAPICommonFlagSet("integration api "+spec.Command, spec.Usage, spec.Description, stderr)
	valueFlags := addIntegrationAPIStringFlags(fs, spec.QueryFlags)
	pathFlag := (*string)(nil)
	if strings.TrimSpace(spec.PathFlag) != "" {
		pathFlag = fs.String(spec.PathFlag, "", spec.PathUsage)
	}
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 1
	}
	if fs.NArg() != 0 {
		fmt.Fprintf(stderr, "unexpected arguments: %s\n", strings.Join(fs.Args(), " "))
		return 1
	}
	if spec.RequireOutput && strings.TrimSpace(*output) == "" {
		fmt.Fprintln(stderr, "--output is required")
		return 1
	}
	base, err := resolveIntegrationAPIBase(*addr, *port)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	targetPath := spec.Path
	if pathFlag != nil && strings.TrimSpace(*pathFlag) != "" {
		targetPath = appendEscapedURLPath(spec.Path, *pathFlag)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	resp, reqErr := integrationAPIRequest(ctx, newIntegrationAPIClient(60*time.Second), spec.Method, base, targetPath, buildIntegrationAPIQueryValues(spec.QueryFlags, valueFlags), nil, "", nil)
	return integrationAPIHandleHTTPResult(stdout, stderr, resp, reqErr, *output, *pretty, spec.RawOutput)
}

func loadIntegrationAPIRawBody(raw, path string) ([]byte, error) {
	raw = strings.TrimSpace(raw)
	path = strings.TrimSpace(path)
	switch {
	case raw != "" && path != "":
		return nil, fmt.Errorf("--body and --body-file cannot be used together")
	case path != "":
		return os.ReadFile(path)
	case raw != "":
		return []byte(raw), nil
	default:
		return nil, fmt.Errorf("request body is required")
	}
}

func runIntegrationAPIRawBodyCLI(command, usage, description, method, path string, queryFlags []integrationAPIQueryFlag, args []string, stdout, stderr io.Writer) int {
	fs, addr, port, output, pretty := newIntegrationAPICommonFlagSet("integration api "+command, usage, description, stderr)
	valueFlags := addIntegrationAPIStringFlags(fs, queryFlags)
	bodyRaw := fs.String("body", "", "raw request body")
	bodyFile := fs.String("body-file", "", "read request body from file")
	contentType := fs.String("content-type", "application/json", "request content-type")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 1
	}
	if fs.NArg() != 0 {
		fmt.Fprintf(stderr, "unexpected arguments: %s\n", strings.Join(fs.Args(), " "))
		return 1
	}
	body, err := loadIntegrationAPIRawBody(*bodyRaw, *bodyFile)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	base, err := resolveIntegrationAPIBase(*addr, *port)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	resp, reqErr := integrationAPIRequest(ctx, newIntegrationAPIClient(60*time.Second), method, base, path, buildIntegrationAPIQueryValues(queryFlags, valueFlags), bytes.NewReader(body), strings.TrimSpace(*contentType), nil)
	return integrationAPIHandleHTTPResult(stdout, stderr, resp, reqErr, *output, *pretty, false)
}

func printIntegrationAPIWhisperHelp(stdout io.Writer) {
	fmt.Fprintln(stdout, "Usage:")
	fmt.Fprintln(stdout, "  integration api whisper <check|list|create|cancel|restart|delete|log> [options]")
	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "Commands:")
	fmt.Fprintln(stdout, "  check       Check whether the controlled environment can run Whisper")
	fmt.Fprintln(stdout, "  list        List one Agent's transcription tasks, five tasks per page")
	fmt.Fprintln(stdout, "  create      Queue one or more audio files in one Agent workspace")
	fmt.Fprintln(stdout, "  cancel      Cancel a queued or running task")
	fmt.Fprintln(stdout, "  restart     Requeue one cancelled task")
	fmt.Fprintln(stdout, "  delete      Delete one failed task record; output files are preserved")
	fmt.Fprintln(stdout, "  log         Read a task and its persisted execution log")
	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "Examples:")
	fmt.Fprintln(stdout, "  integration api whisper check")
	fmt.Fprintln(stdout, "  integration api whisper list --agentId demo-agent --status running --page 1")
	fmt.Fprintln(stdout, "  integration api whisper create --agentId demo-agent --path audios/meeting.mp3 --scenario chinese_meeting")
	fmt.Fprintln(stdout, "  integration api whisper cancel --agentId demo-agent --id 12")
	fmt.Fprintln(stdout, "  integration api whisper log --agentId demo-agent --id 12")
}

func runIntegrationAPIWhisperCreateCLI(args []string, stdout, stderr io.Writer) int {
	fs, addr, port, output, pretty := newIntegrationAPICommonFlagSet(
		"integration api whisper create",
		"integration api whisper create --agentId ID --path AUDIO_PATH [--path AUDIO_PATH ...] [--scenario SCENARIO] [--addr URL] [--port 8080]",
		"Call POST /api/whisper/tasks. Each path must be an audio file under the Agent workspace.",
		stderr,
	)
	agentID := fs.String("agentId", "", "agent id")
	fs.StringVar(agentID, "agent", "", "agent id")
	var paths integrationStringSliceFlag
	fs.Var(&paths, "path", "workspace-relative audio path; may be repeated")
	scenario := fs.String("scenario", "", "transcription scenario: chinese_meeting, chinese_accurate, realtime, batch, cpu, or mixed_technical")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 1
	}
	if fs.NArg() != 0 {
		fmt.Fprintf(stderr, "unexpected arguments: %s\n", strings.Join(fs.Args(), " "))
		return 1
	}
	*agentID = strings.TrimSpace(*agentID)
	cleanedPaths := make([]string, 0, len(paths))
	for _, path := range paths {
		if path = strings.TrimSpace(path); path != "" {
			cleanedPaths = append(cleanedPaths, path)
		}
	}
	if *agentID == "" {
		fmt.Fprintln(stderr, "--agentId is required")
		return 1
	}
	if len(cleanedPaths) == 0 {
		fmt.Fprintln(stderr, "at least one --path is required")
		return 1
	}
	body, err := json.Marshal(whisperTaskCreateRequest{AgentID: *agentID, Paths: cleanedPaths, Scenario: strings.TrimSpace(*scenario)})
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	base, err := resolveIntegrationAPIBase(*addr, *port)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	resp, reqErr := integrationAPIRequest(ctx, newIntegrationAPIClient(60*time.Second), http.MethodPost, base, "/api/whisper/tasks", nil, bytes.NewReader(body), "application/json", nil)
	return integrationAPIHandleHTTPResult(stdout, stderr, resp, reqErr, *output, *pretty, false)
}

func runIntegrationAPIWhisperTaskActionCLI(command, path, description string, args []string, stdout, stderr io.Writer) int {
	fs, addr, port, output, pretty := newIntegrationAPICommonFlagSet(
		"integration api whisper "+command,
		"integration api whisper "+command+" --agentId ID --id TASK_ID [--addr URL] [--port 8080]",
		description,
		stderr,
	)
	agentID := fs.String("agentId", "", "agent id")
	fs.StringVar(agentID, "agent", "", "agent id")
	taskID := fs.Int64("id", 0, "whisper task id")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 1
	}
	if fs.NArg() != 0 {
		fmt.Fprintf(stderr, "unexpected arguments: %s\n", strings.Join(fs.Args(), " "))
		return 1
	}
	*agentID = strings.TrimSpace(*agentID)
	if *agentID == "" {
		fmt.Fprintln(stderr, "--agentId is required")
		return 1
	}
	if *taskID <= 0 {
		fmt.Fprintln(stderr, "--id must be a positive integer")
		return 1
	}
	body, err := json.Marshal(map[string]interface{}{"agentId": *agentID, "id": *taskID})
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	base, err := resolveIntegrationAPIBase(*addr, *port)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	resp, reqErr := integrationAPIRequest(ctx, newIntegrationAPIClient(60*time.Second), http.MethodPost, base, path, nil, bytes.NewReader(body), "application/json", nil)
	return integrationAPIHandleHTTPResult(stdout, stderr, resp, reqErr, *output, *pretty, false)
}

func runIntegrationAPIWhisperCLI(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "--help" || strings.TrimSpace(args[0]) == "-h" || strings.TrimSpace(args[0]) == "help" {
		printIntegrationAPIWhisperHelp(stdout)
		return 0
	}
	switch strings.TrimSpace(args[0]) {
	case "check":
		return runIntegrationAPIGenericRequestCLI(integrationAPIGenericRequestSpec{
			Command: "whisper check", Usage: "integration api whisper check [--addr URL] [--port 8080]",
			Description: "Call GET /api/whisper/check.", Method: http.MethodGet, Path: "/api/whisper/check",
		}, args[1:], stdout, stderr)
	case "list":
		return runIntegrationAPIGenericRequestCLI(integrationAPIGenericRequestSpec{
			Command: "whisper list", Usage: "integration api whisper list --agentId ID [--status all|queued|running|completed|cancelled|failed] [--page N] [--addr URL] [--port 8080]",
			Description: "Call GET /api/whisper/tasks. The server returns five tasks per page.", Method: http.MethodGet, Path: "/api/whisper/tasks",
			QueryFlags: []integrationAPIQueryFlag{
				{QueryKey: "agentId", Names: []string{"agentId", "agent"}, Usage: "agent id"},
				{QueryKey: "status", Names: []string{"status"}, Usage: "all, queued, running, completed, cancelled, or failed"},
				{QueryKey: "page", Names: []string{"page"}, Usage: "one-based page number"},
			},
		}, args[1:], stdout, stderr)
	case "create":
		return runIntegrationAPIWhisperCreateCLI(args[1:], stdout, stderr)
	case "cancel":
		return runIntegrationAPIWhisperTaskActionCLI("cancel", "/api/whisper/tasks/cancel", "Call POST /api/whisper/tasks/cancel.", args[1:], stdout, stderr)
	case "restart":
		return runIntegrationAPIWhisperTaskActionCLI("restart", "/api/whisper/tasks/restart", "Call POST /api/whisper/tasks/restart for one cancelled task.", args[1:], stdout, stderr)
	case "delete":
		return runIntegrationAPIWhisperTaskActionCLI("delete", "/api/whisper/tasks/delete", "Call POST /api/whisper/tasks/delete for one failed task record.", args[1:], stdout, stderr)
	case "log":
		return runIntegrationAPIGenericRequestCLI(integrationAPIGenericRequestSpec{
			Command: "whisper log", Usage: "integration api whisper log --agentId ID --id TASK_ID [--addr URL] [--port 8080]",
			Description: "Call GET /api/whisper/tasks/log.", Method: http.MethodGet, Path: "/api/whisper/tasks/log",
			QueryFlags: []integrationAPIQueryFlag{
				{QueryKey: "agentId", Names: []string{"agentId", "agent"}, Usage: "agent id"},
				{QueryKey: "id", Names: []string{"id"}, Usage: "whisper task id"},
			},
		}, args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown whisper api command: %s\n", args[0])
		printIntegrationAPIWhisperHelp(stderr)
		return 1
	}
}

func printIntegrationAPIRembgHelp(stdout io.Writer) {
	fmt.Fprintln(stdout, "Usage:")
	fmt.Fprintln(stdout, "  integration api rembg <check|list|create|cancel|restart|delete|log> [options]")
	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "Commands:")
	fmt.Fprintln(stdout, "  check       Check whether the controlled environment can run rembg")
	fmt.Fprintln(stdout, "  list        List one Agent's image subject extraction tasks, five tasks per page")
	fmt.Fprintln(stdout, "  create      Queue one or more image files in one Agent workspace")
	fmt.Fprintln(stdout, "  cancel      Cancel a queued or running task")
	fmt.Fprintln(stdout, "  restart     Requeue one cancelled task")
	fmt.Fprintln(stdout, "  delete      Delete one failed task record; source and output files are preserved")
	fmt.Fprintln(stdout, "  log         Read a task and its persisted execution log")
	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "Create options:")
	fmt.Fprintln(stdout, "  --path PATH          Workspace-relative image path; may be repeated")
	fmt.Fprintln(stdout, "  --model MODEL        u2net (default), u2net_human_seg, u2netp, u2net_cloth_seg,")
	fmt.Fprintln(stdout, "                       silueta, isnet-general-use, or isnet-anime")
	fmt.Fprintln(stdout, "  --alpha-matting      Enable alpha-matting edge refinement for every submitted image")
	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "Examples:")
	fmt.Fprintln(stdout, "  integration api rembg check")
	fmt.Fprintln(stdout, "  integration api rembg list --agentId demo-agent --status running --page 1")
	fmt.Fprintln(stdout, "  integration api rembg create --agentId demo-agent --path images/product.jpg --model u2net --alpha-matting")
	fmt.Fprintln(stdout, "  integration api rembg create --agentId demo-agent --path images/a.jpg --path images/b.png --model isnet-general-use")
	fmt.Fprintln(stdout, "  integration api rembg cancel --agentId demo-agent --id 12")
	fmt.Fprintln(stdout, "  integration api rembg log --agentId demo-agent --id 12")
}

func runIntegrationAPIRembgCreateCLI(args []string, stdout, stderr io.Writer) int {
	fs, addr, port, output, pretty := newIntegrationAPICommonFlagSet(
		"integration api rembg create",
		"integration api rembg create --agentId ID --path IMAGE_PATH [--path IMAGE_PATH ...] [--model MODEL] [--alpha-matting] [--addr URL] [--port 8080]",
		"Call POST /api/rembg/tasks. Each path must be an image file under the Agent workspace. The selected model and alpha-matting option apply to every submitted image.",
		stderr,
	)
	agentID := fs.String("agentId", "", "agent id")
	fs.StringVar(agentID, "agent", "", "agent id")
	var paths integrationStringSliceFlag
	fs.Var(&paths, "path", "workspace-relative image path; may be repeated")
	model := fs.String("model", rembgDefaultModel, "rembg model")
	var alphaMatting bool
	fs.BoolVar(&alphaMatting, "alpha-matting", false, "enable alpha-matting edge refinement")
	fs.BoolVar(&alphaMatting, "alphaMatting", false, "enable alpha-matting edge refinement")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 1
	}
	if fs.NArg() != 0 {
		fmt.Fprintf(stderr, "unexpected arguments: %s\n", strings.Join(fs.Args(), " "))
		return 1
	}
	*agentID = strings.TrimSpace(*agentID)
	cleanedPaths := make([]string, 0, len(paths))
	for _, path := range paths {
		if path = strings.TrimSpace(path); path != "" {
			cleanedPaths = append(cleanedPaths, path)
		}
	}
	if *agentID == "" {
		fmt.Fprintln(stderr, "--agentId is required")
		return 1
	}
	if len(cleanedPaths) == 0 {
		fmt.Fprintln(stderr, "at least one --path is required")
		return 1
	}
	request := rembgTaskCreateRequest{AgentID: *agentID, Paths: cleanedPaths, Model: strings.TrimSpace(*model), AlphaMatting: alphaMatting}
	body, err := json.Marshal(request)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	base, err := resolveIntegrationAPIBase(*addr, *port)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	resp, reqErr := integrationAPIRequest(ctx, newIntegrationAPIClient(60*time.Second), http.MethodPost, base, "/api/rembg/tasks", nil, bytes.NewReader(body), "application/json", nil)
	return integrationAPIHandleHTTPResult(stdout, stderr, resp, reqErr, *output, *pretty, false)
}

func runIntegrationAPIRembgTaskActionCLI(command, path, description string, args []string, stdout, stderr io.Writer) int {
	fs, addr, port, output, pretty := newIntegrationAPICommonFlagSet(
		"integration api rembg "+command,
		"integration api rembg "+command+" --agentId ID --id TASK_ID [--addr URL] [--port 8080]",
		description,
		stderr,
	)
	agentID := fs.String("agentId", "", "agent id")
	fs.StringVar(agentID, "agent", "", "agent id")
	taskID := fs.Int64("id", 0, "rembg task id")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 1
	}
	if fs.NArg() != 0 {
		fmt.Fprintf(stderr, "unexpected arguments: %s\n", strings.Join(fs.Args(), " "))
		return 1
	}
	*agentID = strings.TrimSpace(*agentID)
	if *agentID == "" {
		fmt.Fprintln(stderr, "--agentId is required")
		return 1
	}
	if *taskID <= 0 {
		fmt.Fprintln(stderr, "--id must be a positive integer")
		return 1
	}
	body, err := json.Marshal(rembgTaskActionRequest{AgentID: *agentID, ID: *taskID})
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	base, err := resolveIntegrationAPIBase(*addr, *port)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	resp, reqErr := integrationAPIRequest(ctx, newIntegrationAPIClient(60*time.Second), http.MethodPost, base, path, nil, bytes.NewReader(body), "application/json", nil)
	return integrationAPIHandleHTTPResult(stdout, stderr, resp, reqErr, *output, *pretty, false)
}

func runIntegrationAPIRembgCLI(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "--help" || strings.TrimSpace(args[0]) == "-h" || strings.TrimSpace(args[0]) == "help" {
		printIntegrationAPIRembgHelp(stdout)
		return 0
	}
	switch strings.TrimSpace(args[0]) {
	case "check":
		return runIntegrationAPIGenericRequestCLI(integrationAPIGenericRequestSpec{
			Command: "rembg check", Usage: "integration api rembg check [--addr URL] [--port 8080]",
			Description: "Call GET /api/rembg/check.", Method: http.MethodGet, Path: "/api/rembg/check",
		}, args[1:], stdout, stderr)
	case "list":
		return runIntegrationAPIGenericRequestCLI(integrationAPIGenericRequestSpec{
			Command: "rembg list", Usage: "integration api rembg list --agentId ID [--status all|queued|running|completed|cancelled|failed] [--page N] [--addr URL] [--port 8080]",
			Description: "Call GET /api/rembg/tasks. The server returns five tasks per page.", Method: http.MethodGet, Path: "/api/rembg/tasks",
			QueryFlags: []integrationAPIQueryFlag{
				{QueryKey: "agentId", Names: []string{"agentId", "agent"}, Usage: "agent id"},
				{QueryKey: "status", Names: []string{"status"}, Usage: "all, queued, running, completed, cancelled, or failed"},
				{QueryKey: "page", Names: []string{"page"}, Usage: "one-based page number"},
			},
		}, args[1:], stdout, stderr)
	case "create":
		return runIntegrationAPIRembgCreateCLI(args[1:], stdout, stderr)
	case "cancel":
		return runIntegrationAPIRembgTaskActionCLI("cancel", "/api/rembg/tasks/cancel", "Call POST /api/rembg/tasks/cancel.", args[1:], stdout, stderr)
	case "restart":
		return runIntegrationAPIRembgTaskActionCLI("restart", "/api/rembg/tasks/restart", "Call POST /api/rembg/tasks/restart for one cancelled task.", args[1:], stdout, stderr)
	case "delete":
		return runIntegrationAPIRembgTaskActionCLI("delete", "/api/rembg/tasks/delete", "Call POST /api/rembg/tasks/delete for one failed task record.", args[1:], stdout, stderr)
	case "log":
		return runIntegrationAPIGenericRequestCLI(integrationAPIGenericRequestSpec{
			Command: "rembg log", Usage: "integration api rembg log --agentId ID --id TASK_ID [--addr URL] [--port 8080]",
			Description: "Call GET /api/rembg/tasks/log.", Method: http.MethodGet, Path: "/api/rembg/tasks/log",
			QueryFlags: []integrationAPIQueryFlag{
				{QueryKey: "agentId", Names: []string{"agentId", "agent"}, Usage: "agent id"},
				{QueryKey: "id", Names: []string{"id"}, Usage: "rembg task id"},
			},
		}, args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown rembg api command: %s\n", args[0])
		printIntegrationAPIRembgHelp(stderr)
		return 1
	}
}

func printIntegrationAPIVoxCPMHelp(stdout io.Writer) {
	fmt.Fprintln(stdout, "Usage:")
	fmt.Fprintln(stdout, "  integration api voxcpm <check|list|create|cancel|restart|delete|log> [options]")
	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "Commands:")
	fmt.Fprintln(stdout, "  check       Check whether the controlled environment can run VoxCPM")
	fmt.Fprintln(stdout, "  list        List one Agent's text-to-speech tasks, five tasks per page")
	fmt.Fprintln(stdout, "  create      Queue one to five text-to-speech tasks")
	fmt.Fprintln(stdout, "  cancel      Cancel a queued or running task")
	fmt.Fprintln(stdout, "  restart     Requeue one cancelled task")
	fmt.Fprintln(stdout, "  delete      Delete one failed task record; source and output files are preserved")
	fmt.Fprintln(stdout, "  log         Read a task and its persisted execution log")
	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "Create options:")
	fmt.Fprintln(stdout, "  --textPath PATH              Workspace-relative UTF-8 text path; may be repeated")
	fmt.Fprintln(stdout, "  --referenceAudioPath PATH    Optional reference audio path; one value applies to all tasks,")
	fmt.Fprintln(stdout, "                               or provide one value for each text path")
	fmt.Fprintln(stdout, "  --scenario NAME              balanced (default), quality, fast, clean, warm_narration, or lively")
	fmt.Fprintln(stdout, "  --control TEXT               Optional expression-style instruction; one value applies to all")
	fmt.Fprintln(stdout, "                               tasks, or provide one value for each text path")
	fmt.Fprintln(stdout, "  --outputName NAME.wav        Required output name; batch tasks receive collision-safe names")
	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "Examples:")
	fmt.Fprintln(stdout, "  integration api voxcpm check")
	fmt.Fprintln(stdout, "  integration api voxcpm list --agentId demo-agent --status running --page 1")
	fmt.Fprintln(stdout, "  integration api voxcpm create --agentId demo-agent --textPath scripts/intro.txt --outputName intro.wav --scenario warm_narration")
	fmt.Fprintln(stdout, "  integration api voxcpm create --agentId demo-agent --textPath scripts/a.txt --textPath scripts/b.txt --outputName narration.wav --referenceAudioPath audios/voice.wav")
	fmt.Fprintln(stdout, "  integration api voxcpm cancel --agentId demo-agent --id 12")
	fmt.Fprintln(stdout, "  integration api voxcpm log --agentId demo-agent --id 12")
}

func runIntegrationAPIVoxCPMCreateCLI(args []string, stdout, stderr io.Writer) int {
	fs, addr, port, output, pretty := newIntegrationAPICommonFlagSet("integration api voxcpm create", "integration api voxcpm create --agentId ID --textPath PATH [--textPath PATH ...] --outputName NAME.wav [--referenceAudioPath PATH] [--scenario SCENARIO] [--control TEXT] [--addr URL] [--port 8080]", "Call POST /api/voxcpm/tasks with one to five voice-design or cloning tasks. Repeated reference-audio, scenario, and control options must occur once or once per text path.", stderr)
	agentID := fs.String("agentId", "", "agent id")
	fs.StringVar(agentID, "agent", "", "agent id")
	var textPaths integrationStringSliceFlag
	var referencePaths integrationStringSliceFlag
	var scenarios integrationStringSliceFlag
	var controls integrationStringSliceFlag
	fs.Var(&textPaths, "textPath", "workspace-relative UTF-8 text file; may be repeated")
	fs.Var(&textPaths, "text-path", "workspace-relative UTF-8 text file; may be repeated")
	fs.Var(&referencePaths, "referenceAudioPath", "workspace-relative reference audio file; may be repeated")
	fs.Var(&referencePaths, "reference-audio-path", "workspace-relative reference audio file; may be repeated")
	outputName := fs.String("outputName", "", "requested WAV output name")
	fs.Var(&scenarios, "scenario", "generation scenario; may be repeated")
	fs.Var(&controls, "control", "optional VoxCPM expression style instruction; may be repeated")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 1
	}
	if fs.NArg() != 0 {
		fmt.Fprintf(stderr, "unexpected arguments: %s\n", strings.Join(fs.Args(), " "))
		return 1
	}
	request := voxcpmTaskCreateRequest{AgentID: strings.TrimSpace(*agentID), OutputName: strings.TrimSpace(*outputName)}
	if request.AgentID == "" || request.OutputName == "" || len(textPaths) == 0 {
		fmt.Fprintln(stderr, "--agentId, at least one --textPath, and --outputName are required; --referenceAudioPath is optional")
		return 1
	}
	if len(textPaths) > 5 {
		fmt.Fprintln(stderr, "at most five --textPath values are allowed")
		return 1
	}
	for index, textPath := range textPaths {
		referencePath, err := integrationAPIRepeatedTaskValue(referencePaths, index, len(textPaths), "", "--referenceAudioPath")
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		scenario, err := integrationAPIRepeatedTaskValue(scenarios, index, len(textPaths), voxcpmScenarioBalanced, "--scenario")
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		control, err := integrationAPIRepeatedTaskValue(controls, index, len(textPaths), "", "--control")
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		request.Tasks = append(request.Tasks, voxcpmTaskCreateItem{TextPath: strings.TrimSpace(textPath), ReferenceAudioPath: referencePath, Scenario: scenario, Control: control})
	}
	body, err := json.Marshal(request)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	base, err := resolveIntegrationAPIBase(*addr, *port)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	resp, reqErr := integrationAPIRequest(ctx, newIntegrationAPIClient(60*time.Second), http.MethodPost, base, "/api/voxcpm/tasks", nil, bytes.NewReader(body), "application/json", nil)
	return integrationAPIHandleHTTPResult(stdout, stderr, resp, reqErr, *output, *pretty, false)
}

func runIntegrationAPIVoxCPMTaskActionCLI(command string, args []string, stdout, stderr io.Writer) int {
	fs, addr, port, output, pretty := newIntegrationAPICommonFlagSet("integration api voxcpm "+command, "integration api voxcpm "+command+" --agentId ID --id TASK_ID [--addr URL] [--port 8080]", "Call a VoxCPM task action.", stderr)
	agentID := fs.String("agentId", "", "agent id")
	fs.StringVar(agentID, "agent", "", "agent id")
	taskID := fs.Int64("id", 0, "VoxCPM task id")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 1
	}
	if fs.NArg() != 0 {
		fmt.Fprintf(stderr, "unexpected arguments: %s\n", strings.Join(fs.Args(), " "))
		return 1
	}
	if strings.TrimSpace(*agentID) == "" || *taskID <= 0 {
		fmt.Fprintln(stderr, "--agentId and a positive --id are required")
		return 1
	}
	body, err := json.Marshal(map[string]interface{}{"agentId": strings.TrimSpace(*agentID), "id": *taskID})
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	base, err := resolveIntegrationAPIBase(*addr, *port)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	resp, reqErr := integrationAPIRequest(ctx, newIntegrationAPIClient(60*time.Second), http.MethodPost, base, "/api/voxcpm/tasks/"+command, nil, bytes.NewReader(body), "application/json", nil)
	return integrationAPIHandleHTTPResult(stdout, stderr, resp, reqErr, *output, *pretty, false)
}

func runIntegrationAPIVoxCPMCLI(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "--help" || strings.TrimSpace(args[0]) == "-h" || strings.TrimSpace(args[0]) == "help" {
		printIntegrationAPIVoxCPMHelp(stdout)
		return 0
	}
	switch strings.TrimSpace(args[0]) {
	case "check":
		return runIntegrationAPIGenericRequestCLI(integrationAPIGenericRequestSpec{Command: "voxcpm check", Usage: "integration api voxcpm check [--addr URL] [--port 8080]", Description: "Call GET /api/voxcpm/check.", Method: http.MethodGet, Path: "/api/voxcpm/check"}, args[1:], stdout, stderr)
	case "list":
		return runIntegrationAPIGenericRequestCLI(integrationAPIGenericRequestSpec{Command: "voxcpm list", Usage: "integration api voxcpm list --agentId ID [--status all|queued|running|completed|cancelled|failed] [--page N] [--addr URL] [--port 8080]", Description: "Call GET /api/voxcpm/tasks.", Method: http.MethodGet, Path: "/api/voxcpm/tasks", QueryFlags: []integrationAPIQueryFlag{{QueryKey: "agentId", Names: []string{"agentId", "agent"}, Usage: "agent id"}, {QueryKey: "status", Names: []string{"status"}, Usage: "task status"}, {QueryKey: "page", Names: []string{"page"}, Usage: "one-based page number"}}}, args[1:], stdout, stderr)
	case "create":
		return runIntegrationAPIVoxCPMCreateCLI(args[1:], stdout, stderr)
	case "cancel", "restart", "delete":
		return runIntegrationAPIVoxCPMTaskActionCLI(strings.TrimSpace(args[0]), args[1:], stdout, stderr)
	case "log":
		return runIntegrationAPIGenericRequestCLI(integrationAPIGenericRequestSpec{Command: "voxcpm log", Usage: "integration api voxcpm log --agentId ID --id TASK_ID [--addr URL] [--port 8080]", Description: "Call GET /api/voxcpm/tasks/log.", Method: http.MethodGet, Path: "/api/voxcpm/tasks/log", QueryFlags: []integrationAPIQueryFlag{{QueryKey: "agentId", Names: []string{"agentId", "agent"}, Usage: "agent id"}, {QueryKey: "id", Names: []string{"id"}, Usage: "task id"}}}, args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown voxcpm api command: %s\n", args[0])
		printIntegrationAPIVoxCPMHelp(stderr)
		return 1
	}
}

func printIntegrationAPIAgentHelp() {
	fmt.Println("Usage:")
	fmt.Println("  integration api agent export --agent ID [--output PATH] [--agent-dir DIR]")
	fmt.Println("  integration api agent import --input PATH [--agent-dir DIR]")
	fmt.Println("  integration api agent copy --source ID --target ID [--agent-dir DIR]")
	fmt.Println("  integration api agent init --name ID [--addr URL]")
	fmt.Println("  integration api agent delete --name ID [--addr URL]")
	fmt.Println("  integration api agent create --agentId ID --name PATH --type 0|1 [--addr URL]")
	fmt.Println("")
	fmt.Println("Notes:")
	fmt.Println("  export/import/copy 直接复用现有本地 CLI 实现。")
	fmt.Println("  init/delete/create 通过运行中的 integration HTTP 服务调用对应 API。")
}

func runIntegrationAPIAgentCLI(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "--help" || strings.TrimSpace(args[0]) == "-h" || strings.TrimSpace(args[0]) == "help" {
		printIntegrationAPIAgentHelp()
		return 0
	}
	switch strings.TrimSpace(args[0]) {
	case "export", "import", "copy":
		return runIntegrationAgentCLI(args, stdout, stderr)
	case "init":
		return runIntegrationAPIGenericRequestCLI(integrationAPIGenericRequestSpec{
			Command:     "agent init",
			Usage:       "integration api agent init --name ID [--addr URL] [--port 8080]",
			Description: "Call GET /api/agent/init to create a new agent directory from the default template.",
			Method:      http.MethodGet,
			Path:        "/api/agent/init",
			QueryFlags: []integrationAPIQueryFlag{
				{QueryKey: "name", Names: []string{"name"}, Usage: "new agent id"},
			},
		}, args[1:], stdout, stderr)
	case "delete":
		return runIntegrationAPIGenericRequestCLI(integrationAPIGenericRequestSpec{
			Command:     "agent delete",
			Usage:       "integration api agent delete --name ID [--addr URL] [--port 8080]",
			Description: "Call GET /api/agent/delete to delete one agent directory.",
			Method:      http.MethodGet,
			Path:        "/api/agent/delete",
			QueryFlags: []integrationAPIQueryFlag{
				{QueryKey: "name", Names: []string{"name"}, Usage: "agent id to delete"},
			},
		}, args[1:], stdout, stderr)
	case "create":
		return runIntegrationAPIGenericRequestCLI(integrationAPIGenericRequestSpec{
			Command:     "agent create",
			Usage:       "integration api agent create --agentId ID --name PATH --type 0|1 [--addr URL] [--port 8080]",
			Description: "Call GET /api/agent/create to create one file or directory under an agent workspace.",
			Method:      http.MethodGet,
			Path:        "/api/agent/create",
			QueryFlags: []integrationAPIQueryFlag{
				{QueryKey: "agentId", Names: []string{"agentId", "agent"}, Usage: "target agent id"},
				{QueryKey: "name", Names: []string{"name"}, Usage: "relative path under workspace"},
				{QueryKey: "type", Names: []string{"type"}, Usage: "0=directory, 1=file"},
			},
		}, args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown api agent command: %s\n", args[0])
		printIntegrationAPIAgentHelp()
		return 1
	}
}

func printIntegrationAPIConnectHelp() {
	fmt.Println("Usage:")
	fmt.Println("  integration api connect <subcommand> [options]")
	fmt.Println("")
	fmt.Println("Notes:")
	fmt.Println("  直接复用现有 integration connect CLI，并对齐到 /api/connect/meta、/api/connect/request、/api/connect/response。")
	fmt.Println("  可用子命令仍为 meta-create/meta-update/meta-delete/meta-get/meta-list/add-request/request-list/add-response/response-list。")
}

func runIntegrationAPIConnectCLI(args []string) int {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "--help" || strings.TrimSpace(args[0]) == "-h" || strings.TrimSpace(args[0]) == "help" {
		printIntegrationAPIConnectHelp()
		return 0
	}
	runIntegrationConnectCLI(args)
	return 0
}

func printIntegrationAPICronHelp() {
	fmt.Println("Usage:")
	fmt.Println("  integration api cron create ...")
	fmt.Println("  integration api cron create-cron ...")
	fmt.Println("  integration api cron detail-metadata ...")
	fmt.Println("  integration api cron detail-list ...")
	fmt.Println("  integration api cron delete ...")
	fmt.Println("  integration api cron detail-delete ...")
	fmt.Println("  integration api cron detail-status --agentId ID --detailId ID --status N [--addr URL]")
	fmt.Println("")
	fmt.Println("Notes:")
	fmt.Println("  create/create-cron/detail-metadata/detail-list/delete/detail-delete 复用现有 cron CLI。")
	fmt.Println("  detail-status 通过运行中的 integration HTTP 服务调用 /api/cron/detail/status。")
}

func runIntegrationAPICronCLI(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "--help" || strings.TrimSpace(args[0]) == "-h" || strings.TrimSpace(args[0]) == "help" {
		printIntegrationAPICronHelp()
		return 0
	}
	switch strings.TrimSpace(args[0]) {
	case "create", "create-cron":
		runIntegrationCronCLI(args)
		return 0
	case "detail-metadata":
		runIntegrationCronCLI(append([]string{"find-meta"}, args[1:]...))
		return 0
	case "detail-list":
		runIntegrationCronCLI(append([]string{"find-detail"}, args[1:]...))
		return 0
	case "delete":
		runIntegrationCronCLI(append([]string{"delete-meta"}, args[1:]...))
		return 0
	case "detail-delete":
		runIntegrationCronCLI(append([]string{"delete-detail"}, args[1:]...))
		return 0
	case "detail-status":
		return runIntegrationAPIGenericRequestCLI(integrationAPIGenericRequestSpec{
			Command:     "cron detail-status",
			Usage:       "integration api cron detail-status --agentId ID --detailId ID --status N [--addr URL] [--port 8080]",
			Description: "Call POST /api/cron/detail/status to update one cron detail status.",
			Method:      http.MethodPost,
			Path:        "/api/cron/detail/status",
			QueryFlags: []integrationAPIQueryFlag{
				{QueryKey: "agentId", Names: []string{"agentId", "agent"}, Usage: "agent id"},
				{QueryKey: "detailId", Names: []string{"detailId"}, Usage: "detail id"},
				{QueryKey: "status", Names: []string{"status"}, Usage: "new detail status"},
			},
		}, args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown api cron command: %s\n", args[0])
		printIntegrationAPICronHelp()
		return 1
	}
}

func printIntegrationAPIPluginsHelp() {
	fmt.Println("Usage:")
	fmt.Println("  integration api plugins meta [--addr URL]")
	fmt.Println("  integration api plugins status --key KEY [plugin flags...]")
	fmt.Println("  integration api plugins config --body-file PATH [--addr URL]")
	fmt.Println("  integration api plugins start --key KEY [plugin flags...]")
	fmt.Println("  integration api plugins stop --key KEY [plugin flags...]")
	fmt.Println("  integration api plugins exec --key KEY --command 'SUBCOMMAND [ARGS...]' [plugin flags...]")
	fmt.Println("  integration api plugins log --key KEY [--last N] [--output PATH]")
	fmt.Println("")
	fmt.Println("Notes:")
	fmt.Println("  start/stop/exec/status 直接复用现有 plugins CLI。")
	fmt.Println("  meta/config/log 通过运行中的 integration HTTP 服务调用对应 API。")
}

func runIntegrationAPIPluginsCLI(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "--help" || strings.TrimSpace(args[0]) == "-h" || strings.TrimSpace(args[0]) == "help" {
		printIntegrationAPIPluginsHelp()
		return 0
	}
	switch strings.TrimSpace(args[0]) {
	case "start", "stop", "exec", "status":
		runIntegrationPluginCLI(args)
		return 0
	case "meta":
		return runIntegrationAPIGenericRequestCLI(integrationAPIGenericRequestSpec{
			Command:     "plugins meta",
			Usage:       "integration api plugins meta [--addr URL] [--port 8080]",
			Description: "Call GET /api/plugins/meta.",
			Method:      http.MethodGet,
			Path:        "/api/plugins/meta",
		}, args[1:], stdout, stderr)
	case "config":
		return runIntegrationAPIRawBodyCLI(
			"plugins config",
			"integration api plugins config --body-file PATH [--addr URL] [--port 8080]",
			"Call POST /api/plugins/config. Use --body/--body-file to provide the same JSON payload as the HTTP API.",
			http.MethodPost,
			"/api/plugins/config",
			nil,
			args[1:],
			stdout,
			stderr,
		)
	case "log":
		return runIntegrationAPIGenericRequestCLI(integrationAPIGenericRequestSpec{
			Command:     "plugins log",
			Usage:       "integration api plugins log --key KEY [--last N] [--output PATH] [--addr URL] [--port 8080]",
			Description: "Call GET /api/plugins/log and stream plugin log output.",
			Method:      http.MethodGet,
			Path:        "/api/plugins/log",
			QueryFlags: []integrationAPIQueryFlag{
				{QueryKey: "key", Names: []string{"key"}, Usage: "plugin key"},
				{QueryKey: "last", Names: []string{"last"}, Usage: "number of last lines before streaming"},
			},
			AllowOutput: true,
			RawOutput:   true,
		}, args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown api plugins command: %s\n", args[0])
		printIntegrationAPIPluginsHelp()
		return 1
	}
}

func printIntegrationAPITokenHelp() {
	fmt.Println("Usage:")
	fmt.Println("  integration api token get [--addr URL]")
	fmt.Println("  integration api token set --body-file PATH [--addr URL]")
	fmt.Println("")
	fmt.Println("Notes:")
	fmt.Println("  get 对应 GET /api/token。")
	fmt.Println("  set 对应 POST /api/token；请求体需要与 HTTP API 一致。")
}

func runIntegrationAPITokenCLI(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "--help" || strings.TrimSpace(args[0]) == "-h" || strings.TrimSpace(args[0]) == "help" {
		printIntegrationAPITokenHelp()
		return 0
	}
	switch strings.TrimSpace(args[0]) {
	case "get":
		return runIntegrationAPIGenericRequestCLI(integrationAPIGenericRequestSpec{
			Command:     "token get",
			Usage:       "integration api token get [--addr URL] [--port 8080]",
			Description: "Call GET /api/token to read the token/model store.",
			Method:      http.MethodGet,
			Path:        "/api/token",
		}, args[1:], stdout, stderr)
	case "set":
		return runIntegrationAPIRawBodyCLI(
			"token set",
			"integration api token set --body-file PATH [--addr URL] [--port 8080]",
			"Call POST /api/token. Use --body/--body-file with the same JSON payload as the HTTP API.",
			http.MethodPost,
			"/api/token",
			nil,
			args[1:],
			stdout,
			stderr,
		)
	default:
		fmt.Fprintf(stderr, "unknown api token command: %s\n", args[0])
		printIntegrationAPITokenHelp()
		return 1
	}
}

func runIntegrationAPIChatCompletionsCLI(args []string, stdout, stderr io.Writer) int {
	return runIntegrationAPIRawBodyCLI(
		"chat-completions",
		"integration api chat-completions --body-file PATH [--addr URL] [--port 8080]",
		"Call POST /v1/chat/completions. Use --body/--body-file with the full JSON request body.",
		http.MethodPost,
		"/v1/chat/completions",
		nil,
		args,
		stdout,
		stderr,
	)
}

func runIntegrationAPISkillStateCLI(args []string, stdout, stderr io.Writer) int {
	fs, addr, port, output, pretty := newIntegrationAPICommonFlagSet(
		"integration api skill-state",
		"integration api skill-state --chatId ID --path /abs/skill/dir --disabled true|false [--addr URL] [--port 8080]",
		"Call POST /api/skill_state to enable or disable one skill directory for a chat session.",
		stderr,
	)
	chatID := fs.String("chatId", "", "chat id")
	chat := fs.String("chat", "", "chat id")
	path := fs.String("path", "", "absolute skill directory path")
	disabled := fs.String("disabled", "", "true or false")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 1
	}
	if fs.NArg() != 0 {
		fmt.Fprintf(stderr, "unexpected arguments: %s\n", strings.Join(fs.Args(), " "))
		return 1
	}
	targetChatID := firstNonEmpty(strings.TrimSpace(*chatID), strings.TrimSpace(*chat))
	if targetChatID == "" || strings.TrimSpace(*path) == "" || strings.TrimSpace(*disabled) == "" {
		fmt.Fprintln(stderr, "chatId, path and disabled are required")
		return 1
	}
	value, ok := sharedutil.ToBoolValue(*disabled)
	if !ok {
		fmt.Fprintln(stderr, "disabled must be true or false")
		return 1
	}
	body, _ := json.Marshal(map[string]any{
		"chatId":   targetChatID,
		"path":     strings.TrimSpace(*path),
		"disabled": value,
	})
	base, err := resolveIntegrationAPIBase(*addr, *port)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	resp, reqErr := integrationAPIRequest(ctx, newIntegrationAPIClient(60*time.Second), http.MethodPost, base, "/api/skill_state", nil, bytes.NewReader(body), "application/json", nil)
	return integrationAPIHandleHTTPResult(stdout, stderr, resp, reqErr, *output, *pretty, false)
}

func runIntegrationAPIEditCLI(args []string, stdout, stderr io.Writer) int {
	fs, addr, port, output, pretty := newIntegrationAPICommonFlagSet(
		"integration api edit",
		"integration api edit --agentId ID --path REL_PATH (--content TEXT | --content-file PATH | --base64 TEXT) [--save-as-new] [--addr URL] [--port 8080]",
		"Call POST /api/edit to write one text or binary file under an agent workspace.",
		stderr,
	)
	agentID := fs.String("agentId", "", "agent id")
	agent := fs.String("agent", "", "agent id")
	relPath := fs.String("path", "", "relative workspace path")
	content := fs.String("content", "", "text content")
	contentFile := fs.String("content-file", "", "read text content from file")
	contentBase64 := fs.String("base64", "", "base64 content for binary files")
	saveAsNew := fs.Bool("save-as-new", false, "append timestamp to target filename")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 1
	}
	if fs.NArg() != 0 {
		fmt.Fprintf(stderr, "unexpected arguments: %s\n", strings.Join(fs.Args(), " "))
		return 1
	}
	targetAgentID := firstNonEmpty(strings.TrimSpace(*agentID), strings.TrimSpace(*agent))
	if targetAgentID == "" || strings.TrimSpace(*relPath) == "" {
		fmt.Fprintln(stderr, "agentId and path are required")
		return 1
	}
	contentSources := 0
	if strings.TrimSpace(*content) != "" {
		contentSources++
	}
	if strings.TrimSpace(*contentFile) != "" {
		contentSources++
	}
	if strings.TrimSpace(*contentBase64) != "" {
		contentSources++
	}
	if contentSources != 1 {
		fmt.Fprintln(stderr, "exactly one of --content, --content-file or --base64 is required")
		return 1
	}
	contentValue := strings.TrimSpace(*content)
	if strings.TrimSpace(*contentFile) != "" {
		data, err := os.ReadFile(strings.TrimSpace(*contentFile))
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		contentValue = string(data)
	}
	if strings.TrimSpace(*contentBase64) != "" {
		contentValue = strings.TrimSpace(*contentBase64)
	}
	body, _ := json.Marshal(map[string]any{"content": contentValue})
	query := url.Values{
		"agentId": []string{targetAgentID},
		"path":    []string{strings.TrimSpace(*relPath)},
	}
	if *saveAsNew {
		query.Set("saveAsNew", "true")
	}
	base, err := resolveIntegrationAPIBase(*addr, *port)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	resp, reqErr := integrationAPIRequest(ctx, newIntegrationAPIClient(60*time.Second), http.MethodPost, base, "/api/edit", query, bytes.NewReader(body), "application/json", nil)
	return integrationAPIHandleHTTPResult(stdout, stderr, resp, reqErr, *output, *pretty, false)
}

func runIntegrationAPICmdCLI(args []string, stdout, stderr io.Writer) int {
	fs, addr, port, output, pretty := newIntegrationAPICommonFlagSet(
		"integration api cmd",
		"integration api cmd --agentId ID --chatId ID --cmd 'COMMAND' [--timeout MS] [--tid ID] [--addr URL] [--port 8080]",
		"Call POST /api/cmd to execute one command for an agent/chat session.",
		stderr,
	)
	agentID := fs.String("agentId", "", "agent id")
	agent := fs.String("agent", "", "agent id")
	chatID := fs.String("chatId", "", "chat id")
	chat := fs.String("chat", "", "chat id")
	command := fs.String("cmd", "", "shell command")
	timeout := fs.Int("timeout", 0, "timeout in milliseconds")
	tid := fs.String("tid", "", "optional task id")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 1
	}
	if fs.NArg() != 0 {
		fmt.Fprintf(stderr, "unexpected arguments: %s\n", strings.Join(fs.Args(), " "))
		return 1
	}
	targetAgentID := firstNonEmpty(strings.TrimSpace(*agentID), strings.TrimSpace(*agent))
	targetChatID := firstNonEmpty(strings.TrimSpace(*chatID), strings.TrimSpace(*chat))
	if targetAgentID == "" || targetChatID == "" || strings.TrimSpace(*command) == "" {
		fmt.Fprintln(stderr, "agentId, chatId and cmd are required")
		return 1
	}
	payload := map[string]any{
		"agentId": targetAgentID,
		"chatId":  targetChatID,
		"cmd":     strings.TrimSpace(*command),
	}
	if *timeout > 0 {
		payload["timeout"] = *timeout
	}
	if strings.TrimSpace(*tid) != "" {
		payload["tid"] = strings.TrimSpace(*tid)
	}
	body, _ := json.Marshal(payload)
	base, err := resolveIntegrationAPIBase(*addr, *port)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	resp, reqErr := integrationAPIRequest(ctx, newIntegrationAPIClient(60*time.Second), http.MethodPost, base, "/api/cmd", nil, bytes.NewReader(body), "application/json", nil)
	return integrationAPIHandleHTTPResult(stdout, stderr, resp, reqErr, *output, *pretty, false)
}

func runIntegrationAPIKillCLI(args []string, stdout, stderr io.Writer) int {
	fs, addr, port, output, pretty := newIntegrationAPICommonFlagSet(
		"integration api kill",
		"integration api kill --agentId ID --chatId ID --cmd 'COMMAND' [--tid ID] [--addr URL] [--port 8080]",
		"Call POST /api/kill to stop one active /api/cmd process.",
		stderr,
	)
	agentID := fs.String("agentId", "", "agent id")
	agent := fs.String("agent", "", "agent id")
	chatID := fs.String("chatId", "", "chat id")
	chat := fs.String("chat", "", "chat id")
	command := fs.String("cmd", "", "shell command")
	tid := fs.String("tid", "", "optional task id")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 1
	}
	if fs.NArg() != 0 {
		fmt.Fprintf(stderr, "unexpected arguments: %s\n", strings.Join(fs.Args(), " "))
		return 1
	}
	targetAgentID := firstNonEmpty(strings.TrimSpace(*agentID), strings.TrimSpace(*agent))
	targetChatID := firstNonEmpty(strings.TrimSpace(*chatID), strings.TrimSpace(*chat))
	if targetAgentID == "" || targetChatID == "" || strings.TrimSpace(*command) == "" {
		fmt.Fprintln(stderr, "agentId, chatId and cmd are required")
		return 1
	}
	payload := map[string]any{
		"agentId": targetAgentID,
		"chatId":  targetChatID,
		"cmd":     strings.TrimSpace(*command),
	}
	if strings.TrimSpace(*tid) != "" {
		payload["tid"] = strings.TrimSpace(*tid)
	}
	body, _ := json.Marshal(payload)
	base, err := resolveIntegrationAPIBase(*addr, *port)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	resp, reqErr := integrationAPIRequest(ctx, newIntegrationAPIClient(30*time.Second), http.MethodPost, base, "/api/kill", nil, bytes.NewReader(body), "application/json", nil)
	return integrationAPIHandleHTTPResult(stdout, stderr, resp, reqErr, *output, *pretty, false)
}

func runIntegrationAPIUploadCLI(args []string, stdout, stderr io.Writer) int {
	fs, addr, port, output, pretty := newIntegrationAPICommonFlagSet(
		"integration api upload",
		"integration api upload --agentId ID --file PATH [--file PATH ...] [--path REL ...] [--addr URL] [--port 8080]",
		"Call POST /api/upload to upload one or more files into an agent workspace tmp directory.",
		stderr,
	)
	agentID := fs.String("agentId", "", "agent id")
	agent := fs.String("agent", "", "agent id")
	var files integrationStringSliceFlag
	var relPaths integrationStringSliceFlag
	fs.Var(&files, "file", "file path to upload; repeatable")
	fs.Var(&relPaths, "path", "optional relative path for the corresponding uploaded file; repeatable")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 1
	}
	if fs.NArg() != 0 {
		fmt.Fprintf(stderr, "unexpected arguments: %s\n", strings.Join(fs.Args(), " "))
		return 1
	}
	targetAgentID := firstNonEmpty(strings.TrimSpace(*agentID), strings.TrimSpace(*agent))
	if targetAgentID == "" {
		fmt.Fprintln(stderr, "agentId is required")
		return 1
	}
	if len(files) == 0 {
		fmt.Fprintln(stderr, "at least one --file is required")
		return 1
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for _, filePath := range files {
		filePath = strings.TrimSpace(filePath)
		if filePath == "" {
			continue
		}
		src, err := os.Open(filePath)
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		part, err := writer.CreateFormFile("files", filepath.Base(filePath))
		if err != nil {
			src.Close()
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		if _, err := io.Copy(part, src); err != nil {
			src.Close()
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		src.Close()
	}
	if len(relPaths) > 0 {
		pathJSON, _ := json.Marshal([]string(relPaths))
		_ = writer.WriteField("pathsJson", string(pathJSON))
	}
	if err := writer.Close(); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	base, err := resolveIntegrationAPIBase(*addr, *port)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return 1
	}
	query := url.Values{"agentId": []string{targetAgentID}}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	resp, reqErr := integrationAPIRequest(ctx, newIntegrationAPIClient(120*time.Second), http.MethodPost, base, "/api/upload", query, bytes.NewReader(body.Bytes()), writer.FormDataContentType(), nil)
	return integrationAPIHandleHTTPResult(stdout, stderr, resp, reqErr, *output, *pretty, false)
}

func runIntegrationAPIDownloadCLI(args []string, stdout, stderr io.Writer) int {
	return runIntegrationAPIGenericRequestCLI(integrationAPIGenericRequestSpec{
		Command:       "download",
		Usage:         "integration api download --path PATH --output FILE [--addr URL] [--port 8080]",
		Description:   "Call GET /api/download to download one file or directory. --output is required.",
		Method:        http.MethodGet,
		Path:          "/api/download",
		RequireOutput: true,
		RawOutput:     true,
		QueryFlags: []integrationAPIQueryFlag{
			{QueryKey: "path", Names: []string{"path"}, Usage: "absolute or relative target path"},
		},
	}, args, stdout, stderr)
}

func runIntegrationAPIKnowledgeCLI(args []string, stdout, stderr io.Writer) int {
	return runIntegrationAPIGenericRequestCLI(integrationAPIGenericRequestSpec{
		Command:     "knowledge",
		Usage:       "integration api knowledge [--path REL_PATH] [--output FILE] [--addr URL] [--port 8080]",
		Description: "Call GET /knowledge or GET /knowledge/<path> to browse the knowledge tree or read one file.",
		Method:      http.MethodGet,
		Path:        "/knowledge",
		PathFlag:    "path",
		PathUsage:   "relative path under the knowledge root",
		AllowOutput: true,
		RawOutput:   true,
	}, args, stdout, stderr)
}

func integrationAPICommonSimpleSpecs() map[string]integrationAPIGenericRequestSpec {
	specs := []integrationAPIGenericRequestSpec{
		{
			Command:     "heartbeat",
			Usage:       "integration api heartbeat [--addr URL] [--port 8080]",
			Description: "Call GET /api/heartbeat.",
			Method:      http.MethodGet,
			Path:        "/api/heartbeat",
		},
		{
			Command:     "log-cleanup-status",
			Usage:       "integration api log-cleanup-status [--addr URL] [--port 8080]",
			Description: "Call GET /api/log_cleanup_status.",
			Method:      http.MethodGet,
			Path:        "/api/log_cleanup_status",
		},
		{
			Command:     "agent-id",
			Usage:       "integration api agent-id [--addr URL] [--port 8080]",
			Description: "Call GET /api/agentId.",
			Method:      http.MethodGet,
			Path:        "/api/agentId",
		},
		{
			Command:     "swarm-agent",
			Usage:       "integration api swarm-agent [--agentId ID] [--addr URL] [--port 8080]",
			Description: "Call GET /api/swarm_agent.",
			Method:      http.MethodGet,
			Path:        "/api/swarm_agent",
			QueryFlags: []integrationAPIQueryFlag{
				{QueryKey: "agentId", Names: []string{"agentId", "agent"}, Usage: "optional agent id to exclude from results"},
			},
		},
		{
			Command:     "device-id",
			Usage:       "integration api device-id [--addr URL] [--port 8080]",
			Description: "Call GET /api/deviceId.",
			Method:      http.MethodGet,
			Path:        "/api/deviceId",
		},
		{
			Command:     "folder",
			Usage:       "integration api folder [--agentId ID | --path ABS_PATH] [--addr URL] [--port 8080]",
			Description: "Call GET /api/folder to open one agent workspace or absolute path.",
			Method:      http.MethodGet,
			Path:        "/api/folder",
			QueryFlags: []integrationAPIQueryFlag{
				{QueryKey: "agentId", Names: []string{"agentId", "agent"}, Usage: "agent id"},
				{QueryKey: "path", Names: []string{"path"}, Usage: "absolute path"},
			},
		},
		{
			Command:     "skills",
			Usage:       "integration api skills --agentId ID [--chatId ID] [--addr URL] [--port 8080]",
			Description: "Call GET /api/skills.",
			Method:      http.MethodGet,
			Path:        "/api/skills",
			QueryFlags: []integrationAPIQueryFlag{
				{QueryKey: "agentId", Names: []string{"agentId", "agent"}, Usage: "agent id"},
				{QueryKey: "chatId", Names: []string{"chatId", "chat"}, Usage: "optional chat id"},
			},
		},
		{
			Command:     "files",
			Usage:       "integration api files --path PATH [--addr URL] [--port 8080]",
			Description: "Call GET /api/files.",
			Method:      http.MethodGet,
			Path:        "/api/files",
			QueryFlags: []integrationAPIQueryFlag{
				{QueryKey: "path", Names: []string{"path"}, Usage: "absolute path or prefix path"},
			},
		},
		{
			Command:     "data",
			Usage:       "integration api data --path PATH [--addr URL] [--port 8080]",
			Description: "Call GET /api/data.",
			Method:      http.MethodGet,
			Path:        "/api/data",
			QueryFlags: []integrationAPIQueryFlag{
				{QueryKey: "path", Names: []string{"path"}, Usage: "file path"},
			},
		},
		{
			Command:     "workspace",
			Usage:       "integration api workspace --agentId ID [--addr URL] [--port 8080]",
			Description: "Call GET /api/workspace.",
			Method:      http.MethodGet,
			Path:        "/api/workspace",
			QueryFlags: []integrationAPIQueryFlag{
				{QueryKey: "agentId", Names: []string{"agentId", "agent"}, Usage: "agent id"},
			},
		},
		{
			Command:     "url-preview-probe",
			Usage:       "integration api url-preview-probe --url URL [--addr URL] [--port 8080]",
			Description: "Call GET /api/url_preview_probe.",
			Method:      http.MethodGet,
			Path:        "/api/url_preview_probe",
			QueryFlags: []integrationAPIQueryFlag{
				{QueryKey: "url", Names: []string{"url"}, Usage: "target URL"},
			},
		},
		{
			Command:     "del",
			Usage:       "integration api del --agentId ID --path REL_PATH [--addr URL] [--port 8080]",
			Description: "Call GET /api/del.",
			Method:      http.MethodGet,
			Path:        "/api/del",
			QueryFlags: []integrationAPIQueryFlag{
				{QueryKey: "agentId", Names: []string{"agentId", "agent"}, Usage: "agent id"},
				{QueryKey: "path", Names: []string{"path"}, Usage: "relative workspace path"},
			},
		},
		{
			Command:     "raw",
			Usage:       "integration api raw --agentId ID --path REL_PATH [--addr URL] [--port 8080]",
			Description: "Call GET /api/raw.",
			Method:      http.MethodGet,
			Path:        "/api/raw",
			QueryFlags: []integrationAPIQueryFlag{
				{QueryKey: "agentId", Names: []string{"agentId", "agent"}, Usage: "agent id"},
				{QueryKey: "path", Names: []string{"path"}, Usage: "relative workspace path"},
			},
		},
		{
			Command:     "file-last-update",
			Usage:       "integration api file-last-update --file PATH [--agentId ID] [--addr URL] [--port 8080]",
			Description: "Call GET /file/lastUpdate.",
			Method:      http.MethodGet,
			Path:        "/file/lastUpdate",
			QueryFlags: []integrationAPIQueryFlag{
				{QueryKey: "file", Names: []string{"file"}, Usage: "target file path"},
				{QueryKey: "agentId", Names: []string{"agentId", "agent"}, Usage: "agent id for relative paths"},
			},
		},
		{
			Command:     "site-access",
			Usage:       "integration api site-access [--addr URL] [--port 8080]",
			Description: "Call GET /api/site/access.",
			Method:      http.MethodGet,
			Path:        "/api/site/access",
		},
		{
			Command:     "consume",
			Usage:       "integration api consume --start TIME --close TIME [--agentId ID] [--limit N] [--addr URL] [--port 8080]",
			Description: "Call GET /api/consume. TIME uses yyyyMMdd-hhmmss.",
			Method:      http.MethodGet,
			Path:        "/api/consume",
			QueryFlags: []integrationAPIQueryFlag{
				{QueryKey: "starTime", Names: []string{"start"}, Usage: "query start time, format yyyyMMdd-hhmmss"},
				{QueryKey: "closeTime", Names: []string{"close"}, Usage: "query close time, format yyyyMMdd-hhmmss"},
				{QueryKey: "agentId", Names: []string{"agentId", "agent"}, Usage: "optional agent id"},
				{QueryKey: "limit", Names: []string{"limit", "n"}, Usage: "query limit"},
			},
		},
		{
			Command:     "cancel",
			Usage:       "integration api cancel --chat ID [--addr URL] [--port 8080]",
			Description: "Call POST /api/cancel.",
			Method:      http.MethodPost,
			Path:        "/api/cancel",
			QueryFlags: []integrationAPIQueryFlag{
				{QueryKey: "chat", Names: []string{"chat", "chatId"}, Usage: "chat id"},
			},
		},
		{
			Command:     "restore",
			Usage:       "integration api restore --chat ID (--timeline RFC3339 [--lastId N] | --history true [--beforeTimeline RFC3339] [--beforeId N] [--limit N]) [--agentId ID] [--addr URL] [--port 8080]",
			Description: "Call POST /api/restore for forward replay or backward history pagination.",
			Method:      http.MethodPost,
			Path:        "/api/restore",
			QueryFlags: []integrationAPIQueryFlag{
				{QueryKey: "agentId", Names: []string{"agentId", "agent"}, Usage: "optional agent id"},
				{QueryKey: "chat", Names: []string{"chat", "chatId"}, Usage: "chat id"},
				{QueryKey: "timeline", Names: []string{"timeline"}, Usage: "lower time bound"},
				{QueryKey: "lastId", Names: []string{"lastId"}, Usage: "optional last seen record id"},
				{QueryKey: "history", Names: []string{"history"}, Usage: "set true/1 to fetch paged history"},
				{QueryKey: "beforeTimeline", Names: []string{"beforeTimeline"}, Usage: "history cursor timeline"},
				{QueryKey: "beforeId", Names: []string{"beforeId"}, Usage: "history cursor record id"},
				{QueryKey: "limit", Names: []string{"limit"}, Usage: "history page size"},
			},
		},
		{
			Command:     "chat-session-log",
			Usage:       "integration api chat-session-log --chatId ID [--agentId ID] [--limit N] [--start TIME] [--close TIME] [--addr URL] [--port 8080]",
			Description: "Call GET /api/chat_session_log.",
			Method:      http.MethodGet,
			Path:        "/api/chat_session_log",
			QueryFlags: []integrationAPIQueryFlag{
				{QueryKey: "agentId", Names: []string{"agentId", "agent"}, Usage: "optional agent id"},
				{QueryKey: "chatId", Names: []string{"chatId", "chat"}, Usage: "chat id"},
				{QueryKey: "limit", Names: []string{"limit"}, Usage: "result limit"},
				{QueryKey: "start", Names: []string{"start"}, Usage: "optional start time"},
				{QueryKey: "close", Names: []string{"close"}, Usage: "optional close time"},
			},
		},
		{
			Command:       "download",
			Usage:         "integration api download --path PATH --output FILE [--addr URL] [--port 8080]",
			Description:   "Call GET /api/download.",
			Method:        http.MethodGet,
			Path:          "/api/download",
			RequireOutput: true,
			RawOutput:     true,
			QueryFlags: []integrationAPIQueryFlag{
				{QueryKey: "path", Names: []string{"path"}, Usage: "absolute or relative path"},
			},
		},
		{
			Command:     "skills-warning",
			Usage:       "integration api skills-warning [--addr URL] [--port 8080]",
			Description: "Call GET /skills_warning.",
			Method:      http.MethodGet,
			Path:        "/skills_warning",
		},
		{
			Command:     "install-app",
			Usage:       "integration api install-app [--addr URL] [--port 8080]",
			Description: "Call GET /install_app.",
			Method:      http.MethodGet,
			Path:        "/install_app",
		},
		{
			Command:     "log-skill",
			Usage:       "integration api log-skill [--chatId ID] [--round N] [--start TIME] [--close TIME] [--addr URL] [--port 8080]",
			Description: "Call GET /log_skill.",
			Method:      http.MethodGet,
			Path:        "/log_skill",
			QueryFlags: []integrationAPIQueryFlag{
				{QueryKey: "chatId", Names: []string{"chatId", "chat"}, Usage: "chat id"},
				{QueryKey: "round", Names: []string{"round"}, Usage: "round number"},
				{QueryKey: "start", Names: []string{"start"}, Usage: "optional start time"},
				{QueryKey: "close", Names: []string{"close"}, Usage: "optional close time"},
			},
		},
		{
			Command:     "log-skill-status",
			Usage:       "integration api log-skill-status [--agentId ID] [--chatId ID] [--round N] [--addr URL] [--port 8080]",
			Description: "Call GET /log_skill_status.",
			Method:      http.MethodGet,
			Path:        "/log_skill_status",
			QueryFlags: []integrationAPIQueryFlag{
				{QueryKey: "agentId", Names: []string{"agentId", "agent"}, Usage: "agent id"},
				{QueryKey: "chatId", Names: []string{"chatId", "chat"}, Usage: "chat id"},
				{QueryKey: "round", Names: []string{"round"}, Usage: "round number"},
			},
		},
		{
			Command:     "knowledge-last-update",
			Usage:       "integration api knowledge-last-update [--agentId ID] [--addr URL] [--port 8080]",
			Description: "Call GET /knowledge_lastUpdate.",
			Method:      http.MethodGet,
			Path:        "/knowledge_lastUpdate",
			QueryFlags: []integrationAPIQueryFlag{
				{QueryKey: "agentId", Names: []string{"agentId", "agent"}, Usage: "optional agent id"},
			},
			RawOutput: true,
		},
		{
			Command:     "knowledge-path",
			Usage:       "integration api knowledge-path [--agentId ID] [--addr URL] [--port 8080]",
			Description: "Call GET /knowledge_path.",
			Method:      http.MethodGet,
			Path:        "/knowledge_path",
			QueryFlags: []integrationAPIQueryFlag{
				{QueryKey: "agentId", Names: []string{"agentId", "agent"}, Usage: "optional agent id"},
			},
			RawOutput: true,
		},
	}
	out := make(map[string]integrationAPIGenericRequestSpec, len(specs))
	for _, spec := range specs {
		out[spec.Command] = spec
	}
	return out
}

func runIntegrationAPICLI(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || strings.TrimSpace(args[0]) == "--help" || strings.TrimSpace(args[0]) == "-h" || strings.TrimSpace(args[0]) == "help" {
		printIntegrationAPIHelp()
		return 0
	}
	command := strings.TrimSpace(args[0])
	switch command {
	case "host":
		return runIntegrationHostCLI(args[1:], stdout, stderr)
	case "standalone":
		return runIntegrationStandaloneCLI(args[1:], stdout, stderr)
	case "sandbox":
		runIntegrationSandboxCLI(args[1:])
		return 0
	case "connect":
		return runIntegrationAPIConnectCLI(args[1:])
	case "cron":
		return runIntegrationAPICronCLI(args[1:], stdout, stderr)
	case "agent":
		return runIntegrationAPIAgentCLI(args[1:], stdout, stderr)
	case "message-insert":
		runIntegrationMessageInsertCLI(args[1:])
		return 0
	case "plugins":
		return runIntegrationAPIPluginsCLI(args[1:], stdout, stderr)
	case "token":
		return runIntegrationAPITokenCLI(args[1:], stdout, stderr)
	case "chat-completions":
		return runIntegrationAPIChatCompletionsCLI(args[1:], stdout, stderr)
	case "skill-state":
		return runIntegrationAPISkillStateCLI(args[1:], stdout, stderr)
	case "edit":
		return runIntegrationAPIEditCLI(args[1:], stdout, stderr)
	case "cmd":
		return runIntegrationAPICmdCLI(args[1:], stdout, stderr)
	case "kill":
		return runIntegrationAPIKillCLI(args[1:], stdout, stderr)
	case "upload":
		return runIntegrationAPIUploadCLI(args[1:], stdout, stderr)
	case "config":
		return runIntegrationAPIRawBodyCLI(
			"config",
			"integration api config [--agentId ID] --body-file PATH [--addr URL] [--port 8080]",
			"Call POST /api/config. Use --agentId when updating one agent config.json; omit it for action payloads such as delete_model.",
			http.MethodPost,
			"/api/config",
			[]integrationAPIQueryFlag{
				{QueryKey: "agentId", Names: []string{"agentId", "agent"}, Usage: "target agent id"},
			},
			args[1:],
			stdout,
			stderr,
		)
	case "download":
		return runIntegrationAPIDownloadCLI(args[1:], stdout, stderr)
	case "knowledge":
		return runIntegrationAPIKnowledgeCLI(args[1:], stdout, stderr)
	case "whisper":
		return runIntegrationAPIWhisperCLI(args[1:], stdout, stderr)
	case "rembg":
		return runIntegrationAPIRembgCLI(args[1:], stdout, stderr)
	case "voxcpm":
		return runIntegrationAPIVoxCPMCLI(args[1:], stdout, stderr)
	case "log-round":
		cfg := &Config{AgentDir: integrationDefaultAgentDir(), AgentCacheMs: 120000}
		if value, ok := readIntegrationStartupConfigValue("agent-dir"); ok {
			cfg.AgentDir = firstNonEmpty(value, cfg.AgentDir)
		}
		if value, ok := readIntegrationStartupConfigValue("device"); ok {
			cfg.Device = value
		}
		if raw, ok := readIntegrationStartupConfigValue("agent-cache"); ok {
			if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
				cfg.AgentCacheMs = parsed
			}
		}
		runIntegrationLogRoundCLI(args[1:], cfg)
		return 0
	default:
		specs := integrationAPICommonSimpleSpecs()
		spec, ok := specs[command]
		if !ok {
			fmt.Fprintf(stderr, "unknown api command: %s\n", command)
			printIntegrationAPIHelp()
			return 1
		}
		return runIntegrationAPIGenericRequestCLI(spec, args[1:], stdout, stderr)
	}
}
