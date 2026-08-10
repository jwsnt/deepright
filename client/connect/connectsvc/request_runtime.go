package connectsvc

import (
	"crypto/sha1"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

type CompletedTaskDetail struct {
	ID            int
	TaskType      string
	MetaRef       string
	ResultContent string
	ReplyUUID     string
}

type CompletedReplyDispatchOptions struct {
	QueryDB           *sql.DB
	Service           *Service
	DefaultTaskType   string
	SinceUnix         int64
	ResolveCallback   func(pluginKey string) (string, error)
	SendReply         func(callbackPath string, flags map[string]string) error
	SkipSendError     func(error) bool
	BuildFlags        func(pluginKey string, request Request, replyText string) (map[string]string, error)
	MarkDetailReplied func(detailID int) error
	Logger            func(format string, args ...interface{})
}

type ImmediateCronTaskSeed struct {
	RawTime        string
	ExecTime       int64
	AgentID        string
	ChatID         string
	TaskType       string
	Model          string
	Thinking       bool
	Verify         bool
	RouterDisable  bool
	Content        string
	ResponseSchema string
	MetaRef        string
	Started        int
	LastRequestID  string
}

type ImmediateCronTaskResult struct {
	MetaID   int
	DetailID int
	ChatID   string
	Task     ImmediateCronTaskSeed
}

type ImmediateCronTaskPersistence struct {
	InsertMeta      func(seed *ImmediateCronTaskSeed) (int, error)
	AppendMetaLog   func(metaID int, seed *ImmediateCronTaskSeed)
	InsertDetail    func(metaID int, chatID string, seed *ImmediateCronTaskSeed) (int, error)
	AppendDetailLog func(detailID, metaID int, chatID string, seed *ImmediateCronTaskSeed)
}

type PendingRequestPlan struct {
	DetailStarted int
	NextStatus    int
	Notify        bool
}

type PendingRequestNotification struct {
	RuntimeKey string
	Meta       Meta
	Request    Request
	Created    any
}

type PendingRequestSyncOptions struct {
	Service          *Service
	Logger           func(format string, args ...interface{})
	RuntimeKey       func(meta Meta) string
	BuildTextContent func(reqs []Request) string
	DecidePlan       func(meta Meta, reqs []Request, now time.Time, textContent string) (*PendingRequestPlan, error)
	CreateTask       func(meta *Meta, reqs []Request, detailStarted int) (any, string, error)
	AfterCreate      func(meta Meta, selected []Request, plan PendingRequestPlan, created any) error
	Now              func() time.Time
}

func LastRequest(reqs []Request) *Request {
	if len(reqs) == 0 {
		return nil
	}
	last := reqs[0]
	for _, req := range reqs[1:] {
		if req.ID >= last.ID {
			last = req
		}
	}
	return &last
}

func LastRequestID(reqs []Request) string {
	last := LastRequest(reqs)
	if last == nil || last.ID <= 0 {
		return ""
	}
	return fmt.Sprintf("%d", last.ID)
}

func LastMetaRefID(metaRef string) string {
	parts := strings.Split(strings.TrimSpace(metaRef), ",")
	for i := len(parts) - 1; i >= 0; i-- {
		item := strings.TrimSpace(parts[i])
		if item != "" {
			return item
		}
	}
	return ""
}

func FindRequestByID(svc *Service, key string, requestID int) (*Request, error) {
	if svc == nil {
		return nil, fmt.Errorf("connect service is required")
	}
	if requestID <= 0 {
		return nil, fmt.Errorf("invalid request id: %d", requestID)
	}
	reqs, err := svc.ListRequests(RequestFilter{
		Key:     strings.TrimSpace(key),
		AfterID: requestID - 1,
		Limit:   1,
	})
	if err != nil {
		return nil, err
	}
	if len(reqs) == 0 || reqs[0].ID != requestID {
		return nil, nil
	}
	req := reqs[0]
	return &req, nil
}

func RequestCreatedAt(req Request) (time.Time, error) {
	createdAt := strings.TrimSpace(req.CreatedAt)
	if createdAt == "" {
		return time.Time{}, fmt.Errorf("empty request createdAt: %d", req.ID)
	}
	parsed, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse request createdAt %d: %w", req.ID, err)
	}
	return parsed, nil
}

func OldestRequestTime(reqs []Request) (time.Time, error) {
	if len(reqs) == 0 {
		return time.Time{}, fmt.Errorf("requests are required")
	}
	var oldest time.Time
	for i, req := range reqs {
		createdAt, err := RequestCreatedAt(req)
		if err != nil {
			return time.Time{}, err
		}
		if i == 0 || createdAt.Before(oldest) {
			oldest = createdAt
		}
	}
	return oldest, nil
}

func ListCompletedTaskDetails(db *sql.DB, defaultTaskType string, sinceUnix int64) ([]CompletedTaskDetail, error) {
	if db == nil {
		return nil, fmt.Errorf("db is required")
	}
	rows, err := db.Query(`
		SELECT id, task_type, meta_ref, result_content, reply_uuid
		FROM task_detail
		WHERE started = 3
		  AND task_type != ?
		  AND replied_at = ''
		  AND reply_state = 'pending'
		  AND TRIM(result_content) != ''
		  AND exec_time >= ?
		ORDER BY exec_time, id
	`, strings.TrimSpace(defaultTaskType), sinceUnix)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]CompletedTaskDetail, 0)
	for rows.Next() {
		var item CompletedTaskDetail
		if err := rows.Scan(&item.ID, &item.TaskType, &item.MetaRef, &item.ResultContent, &item.ReplyUUID); err != nil {
			continue
		}
		item.TaskType = strings.TrimSpace(item.TaskType)
		items = append(items, item)
	}
	return items, rows.Err()
}

// completedReplyUUID produces a stable RFC 4122 version 5 UUID for one
// externally visible completion reply. It is intentionally derived from the
// task detail rather than task metadata: a detail represents exactly one
// execution, while a meta record can create many executions.
func completedReplyUUID(detailID int, taskType string) string {
	source := fmt.Sprintf("deepright:completion-reply:%s:%d", strings.TrimSpace(taskType), detailID)
	digest := sha1.Sum([]byte(source))
	digest[6] = (digest[6] & 0x0f) | 0x50
	digest[8] = (digest[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		digest[0:4], digest[4:6], digest[6:8], digest[8:10], digest[10:16])
}

// claimCompletedReply moves one pending detail into sending. The conditional
// update is the cross-process lock: only its winner may call a plugin.
func claimCompletedReply(db *sql.DB, detail CompletedTaskDetail) (string, bool, error) {
	if db == nil || detail.ID <= 0 {
		return "", false, nil
	}
	replyUUID := strings.TrimSpace(detail.ReplyUUID)
	if replyUUID == "" {
		replyUUID = completedReplyUUID(detail.ID, detail.TaskType)
	}
	result, err := db.Exec(`
		UPDATE task_detail
		SET reply_state = 'sending', reply_uuid = ?, reply_started_at = ?
		WHERE id = ? AND started = 3 AND replied_at = '' AND reply_state = 'pending'
	`, replyUUID, time.Now().Unix(), detail.ID)
	if err != nil {
		return "", false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return "", false, err
	}
	return replyUUID, affected == 1, nil
}

func markCompletedReplyUnknown(db *sql.DB, detailID int) error {
	if db == nil || detailID <= 0 {
		return nil
	}
	_, err := db.Exec(`UPDATE task_detail SET reply_state = 'unknown' WHERE id = ? AND reply_state = 'sending'`, detailID)
	return err
}

func BuildImmediateCronTaskSeed(meta *Meta, reqs []Request, started int, buildContent func([]Request) string) (*ImmediateCronTaskSeed, error) {
	if meta == nil {
		return nil, fmt.Errorf("connect meta is required")
	}
	if len(reqs) == 0 {
		return nil, nil
	}
	if started < 0 || started > 3 {
		return nil, fmt.Errorf("invalid started status: %d", started)
	}
	if buildContent == nil {
		return nil, fmt.Errorf("build content callback is required")
	}

	content := strings.TrimSpace(buildContent(reqs))
	if content == "" {
		return nil, nil
	}

	metaRefs := make([]string, 0, len(reqs))
	for _, req := range reqs {
		metaRefs = append(metaRefs, fmt.Sprintf("%d", req.ID))
	}

	return &ImmediateCronTaskSeed{
		RawTime:        time.Now().Format("2006-01-02 15:04"),
		ExecTime:       time.Now().Unix(),
		AgentID:        strings.TrimSpace(meta.AgentID),
		ChatID:         strings.TrimSpace(meta.ChatID),
		TaskType:       strings.TrimSpace(firstNonEmpty(meta.Key, meta.Name)),
		Model:          strings.TrimSpace(meta.Model),
		Thinking:       meta.Thinking,
		Verify:         meta.Verify,
		RouterDisable:  meta.RouterDisable,
		Content:        content,
		ResponseSchema: strings.TrimSpace(LastRequest(reqs).ResponseSchema),
		MetaRef:        strings.Join(metaRefs, ","),
		Started:        started,
		LastRequestID:  LastRequestID(reqs),
	}, nil
}

func CreateImmediateCronTask(meta *Meta, reqs []Request, started int, buildContent func([]Request) string, persist ImmediateCronTaskPersistence) (*ImmediateCronTaskResult, error) {
	if persist.InsertMeta == nil || persist.InsertDetail == nil {
		return nil, fmt.Errorf("cron task persistence callbacks are required")
	}

	seed, err := BuildImmediateCronTaskSeed(meta, reqs, started, buildContent)
	if err != nil || seed == nil {
		return nil, err
	}

	metaID, err := persist.InsertMeta(seed)
	if err != nil {
		return nil, err
	}
	if persist.AppendMetaLog != nil {
		persist.AppendMetaLog(metaID, seed)
	}

	chatID := strings.TrimSpace(seed.ChatID)
	if chatID == "" {
		chatID = fmt.Sprintf("%d@0", metaID)
	}

	detailID, err := persist.InsertDetail(metaID, chatID, seed)
	if err != nil {
		return nil, err
	}
	if persist.AppendDetailLog != nil && detailID > 0 {
		persist.AppendDetailLog(detailID, metaID, chatID, seed)
	}

	return &ImmediateCronTaskResult{
		MetaID:   metaID,
		DetailID: detailID,
		ChatID:   chatID,
		Task:     *seed,
	}, nil
}

func DispatchCompletedReplies(opts CompletedReplyDispatchOptions) {
	if opts.QueryDB == nil || opts.Service == nil {
		return
	}
	if opts.ResolveCallback == nil || opts.SendReply == nil || opts.BuildFlags == nil || opts.MarkDetailReplied == nil {
		return
	}

	details, err := ListCompletedTaskDetails(opts.QueryDB, opts.DefaultTaskType, opts.SinceUnix)
	if err != nil {
		logDispatch(opts.Logger, "[connect-cron] list completed detail failed: %v", err)
		return
	}

	for _, detail := range details {
		replyUUID, claimed, err := claimCompletedReply(opts.QueryDB, detail)
		if err != nil {
			logDispatch(opts.Logger, "[connect-cron] claim completed reply failed: detail=%d: %v", detail.ID, err)
			continue
		}
		if !claimed {
			continue
		}
		markUnknown := func(format string, args ...interface{}) {
			logDispatch(opts.Logger, format, args...)
			if err := markCompletedReplyUnknown(opts.QueryDB, detail.ID); err != nil {
				logDispatch(opts.Logger, "[connect-cron] mark completed reply unknown failed: detail=%d uuid=%s: %v", detail.ID, replyUUID, err)
			}
		}
		requestIDText := LastMetaRefID(detail.MetaRef)
		if requestIDText == "" {
			markUnknown("[connect-cron] completed reply has no target request: detail=%d uuid=%s", detail.ID, replyUUID)
			continue
		}
		requestID, err := strconvAtoi(requestIDText)
		if err != nil || requestID <= 0 {
			markUnknown("[connect-cron] completed reply has invalid target request: detail=%d uuid=%s", detail.ID, replyUUID)
			continue
		}
		replyText := strings.TrimSpace(detail.ResultContent)
		if replyText == "" {
			markUnknown("[connect-cron] completed reply has empty result: detail=%d uuid=%s", detail.ID, replyUUID)
			continue
		}

		startedStatus := RequestStatusStarted
		request, err := FindRequestByID(opts.Service, detail.TaskType, requestID)
		if err != nil {
			markUnknown("[connect-cron] get target request failed: detail=%d request=%d uuid=%s: %v", detail.ID, requestID, replyUUID, err)
			continue
		}
		if request == nil || request.Status != startedStatus {
			markUnknown("[connect-cron] target request is not started: detail=%d request=%d uuid=%s", detail.ID, requestID, replyUUID)
			continue
		}

		callbackPath, err := opts.ResolveCallback(detail.TaskType)
		if err != nil {
			markUnknown("[connect-cron] resolve callback failed: detail=%d plugin=%s uuid=%s: %v", detail.ID, detail.TaskType, replyUUID, err)
			continue
		}

		flags, err := opts.BuildFlags(detail.TaskType, *request, replyText)
		if err != nil {
			markUnknown("[connect-cron] build callback flags failed: detail=%d request=%d uuid=%s: %v", detail.ID, request.ID, replyUUID, err)
			continue
		}
		if flags == nil {
			flags = make(map[string]string)
		}
		flags["idempotency-key"] = replyUUID
		if err := opts.SendReply(callbackPath, flags); err != nil {
			markUnknown("[connect-cron] notify completed failed: detail=%d request=%d uuid=%s: %v", detail.ID, request.ID, replyUUID, err)
			continue
		}

		repliedStatus := RequestStatusReplied
		if _, err := opts.Service.UpdateRequestStatus(RequestStatusUpdate{
			ID:     request.ID,
			Key:    request.Key,
			From:   &startedStatus,
			To:     repliedStatus,
			Strict: true,
		}); err != nil {
			markUnknown("[connect-cron] update replied failed: detail=%d request=%d uuid=%s: %v", detail.ID, request.ID, replyUUID, err)
			continue
		}

		completedStatus := RequestStatusCompleted
		earlierReqs, err := opts.Service.ListRequests(RequestFilter{
			Key:    request.Key,
			Limit:  1000,
			Status: &startedStatus,
		})
		if err != nil {
			logDispatch(opts.Logger, "[connect-cron] list earlier started failed: detail=%d request=%d: %v", detail.ID, request.ID, err)
		} else {
			for _, earlier := range earlierReqs {
				if earlier.ID >= request.ID {
					continue
				}
				_, _ = opts.Service.UpdateRequestStatus(RequestStatusUpdate{
					ID:  earlier.ID,
					Key: earlier.Key,
					To:  completedStatus,
				})
			}
		}

		if err := opts.MarkDetailReplied(detail.ID); err != nil {
			markUnknown("[connect-cron] mark completed reply sent failed: detail=%d request=%d uuid=%s: %v", detail.ID, request.ID, replyUUID, err)
		}
	}
}

func SyncPendingRequests(opts PendingRequestSyncOptions) []PendingRequestNotification {
	if opts.Service == nil || opts.DecidePlan == nil || opts.CreateTask == nil {
		return nil
	}

	nowFn := opts.Now
	if nowFn == nil {
		nowFn = time.Now
	}

	metas, err := opts.Service.ListMeta(false)
	if err != nil {
		logDispatch(opts.Logger, "[connect-cron] list meta failed: %v", err)
		return nil
	}

	notifications := make([]PendingRequestNotification, 0)
	for _, meta := range metas {
		pendingStatus := RequestStatusPending
		reqs, err := opts.Service.ListRequests(RequestFilter{
			Key:    meta.Key,
			Limit:  1000,
			Status: &pendingStatus,
		})
		if err != nil {
			logDispatch(opts.Logger, "[connect-cron] list requests failed: %s: %v", meta.Name, err)
			continue
		}
		if len(reqs) == 0 {
			continue
		}

		sort.Slice(reqs, func(i, j int) bool { return reqs[i].ID < reqs[j].ID })

		textContent := ""
		if opts.BuildTextContent != nil {
			textContent = strings.TrimSpace(opts.BuildTextContent(reqs))
		}

		plan, err := opts.DecidePlan(meta, reqs, nowFn(), textContent)
		if err != nil {
			logDispatch(opts.Logger, "[connect-cron] decide pending plan failed: %s: %v", meta.Name, err)
			continue
		}
		if plan == nil {
			continue
		}

		selected := make([]Request, 0, len(reqs))
		for _, req := range reqs {
			updated, err := opts.Service.UpdateRequestStatus(RequestStatusUpdate{
				ID:     req.ID,
				Key:    meta.Key,
				From:   &pendingStatus,
				To:     plan.NextStatus,
				Strict: true,
			})
			if err != nil {
				continue
			}
			selected = append(selected, *updated)
		}
		if len(selected) == 0 {
			continue
		}

		created, lastRequestID, err := opts.CreateTask(&meta, selected, plan.DetailStarted)
		if err != nil {
			logDispatch(opts.Logger, "[connect-cron] create cron task failed: %s: %v", meta.Name, err)
			for _, req := range selected {
				_, _ = opts.Service.UpdateRequestStatus(RequestStatusUpdate{ID: req.ID, Key: meta.Key, To: pendingStatus})
			}
			continue
		}

		if opts.AfterCreate != nil {
			if err := opts.AfterCreate(meta, selected, *plan, created); err != nil {
				logDispatch(opts.Logger, "[connect-cron] post create hook failed: %s: %v", meta.Name, err)
			}
		}

		if !plan.Notify {
			continue
		}
		lastReq := LastRequest(selected)
		if lastReq == nil {
			continue
		}
		if strings.TrimSpace(lastRequestID) != "" && fmt.Sprintf("%d", lastReq.ID) != strings.TrimSpace(lastRequestID) {
			continue
		}

		runtimeKey := strings.TrimSpace(firstNonEmpty(meta.Key, meta.Name))
		if opts.RuntimeKey != nil {
			runtimeKey = strings.TrimSpace(opts.RuntimeKey(meta))
		}
		notifications = append(notifications, PendingRequestNotification{
			RuntimeKey: runtimeKey,
			Meta:       meta,
			Request:    *lastReq,
			Created:    created,
		})
	}

	return notifications
}

func BuildPluginCallbackFlags(pluginKey string, request Request, replyText, connectBin string) (map[string]string, error) {
	messageJSON, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("marshal connect request %d: %w", request.ID, err)
	}
	return map[string]string{
		"name":        strings.TrimSpace(pluginKey),
		"message":     string(messageJSON),
		"content":     strings.TrimSpace(replyText),
		"connect-bin": strings.TrimSpace(connectBin),
	}, nil
}

func logDispatch(logger func(format string, args ...interface{}), format string, args ...interface{}) {
	if logger != nil {
		logger(format, args...)
	}
}

func strconvAtoi(raw string) (int, error) {
	var value int
	_, err := fmt.Sscanf(strings.TrimSpace(raw), "%d", &value)
	return value, err
}
