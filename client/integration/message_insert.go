package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"integration/messageinsert"
)

const (
	messageInsertPublishLimit = 5
	messageInsertListLimit    = 20
)

type messageInsertPublishItem struct {
	Tid     string `json:"tid"`
	Message string `json:"message"`
}

func openIntegrationMessageInsertDB() (*sql.DB, func(), error) {
	if cronDB != nil {
		if err := messageinsert.EnsureSchema(cronDB); err != nil {
			return nil, nil, err
		}
		return cronDB, func() {}, nil
	}

	dbPath := resolveIntegrationDBPath()
	if err := ensureParentDir(dbPath); err != nil {
		return nil, nil, err
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(0)
	if err := messageinsert.EnsureSchema(db); err != nil {
		_ = db.Close()
		return nil, nil, err
	}
	return db, func() { _ = db.Close() }, nil
}

func loadPendingMessageInsertPublishItems(chatID string, limit int) ([]messageInsertPublishItem, error) {
	if strings.TrimSpace(chatID) == "" {
		return nil, nil
	}
	db, closeFn, err := openIntegrationMessageInsertDB()
	if err != nil {
		return nil, err
	}
	defer closeFn()

	items, err := messageinsert.ListPending(db, chatID, limit)
	if err != nil {
		return nil, err
	}
	result := make([]messageInsertPublishItem, 0, len(items))
	for _, item := range items {
		result = append(result, messageInsertPublishItem{
			Tid:     strings.TrimSpace(item.Tid),
			Message: strings.TrimSpace(item.Message),
		})
	}
	return result, nil
}

func markPublishedMessageInsertPublishItems(chatID string, items []messageInsertPublishItem) error {
	if strings.TrimSpace(chatID) == "" || len(items) == 0 {
		return nil
	}
	tids := make([]string, 0, len(items))
	for _, item := range items {
		if tid := strings.TrimSpace(item.Tid); tid != "" {
			tids = append(tids, tid)
		}
	}
	if len(tids) == 0 {
		return nil
	}
	db, closeFn, err := openIntegrationMessageInsertDB()
	if err != nil {
		return err
	}
	defer closeFn()

	_, err = messageinsert.MarkPublished(db, chatID, tids, time.Now())
	return err
}

func markUploadedMessageInsertTIDs(chatID string, tids []string) error {
	if strings.TrimSpace(chatID) == "" || len(tids) == 0 {
		return nil
	}
	db, closeFn, err := openIntegrationMessageInsertDB()
	if err != nil {
		return err
	}
	defer closeFn()

	_, err = messageinsert.MarkUploaded(db, chatID, tids, time.Now())
	return err
}

func handleMessageInsertAdd() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload struct {
			AgentID interface{} `json:"agentId"`
			ChatID  interface{} `json:"chatId"`
			Tid     interface{} `json:"tid"`
			Mid     interface{} `json:"mid"`
			Message interface{} `json:"message"`
		}
		if err := decodeMessageInsertPayload(r, &payload); err != nil {
			writeMessageInsertError(w, http.StatusBadRequest, err)
			return
		}
		db, closeFn, err := openIntegrationMessageInsertDB()
		if err != nil {
			writeMessageInsertError(w, http.StatusInternalServerError, err)
			return
		}
		defer closeFn()

		item, err := messageinsert.UpsertPending(
			db,
			normalizeMessageInsertValue(payload.AgentID),
			normalizeMessageInsertValue(payload.ChatID),
			resolveMessageInsertTID(payload.Tid, payload.Mid),
			normalizeMessageInsertValue(payload.Message),
			time.Now(),
		)
		if err != nil {
			if errors.Is(err, messageinsert.ErrAlreadyReported) {
				writeMessageInsertError(w, http.StatusConflict, fmt.Errorf("插入待处理消息已上报，等待结果中"))
				return
			}
			if errors.Is(err, messageinsert.ErrImmutable) {
				writeMessageInsertError(w, http.StatusConflict, fmt.Errorf("插入待处理消息已结束，无法修改"))
				return
			}
			writeMessageInsertError(w, http.StatusBadRequest, err)
			return
		}
		writeMessageInsertJSON(w, http.StatusOK, map[string]interface{}{
			"status": 0,
			"data":   item,
		})
	}
}

func handleMessageInsertDel() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload struct {
			ChatID interface{} `json:"chatId"`
			Tid    interface{} `json:"tid"`
			Mid    interface{} `json:"mid"`
		}
		if err := decodeMessageInsertPayload(r, &payload); err != nil {
			writeMessageInsertError(w, http.StatusBadRequest, err)
			return
		}
		chatID := normalizeMessageInsertValue(payload.ChatID)
		tid := resolveMessageInsertTID(payload.Tid, payload.Mid)

		db, closeFn, err := openIntegrationMessageInsertDB()
		if err != nil {
			writeMessageInsertError(w, http.StatusInternalServerError, err)
			return
		}
		defer closeFn()

		affected, err := messageinsert.Cancel(db, chatID, tid, time.Now())
		if err != nil {
			if errors.Is(err, messageinsert.ErrAlreadyReported) {
				writeMessageInsertError(w, http.StatusConflict, fmt.Errorf("插入待处理消息已上报，等待结果中"))
				return
			}
			if errors.Is(err, messageinsert.ErrImmutable) {
				writeMessageInsertError(w, http.StatusConflict, fmt.Errorf("插入待处理消息已结束，无法修改"))
				return
			}
			writeMessageInsertError(w, http.StatusBadRequest, err)
			return
		}
		writeMessageInsertJSON(w, http.StatusOK, map[string]interface{}{
			"status": 0,
			"data": map[string]interface{}{
				"chatId":   chatID,
				"tid":      tid,
				"affected": affected,
				"status":   messageinsert.StatusCancelled,
			},
		})
	}
}

func handleMessageInsertDelete() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var payload struct {
			ChatID interface{}   `json:"chatId"`
			Tid    interface{}   `json:"tid"`
			Mid    interface{}   `json:"mid"`
			Tids   []interface{} `json:"tids"`
		}
		if err := decodeMessageInsertPayload(r, &payload); err != nil {
			writeMessageInsertError(w, http.StatusBadRequest, err)
			return
		}
		chatID := normalizeMessageInsertValue(payload.ChatID)
		tids := collectMessageInsertTIDs(payload.Tids, payload.Tid, payload.Mid)
		if chatID == "" {
			writeMessageInsertError(w, http.StatusBadRequest, fmt.Errorf("chatId is required"))
			return
		}
		if len(tids) == 0 {
			writeMessageInsertError(w, http.StatusBadRequest, fmt.Errorf("tid is required"))
			return
		}

		db, closeFn, err := openIntegrationMessageInsertDB()
		if err != nil {
			writeMessageInsertError(w, http.StatusInternalServerError, err)
			return
		}
		defer closeFn()

		affected, err := messageinsert.DeleteActive(db, chatID, tids)
		if err != nil {
			writeMessageInsertError(w, http.StatusBadRequest, err)
			return
		}
		writeMessageInsertJSON(w, http.StatusOK, map[string]interface{}{
			"status": 0,
			"data": map[string]interface{}{
				"chatId":   chatID,
				"tids":     tids,
				"affected": affected,
			},
		})
	}
}

func handleMessageInsertList() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		chatID := strings.TrimSpace(r.URL.Query().Get("chatId"))
		if chatID == "" {
			writeMessageInsertError(w, http.StatusBadRequest, fmt.Errorf("chatId is required"))
			return
		}
		limit := messageInsertListLimit
		if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
			parsed, err := strconv.Atoi(rawLimit)
			if err != nil || parsed <= 0 {
				writeMessageInsertError(w, http.StatusBadRequest, fmt.Errorf("limit is invalid"))
				return
			}
			limit = parsed
		}

		db, closeFn, err := openIntegrationMessageInsertDB()
		if err != nil {
			writeMessageInsertError(w, http.StatusInternalServerError, err)
			return
		}
		defer closeFn()

		items, err := messageinsert.ListActive(db, chatID, limit)
		if err != nil {
			writeMessageInsertError(w, http.StatusBadRequest, err)
			return
		}
		writeMessageInsertJSON(w, http.StatusOK, map[string]interface{}{
			"status": 0,
			"data":   items,
		})
	}
}

func printIntegrationMessageInsertHelp() {
	fmt.Println("Usage:")
	fmt.Println("  integration message-insert add --agentId ID --chatId ID --tid TID --message TEXT")
	fmt.Println("  integration message-insert del --chatId ID --tid TID")
	fmt.Println("  integration message-insert list --chatId ID [--limit N]")
	fmt.Println("")
	fmt.Println("Subcommands:")
	fmt.Println("  add               Save one pending inserted message")
	fmt.Println("  del               Mark one inserted message as cancelled")
	fmt.Println("  list              List pending inserted messages for one chat")
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  integration message-insert add --agentId demo --chatId chat-001 --tid 1718966400000 --message 'HELLO'")
	fmt.Println("  integration message-insert del --chatId chat-001 --tid 1718966400000")
	fmt.Println("  integration message-insert list --chatId chat-001 --limit 20")
}

func runIntegrationMessageInsertCLI(args []string) {
	if len(args) == 0 || hasHelpFlag(args) {
		printIntegrationMessageInsertHelp()
		return
	}
	command := strings.TrimSpace(args[0])
	fs := flag.NewFlagSet("integration message-insert", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	agentID := fs.String("agentId", "", "agent id")
	agentAlias := fs.String("agent", "", "agent id")
	chatID := fs.String("chatId", "", "chat id")
	chatAlias := fs.String("chat", "", "chat id")
	tid := fs.String("tid", "", "message insert id")
	midAlias := fs.String("mid", "", "legacy message insert id")
	limit := fs.Int("limit", messageInsertListLimit, "list limit")
	message := fs.String("message", "", "message content")
	fs.Usage = func() { printIntegrationMessageInsertHelp() }
	if err := fs.Parse(args[1:]); err != nil {
		log.Fatal(err)
	}

	targetAgentID := firstNonEmpty(*agentID, *agentAlias)
	targetChatID := firstNonEmpty(*chatID, *chatAlias)
	targetTID := firstNonEmpty(strings.TrimSpace(*tid), strings.TrimSpace(*midAlias))
	targetMessage := strings.TrimSpace(*message)
	if targetMessage == "" && command == "add" && fs.NArg() > 0 {
		targetMessage = strings.TrimSpace(strings.Join(fs.Args(), " "))
	}

	db, closeFn, err := openIntegrationMessageInsertDB()
	if err != nil {
		log.Fatal(err)
	}
	defer closeFn()

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetEscapeHTML(false)

	switch command {
	case "add":
		item, err := messageinsert.UpsertPending(db, targetAgentID, targetChatID, targetTID, targetMessage, time.Now())
		if err != nil {
			log.Fatal(err)
		}
		if err := encoder.Encode(map[string]interface{}{"status": 0, "data": item}); err != nil {
			log.Fatal(err)
		}
	case "del":
		affected, err := messageinsert.Cancel(db, targetChatID, targetTID, time.Now())
		if err != nil {
			log.Fatal(err)
		}
		if err := encoder.Encode(map[string]interface{}{
			"status": 0,
			"data": map[string]interface{}{
				"chatId":   targetChatID,
				"tid":      targetTID,
				"affected": affected,
				"status":   messageinsert.StatusCancelled,
			},
		}); err != nil {
			log.Fatal(err)
		}
	case "list":
		items, err := messageinsert.ListActive(db, targetChatID, *limit)
		if err != nil {
			log.Fatal(err)
		}
		if err := encoder.Encode(map[string]interface{}{"status": 0, "data": items}); err != nil {
			log.Fatal(err)
		}
	default:
		fmt.Println("Unknown message-insert command:", command)
		printIntegrationMessageInsertHelp()
		os.Exit(1)
	}
}

func decodeMessageInsertPayload(r *http.Request, target interface{}) error {
	decoder := json.NewDecoder(r.Body)
	decoder.UseNumber()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid request: %w", err)
	}
	return nil
}

func normalizeMessageInsertValue(value interface{}) string {
	switch current := value.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(current)
	case json.Number:
		return strings.TrimSpace(current.String())
	default:
		return strings.TrimSpace(fmt.Sprint(current))
	}
}

func resolveMessageInsertTID(values ...interface{}) string {
	for _, value := range values {
		if tid := normalizeMessageInsertValue(value); tid != "" {
			return tid
		}
	}
	return ""
}

func collectMessageInsertTIDs(values ...interface{}) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(values))
	var appendTid func(string)
	appendTid = func(value string) {
		tid := strings.TrimSpace(value)
		if tid == "" {
			return
		}
		if _, ok := seen[tid]; ok {
			return
		}
		seen[tid] = struct{}{}
		result = append(result, tid)
	}
	var walk func(interface{})
	walk = func(value interface{}) {
		switch current := value.(type) {
		case nil:
			return
		case []interface{}:
			for _, item := range current {
				walk(item)
			}
		case []string:
			for _, item := range current {
				appendTid(item)
			}
		default:
			appendTid(normalizeMessageInsertValue(current))
		}
	}
	for _, value := range values {
		walk(value)
	}
	return result
}

func writeMessageInsertJSON(w http.ResponseWriter, statusCode int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeMessageInsertError(w http.ResponseWriter, statusCode int, err error) {
	writeMessageInsertJSON(w, statusCode, map[string]interface{}{
		"status":  1,
		"content": strings.TrimSpace(err.Error()),
	})
}
