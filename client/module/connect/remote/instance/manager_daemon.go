package instance

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"connect/connectsvc"
)

func RunManagerDaemon(flags map[string]string) error {
	runtime, err := runtimePaths(flags)
	if err != nil {
		return err
	}
	if explicit := strings.TrimSpace(connectsvc.FirstValue(flags, "pid-file")); explicit != "" {
		runtime.PIDFile = explicit
	}
	if explicit := strings.TrimSpace(connectsvc.FirstValue(flags, "manager-socket")); explicit != "" {
		runtime.ManagerSocket = explicit
	}
	if explicit := strings.TrimSpace(connectsvc.FirstValue(flags, "runtime-dir")); explicit != "" {
		runtime.RuntimeDir = explicit
	}
	if explicit := strings.TrimSpace(connectsvc.FirstValue(flags, "state")); explicit != "" {
		runtime.StatePath = explicit
	}
	if explicit := strings.TrimSpace(connectsvc.FirstValue(flags, "log")); explicit != "" {
		runtime.LogPath = explicit
	}

	if err := mkdirAllFn(runtime.RootDir, 0o755); err != nil {
		return err
	}
	if err := mkdirAllFn(filepath.Dir(runtime.ManagerSocket), 0o755); err != nil {
		return err
	}
	loggerWriter, err := openLogWriter(runtime.LogPath)
	if err != nil {
		return err
	}
	defer loggerWriter.Close()
	logger := log.New(loggerWriter, "remote-manager ", log.LstdFlags)

	if err := writeFileFn(runtime.PIDFile, []byte(strconv.Itoa(os.Getpid())+"\n"), 0o644); err != nil {
		return err
	}
	defer removeFn(runtime.PIDFile)

	_ = removeFn(runtime.ManagerSocket)
	listener, err := managerListenFn("unix", runtime.ManagerSocket)
	if err != nil {
		return err
	}
	defer func() {
		_ = listener.Close()
		_ = removeFn(runtime.ManagerSocket)
	}()

	signalCh := make(chan os.Signal, 1)
	signalNotify(signalCh)
	defer signalStop(signalCh)
	go func() {
		<-signalCh
		logger.Printf("用参数：插件=remote，做了：收到退出信号，准备关闭远程连接管理服务")
		_ = listener.Close()
	}()

	idleTicker := time.NewTicker(defaultIdleCheckEvery)
	defer idleTicker.Stop()
	go func() {
		for range idleTicker.C {
			if _, err := pruneIdleSessions(withRuntimeFlags(flags, runtime), defaultIdleTimeout, logger); err != nil {
				logger.Printf("用参数：原因=%v，做了：巡检空闲远程连接时出错", err)
			}
		}
	}()

	for {
		if unixListener, ok := listener.(*net.UnixListener); ok {
			_ = unixListener.SetDeadline(time.Now().Add(time.Second))
		}
		conn, err := listener.Accept()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				continue
			}
			if strings.Contains(strings.ToLower(err.Error()), "closed network connection") {
				return nil
			}
			return err
		}
		go func(c net.Conn) {
			defer c.Close()
			var req managerRequest
			if err := json.NewDecoder(c).Decode(&req); err != nil {
				_ = json.NewEncoder(c).Encode(managerResponse{OK: false, Error: err.Error()})
				return
			}
			response := handleManagerRequest(req, runtime, flags, logger)
			_ = json.NewEncoder(c).Encode(response)
			if req.Action == "shutdown" {
				_ = listener.Close()
			}
		}(conn)
	}
}

func handleManagerRequest(req managerRequest, runtime Paths, baseFlags map[string]string, logger *log.Logger) managerResponse {
	flags := withRuntimeFlags(baseFlags, runtime)
	for key, value := range req.Flags {
		flags[key] = value
	}
	switch req.Action {
	case "ping":
		return managerResponse{
			OK:          true,
			PID:         os.Getpid(),
			Version:     managerProtocolVersion,
			Fingerprint: fingerprintString(flags),
		}
	case "create":
		item, err := Create(flags)
		if err != nil {
			return managerResponse{OK: false, Error: err.Error()}
		}
		return managerResponse{OK: true, Item: item}
	case "shutdown-session":
		if err := Shutdown(flags); err != nil {
			return managerResponse{OK: false, Error: err.Error()}
		}
		return managerResponse{OK: true}
	case "list":
		items, err := List(flags)
		if err != nil {
			return managerResponse{OK: false, Error: err.Error()}
		}
		return managerResponse{OK: true, Items: items}
	case "get":
		item, err := Get(flags)
		if err != nil {
			return managerResponse{OK: false, Error: err.Error()}
		}
		return managerResponse{OK: true, Item: item}
	case "exec":
		if req.TimeoutMillisecs > 0 {
			flags["exec-timeout-ms"] = strconv.Itoa(req.TimeoutMillisecs)
		}
		result, err := Exec(flags, req.Command)
		if err != nil {
			return managerResponse{OK: false, Error: err.Error()}
		}
		return managerResponse{
			OK:       true,
			Stdout:   result.Stdout,
			Stderr:   result.Stderr,
			ExitCode: result.ExitCode,
		}
	case "shutdown":
		items, err := List(flags)
		if err == nil {
			for _, item := range items {
				if shutdownErr := cleanupRuntime(flags, item, true); shutdownErr != nil && logger != nil {
					logger.Printf("用参数：Agent=%s；会话=%s；原因=%v，做了：批量关闭远程连接时出错", item.AgentID, item.ChatID, shutdownErr)
				}
			}
		}
		_ = saveState(runtime.StatePath, []Record{})
		return managerResponse{OK: true}
	default:
		return managerResponse{OK: false, Error: fmt.Sprintf("unknown manager action: %s", req.Action)}
	}
}
