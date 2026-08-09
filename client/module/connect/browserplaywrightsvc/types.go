package browserplaywrightsvc

import "time"

const (
	DefaultName              = "browser_playwright"
	DefaultDisplayName       = "Playwright Browser"
	DefaultChromeUserAgent   = "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
	DefaultNavigatorPlatform = "MacIntel"
	DefaultScreenWidth       = 2560
	DefaultScreenHeight      = 1440
	DefaultViewportWidth     = 2560
	DefaultViewportHeight    = 1440
	DefaultMaxTouchPoints    = 5
	DefaultWebGLVendor       = "Apple Inc."
	DefaultWebGLRenderer     = "Apple M1 Pro"
	DefaultLocale            = "en-US"
	DefaultTimezoneID        = "UTC"

	defaultStateDir       = ".browser_playwright"
	defaultPIDFileName    = "browser_playwright.pid"
	defaultLogDirName     = ""
	defaultAddr           = "127.0.0.1:18333"
	defaultBrowserName    = "chromium"
	defaultChromeCDP      = "http://127.0.0.1:9222"
	defaultBrowserTimeout = 120 * time.Second
	defaultActionMS       = 5000
	defaultNavMS          = 60000
	defaultLogMaxSizeMB   = 10
	defaultLogMaxFiles    = 4
)

type Options struct {
	StateDir       string
	Addr           string
	LogFile        string
	PIDFile        string
	DriverDir      string
	ExecutablePath string
	BrowserTimeout time.Duration
	BrowserRetry   int
}

type StartResult struct {
	Status  string `json:"status"`
	PID     int    `json:"pid,omitempty"`
	PIDFile string `json:"pidFile,omitempty"`
	LogFile string `json:"logFile,omitempty"`
	Addr    string `json:"addr,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

type NameInfo struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

type CommandResult struct {
	Session     string         `json:"session,omitempty"`
	Command     string         `json:"command,omitempty"`
	Page        *PageInfo      `json:"page,omitempty"`
	Tabs        []TabInfo      `json:"tabs,omitempty"`
	Snapshot    *SnapshotInfo  `json:"snapshot,omitempty"`
	Cookies     []CookieInfo   `json:"cookies,omitempty"`
	Storage     map[string]any `json:"storage,omitempty"`
	StatePath   string         `json:"statePath,omitempty"`
	OutputPath  string         `json:"outputPath,omitempty"`
	OutputPaths []string       `json:"outputPaths,omitempty"`
	Value       any            `json:"value,omitempty"`
	Message     string         `json:"message,omitempty"`
	Requests    []RequestInfo  `json:"requests,omitempty"`
	Console     []ConsoleEntry `json:"console,omitempty"`
}

type PageInfo struct {
	URL   string `json:"url,omitempty"`
	Title string `json:"title,omitempty"`
	Index int    `json:"index,omitempty"`
}

type TabInfo struct {
	Index  int    `json:"index"`
	URL    string `json:"url,omitempty"`
	Title  string `json:"title,omitempty"`
	Active bool   `json:"active"`
}

type SnapshotInfo struct {
	Path        string        `json:"path"`
	PageURL     string        `json:"pageUrl,omitempty"`
	PageTitle   string        `json:"pageTitle,omitempty"`
	CreatedAt   time.Time     `json:"createdAt"`
	Depth       int           `json:"depth,omitempty"`
	Target      string        `json:"target,omitempty"`
	Items       []SnapshotRef `json:"items,omitempty"`
	Description string        `json:"description,omitempty"`
}

type SnapshotRef struct {
	Ref   string  `json:"ref"`
	Text  string  `json:"text,omitempty"`
	Role  string  `json:"role,omitempty"`
	Tag   string  `json:"tag,omitempty"`
	ID    string  `json:"id,omitempty"`
	Name  string  `json:"name,omitempty"`
	Type  string  `json:"type,omitempty"`
	Href  string  `json:"href,omitempty"`
	Value string  `json:"value,omitempty"`
	X     float64 `json:"x,omitempty"`
	Y     float64 `json:"y,omitempty"`
	W     float64 `json:"w,omitempty"`
	H     float64 `json:"h,omitempty"`
}

type CookieInfo struct {
	Name      string  `json:"name"`
	Value     string  `json:"value"`
	Domain    string  `json:"domain"`
	Path      string  `json:"path"`
	Expires   float64 `json:"expires,omitempty"`
	HTTPOnly  bool    `json:"httpOnly,omitempty"`
	Secure    bool    `json:"secure,omitempty"`
	SameSite  string  `json:"sameSite,omitempty"`
	Partition string  `json:"partitionKey,omitempty"`
}

type ConsoleEntry struct {
	Type      string    `json:"type"`
	Text      string    `json:"text"`
	URL       string    `json:"url,omitempty"`
	Line      int       `json:"line,omitempty"`
	Column    int       `json:"column,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

type RequestInfo struct {
	Index      int               `json:"index"`
	URL        string            `json:"url"`
	Method     string            `json:"method"`
	Resource   string            `json:"resourceType,omitempty"`
	Status     int               `json:"status,omitempty"`
	OK         bool              `json:"ok,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	Failure    string            `json:"failure,omitempty"`
	ResponseTo string            `json:"responseUrl,omitempty"`
}

type SessionSummary struct {
	Name       string    `json:"name"`
	Browser    string    `json:"browser"`
	CDP        string    `json:"cdp,omitempty"`
	Headed     bool      `json:"headed"`
	Persistent bool      `json:"persistent"`
	ProfileDir string    `json:"profileDir,omitempty"`
	LastURL    string    `json:"lastUrl,omitempty"`
	LastTitle  string    `json:"lastTitle,omitempty"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type SessionConfig struct {
	Name           string   `json:"name"`
	Browser        string   `json:"browser"`
	CDP            string   `json:"cdp,omitempty"`
	Channel        string   `json:"channel,omitempty"`
	Persistent     bool     `json:"persistent"`
	ProfileDir     string   `json:"profileDir,omitempty"`
	Headless       bool     `json:"headless"`
	Width          int      `json:"width,omitempty"`
	Height         int      `json:"height,omitempty"`
	UserAgent      string   `json:"userAgent,omitempty"`
	TimeoutMS      float64  `json:"timeoutMs,omitempty"`
	NavTimeout     float64  `json:"navTimeoutMs,omitempty"`
	Locale         string   `json:"locale,omitempty"`
	Languages      []string `json:"languages,omitempty"`
	TimezoneID     string   `json:"timezoneId,omitempty"`
	MaxTouchPoints int      `json:"maxTouchPoints,omitempty"`
	WebGLVendor    string   `json:"webglVendor,omitempty"`
	WebGLRenderer  string   `json:"webglRenderer,omitempty"`
	DownloadDir    string   `json:"downloadDir,omitempty"`
	VideoDir       string   `json:"videoDir,omitempty"`
	TracePath      string   `json:"tracePath,omitempty"`
	RecordVideo    bool     `json:"recordVideo,omitempty"`
}

type CommandRequest struct {
	Session string            `json:"session"`
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Flags   map[string]string `json:"flags,omitempty"`
}

type ManagedInstanceRecord struct {
	AgentID string `json:"agentId"`
	ChatID  string `json:"chatId"`
	Port    int    `json:"port"`
	PID     int    `json:"pid"`
	CDP     string `json:"cdp"`
}

type ManagedInstanceProvider interface {
	GetManagedInstance(flags map[string]string, agentID, chatID string) (ManagedInstanceRecord, error)
	CreateManagedInstance(flags map[string]string, agentID, chatID string) (ManagedInstanceRecord, error)
}
