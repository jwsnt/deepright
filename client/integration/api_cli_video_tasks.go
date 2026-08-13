package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func integrationAPIMediaTaskQueryFlags() []integrationAPIQueryFlag {
	return []integrationAPIQueryFlag{
		{QueryKey: "agentId", Names: []string{"agentId", "agent"}, Usage: "agent id"},
		{QueryKey: "status", Names: []string{"status"}, Usage: "all, queued, running, completed, cancelled, or failed"},
		{QueryKey: "page", Names: []string{"page"}, Usage: "one-based page number; five tasks per page"},
	}
}

func integrationAPITaskScenarioValue(values []string, index, taskCount int, fallback, flagName string) (string, error) {
	switch len(values) {
	case 0:
		return fallback, nil
	case 1:
		return strings.TrimSpace(values[0]), nil
	case taskCount:
		return strings.TrimSpace(values[index]), nil
	default:
		return "", fmt.Errorf("%s 必须提供一次，或为每个 --path 提供一次", flagName)
	}
}

func runIntegrationAPIMediaTaskActionCLI(feature, command, path, description string, args []string, stdout, stderr io.Writer) int {
	fs, addr, port, output, pretty := newIntegrationAPICommonFlagSet(
		"integration api "+feature+" "+command,
		"integration api "+feature+" "+command+" --agentId ID --id TASK_ID [--addr URL] [--port 8080]",
		description,
		stderr,
	)
	agentID := fs.String("agentId", "", "Agent ID")
	fs.StringVar(agentID, "agent", "", "Agent ID")
	taskID := fs.Int64("id", 0, "task ID")
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
	resp, requestErr := integrationAPIRequest(ctx, newIntegrationAPIClient(60*time.Second), http.MethodPost, base, path, nil, bytes.NewReader(body), "application/json", nil)
	return integrationAPIHandleHTTPResult(stdout, stderr, resp, requestErr, *output, *pretty, false)
}

func printIntegrationAPIRVMHelp(stdout io.Writer) {
	fmt.Fprintln(stdout, "Usage:")
	fmt.Fprintln(stdout, "  integration api rvm <check|list|create|cancel|restart|delete|log> [options]")
	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "功能：视频提取人物。任务进入与文字转语音、音频转写、图片主体提取、人物视频对口型共用的模型等待队列；未取得执行名额时状态为 queued。")
	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "Commands:")
	fmt.Fprintln(stdout, "  check       检查受控环境能否运行 RVM")
	fmt.Fprintln(stdout, "  list        查询一个 Agent 的任务；每页固定 5 条")
	fmt.Fprintln(stdout, "  create      将一个或多个工作区视频加入队列")
	fmt.Fprintln(stdout, "  cancel      取消 queued 或 running 任务")
	fmt.Fprintln(stdout, "  restart     将 failed 或 cancelled 任务重新加入队列（旧日志会清空）")
	fmt.Fprintln(stdout, "  delete      删除 failed 或 cancelled 任务记录，不删除文件")
	fmt.Fprintln(stdout, "  log         读取任务与持久化执行日志")
	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "Create options:")
	fmt.Fprintln(stdout, "  --path PATH         工作区相对视频路径；可重复，最多 64 个")
	fmt.Fprintln(stdout, "  --scenario NAME     fast（速度优先，默认）或 quality（质量优先）；可提供一次或每个 --path 一次")
	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "Examples:")
	fmt.Fprintln(stdout, "  integration api rvm check")
	fmt.Fprintln(stdout, "  integration api rvm list --agentId demo-agent --status queued --page 1")
	fmt.Fprintln(stdout, "  integration api rvm create --agentId demo-agent --path videos/source.mp4 --scenario quality")
	fmt.Fprintln(stdout, "  integration api rvm create --agentId demo-agent --path videos/a.mp4 --path videos/b.mov --scenario fast --scenario quality")
	fmt.Fprintln(stdout, "  integration api rvm cancel --agentId demo-agent --id 12")
	fmt.Fprintln(stdout, "  integration api rvm restart --agentId demo-agent --id 12")
	fmt.Fprintln(stdout, "  integration api rvm log --agentId demo-agent --id 12")
}

func runIntegrationAPIRVMCreateCLI(args []string, stdout, stderr io.Writer) int {
	fs, addr, port, output, pretty := newIntegrationAPICommonFlagSet(
		"integration api rvm create",
		"integration api rvm create --agentId ID --path VIDEO_PATH [--path VIDEO_PATH ...] [--scenario fast|quality] [--addr URL] [--port 8080]",
		"Call POST /api/rvm/tasks. Each path must be a video under the selected Agent workspace. A scenario supplied once applies to every path; otherwise provide one scenario per path.",
		stderr,
	)
	agentID := fs.String("agentId", "", "Agent ID")
	fs.StringVar(agentID, "agent", "", "Agent ID")
	var paths, scenarios integrationStringSliceFlag
	fs.Var(&paths, "path", "workspace-relative video path; may be repeated")
	fs.Var(&scenarios, "scenario", "standard, quality, or fast; may be repeated")
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
	cleanPaths := make([]string, 0, len(paths))
	for _, path := range paths {
		if path = strings.TrimSpace(path); path != "" {
			cleanPaths = append(cleanPaths, path)
		}
	}
	if strings.TrimSpace(*agentID) == "" || len(cleanPaths) == 0 {
		fmt.Fprintln(stderr, "--agentId and at least one --path are required")
		return 1
	}
	if len(cleanPaths) > 64 {
		fmt.Fprintln(stderr, "at most 64 --path values are allowed")
		return 1
	}
	request := rvmTaskCreateRequest{AgentID: strings.TrimSpace(*agentID), Tasks: make([]rvmTaskCreateItem, 0, len(cleanPaths))}
	for index, path := range cleanPaths {
		scenario, err := integrationAPITaskScenarioValue(scenarios, index, len(cleanPaths), rvmScenarioFast, "--scenario")
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		request.Tasks = append(request.Tasks, rvmTaskCreateItem{Path: path, Scenario: scenario})
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
	resp, requestErr := integrationAPIRequest(ctx, newIntegrationAPIClient(60*time.Second), http.MethodPost, base, "/api/rvm/tasks", nil, bytes.NewReader(body), "application/json", nil)
	return integrationAPIHandleHTTPResult(stdout, stderr, resp, requestErr, *output, *pretty, false)
}

func runIntegrationAPIRVMCLI(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		printIntegrationAPIRVMHelp(stdout)
		return 0
	}
	switch strings.TrimSpace(args[0]) {
	case "check":
		return runIntegrationAPIGenericRequestCLI(integrationAPIGenericRequestSpec{Command: "rvm check", Usage: "integration api rvm check [--addr URL] [--port 8080]", Description: "Check whether the controlled environment can run RVM.", Method: http.MethodGet, Path: "/api/rvm/check"}, args[1:], stdout, stderr)
	case "list":
		return runIntegrationAPIGenericRequestCLI(integrationAPIGenericRequestSpec{Command: "rvm list", Usage: "integration api rvm list --agentId ID [--status all|queued|running|completed|cancelled|failed] [--page N] [--addr URL] [--port 8080]", Description: "List RVM tasks; the server returns five tasks per page.", Method: http.MethodGet, Path: "/api/rvm/tasks", QueryFlags: integrationAPIMediaTaskQueryFlags()}, args[1:], stdout, stderr)
	case "create":
		return runIntegrationAPIRVMCreateCLI(args[1:], stdout, stderr)
	case "cancel":
		return runIntegrationAPIMediaTaskActionCLI("rvm", "cancel", "/api/rvm/tasks/cancel", "Cancel one queued or running RVM task.", args[1:], stdout, stderr)
	case "restart":
		return runIntegrationAPIMediaTaskActionCLI("rvm", "restart", "/api/rvm/tasks/restart", "Restart one failed or cancelled RVM task; its previous log is cleared.", args[1:], stdout, stderr)
	case "delete":
		return runIntegrationAPIMediaTaskActionCLI("rvm", "delete", "/api/rvm/tasks/delete", "Delete one failed or cancelled RVM task record; files are preserved.", args[1:], stdout, stderr)
	case "log":
		return runIntegrationAPIGenericRequestCLI(integrationAPIGenericRequestSpec{Command: "rvm log", Usage: "integration api rvm log --agentId ID --id TASK_ID [--addr URL] [--port 8080]", Description: "Read one RVM task and its persisted execution log.", Method: http.MethodGet, Path: "/api/rvm/tasks/log", QueryFlags: []integrationAPIQueryFlag{{QueryKey: "agentId", Names: []string{"agentId", "agent"}, Usage: "agent id"}, {QueryKey: "id", Names: []string{"id"}, Usage: "RVM task id"}}}, args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown rvm api command: %s\n", args[0])
		printIntegrationAPIRVMHelp(stderr)
		return 1
	}
}

func printIntegrationAPIWav2LipHelp(stdout io.Writer) {
	fmt.Fprintln(stdout, "Usage:")
	fmt.Fprintln(stdout, "  integration api wav2lip <check|list|create|cancel|restart|delete|log> [options]")
	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "功能：人物视频对口型。视频与音频成对提交；任务与其它四类模型任务共用等待队列，等待时保持 queued。")
	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "Commands:")
	fmt.Fprintln(stdout, "  check       检查受控环境、脚本和 Wav2Lip 权重")
	fmt.Fprintln(stdout, "  list        查询一个 Agent 的任务；每页固定 5 条")
	fmt.Fprintln(stdout, "  create      将一至 64 组视频、音频配对加入队列")
	fmt.Fprintln(stdout, "  cancel      取消 queued 或 running 任务")
	fmt.Fprintln(stdout, "  restart     将 failed 或 cancelled 任务重新加入队列（旧日志会清空）")
	fmt.Fprintln(stdout, "  delete      删除 failed 或 cancelled 任务记录，不删除文件")
	fmt.Fprintln(stdout, "  log         读取任务、失败原因和持久化执行日志")
	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "Create options:")
	fmt.Fprintln(stdout, "  --videoPath PATH    工作区相对视频路径；可重复")
	fmt.Fprintln(stdout, "  --audioPath PATH    工作区相对音频路径；可重复，数量及顺序必须与 --videoPath 一致")
	fmt.Fprintln(stdout, "  --scenario NAME     quality（默认）、fast 或 motion；可提供一次或为每组视频音频提供一次")
	fmt.Fprintln(stdout, "")
	fmt.Fprintln(stdout, "Examples:")
	fmt.Fprintln(stdout, "  integration api wav2lip check")
	fmt.Fprintln(stdout, "  integration api wav2lip list --agentId demo-agent --status running --page 1")
	fmt.Fprintln(stdout, "  integration api wav2lip create --agentId demo-agent --videoPath videos/person.mp4 --audioPath audios/line.wav --scenario quality")
	fmt.Fprintln(stdout, "  integration api wav2lip create --agentId demo-agent --videoPath videos/a.mp4 --audioPath audios/a.wav --videoPath videos/b.mp4 --audioPath audios/b.mp3")
	fmt.Fprintln(stdout, "  integration api wav2lip cancel --agentId demo-agent --id 12")
	fmt.Fprintln(stdout, "  integration api wav2lip restart --agentId demo-agent --id 12")
	fmt.Fprintln(stdout, "  integration api wav2lip log --agentId demo-agent --id 12")
}

func runIntegrationAPIWav2LipCreateCLI(args []string, stdout, stderr io.Writer) int {
	fs, addr, port, output, pretty := newIntegrationAPICommonFlagSet(
		"integration api wav2lip create",
		"integration api wav2lip create --agentId ID --videoPath VIDEO_PATH --audioPath AUDIO_PATH [--videoPath VIDEO_PATH --audioPath AUDIO_PATH ...] [--scenario quality|fast|motion] [--addr URL] [--port 8080]",
		"Call POST /api/wav2lip/tasks. Video and audio values form pairs in their supplied order and must be files under the selected Agent workspace.",
		stderr,
	)
	agentID := fs.String("agentId", "", "Agent ID")
	fs.StringVar(agentID, "agent", "", "Agent ID")
	var videos, audios, scenarios integrationStringSliceFlag
	fs.Var(&videos, "videoPath", "workspace-relative video path; may be repeated")
	fs.Var(&videos, "video-path", "workspace-relative video path; may be repeated")
	fs.Var(&audios, "audioPath", "workspace-relative audio path; may be repeated")
	fs.Var(&audios, "audio-path", "workspace-relative audio path; may be repeated")
	fs.Var(&scenarios, "scenario", "quality, fast, or motion; may be repeated")
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
	if strings.TrimSpace(*agentID) == "" || len(videos) == 0 || len(videos) != len(audios) {
		fmt.Fprintln(stderr, "--agentId and equal non-empty --videoPath/--audioPath pairs are required")
		return 1
	}
	if len(videos) > 64 {
		fmt.Fprintln(stderr, "at most 64 video/audio pairs are allowed")
		return 1
	}
	request := wav2lipTaskCreateRequest{AgentID: strings.TrimSpace(*agentID), Tasks: make([]wav2lipTaskCreateItem, 0, len(videos))}
	for index := range videos {
		video, audio := strings.TrimSpace(videos[index]), strings.TrimSpace(audios[index])
		if video == "" || audio == "" {
			fmt.Fprintln(stderr, "--videoPath and --audioPath values cannot be empty")
			return 1
		}
		scenario, err := integrationAPITaskScenarioValue(scenarios, index, len(videos), wav2lipScenarioQuality, "--scenario")
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return 1
		}
		request.Tasks = append(request.Tasks, wav2lipTaskCreateItem{VideoPath: video, AudioPath: audio, Scenario: scenario})
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
	resp, requestErr := integrationAPIRequest(ctx, newIntegrationAPIClient(60*time.Second), http.MethodPost, base, "/api/wav2lip/tasks", nil, bytes.NewReader(body), "application/json", nil)
	return integrationAPIHandleHTTPResult(stdout, stderr, resp, requestErr, *output, *pretty, false)
}

func runIntegrationAPIWav2LipCLI(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		printIntegrationAPIWav2LipHelp(stdout)
		return 0
	}
	switch strings.TrimSpace(args[0]) {
	case "check":
		return runIntegrationAPIGenericRequestCLI(integrationAPIGenericRequestSpec{Command: "wav2lip check", Usage: "integration api wav2lip check [--addr URL] [--port 8080]", Description: "Check whether the controlled environment can run Wav2Lip.", Method: http.MethodGet, Path: "/api/wav2lip/check"}, args[1:], stdout, stderr)
	case "list":
		return runIntegrationAPIGenericRequestCLI(integrationAPIGenericRequestSpec{Command: "wav2lip list", Usage: "integration api wav2lip list --agentId ID [--status all|queued|running|completed|cancelled|failed] [--page N] [--addr URL] [--port 8080]", Description: "List Wav2Lip tasks; the server returns five tasks per page.", Method: http.MethodGet, Path: "/api/wav2lip/tasks", QueryFlags: integrationAPIMediaTaskQueryFlags()}, args[1:], stdout, stderr)
	case "create":
		return runIntegrationAPIWav2LipCreateCLI(args[1:], stdout, stderr)
	case "cancel":
		return runIntegrationAPIMediaTaskActionCLI("wav2lip", "cancel", "/api/wav2lip/tasks/cancel", "Cancel one queued or running Wav2Lip task.", args[1:], stdout, stderr)
	case "restart":
		return runIntegrationAPIMediaTaskActionCLI("wav2lip", "restart", "/api/wav2lip/tasks/restart", "Restart one failed or cancelled Wav2Lip task; its previous log is cleared.", args[1:], stdout, stderr)
	case "delete":
		return runIntegrationAPIMediaTaskActionCLI("wav2lip", "delete", "/api/wav2lip/tasks/delete", "Delete one failed or cancelled Wav2Lip task record; files are preserved.", args[1:], stdout, stderr)
	case "log":
		return runIntegrationAPIGenericRequestCLI(integrationAPIGenericRequestSpec{Command: "wav2lip log", Usage: "integration api wav2lip log --agentId ID --id TASK_ID [--addr URL] [--port 8080]", Description: "Read one Wav2Lip task and its persisted execution log.", Method: http.MethodGet, Path: "/api/wav2lip/tasks/log", QueryFlags: []integrationAPIQueryFlag{{QueryKey: "agentId", Names: []string{"agentId", "agent"}, Usage: "agent id"}, {QueryKey: "id", Names: []string{"id"}, Usage: "Wav2Lip task id"}}}, args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown wav2lip api command: %s\n", args[0])
		printIntegrationAPIWav2LipHelp(stderr)
		return 1
	}
}
