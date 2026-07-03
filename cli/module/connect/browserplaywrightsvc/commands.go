package browserplaywrightsvc

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
)

func (d *daemonService) cmdOpen(name string, sess *liveSession, req CommandRequest) (*CommandResult, error) {
	page, err := d.activePage(sess)
	if err != nil {
		return nil, err
	}
	if len(req.Args) > 0 && strings.TrimSpace(req.Args[0]) != "" {
		page, err = d.ensureNavigationPage(sess, page)
		if err != nil {
			return nil, err
		}
		if err := d.navigateWithCookies(sess, page, req.Args[0]); err != nil {
			return nil, err
		}
	}
	d.syncPages(sess)
	return resultWithPage(name, req.Command, sess)
}

func (d *daemonService) cmdCreate(name string, sess *liveSession, req CommandRequest) (*CommandResult, error) {
	return d.cmdAttachLike(name, sess, req, "create")
}

func (d *daemonService) cmdAttachLike(name string, sess *liveSession, req CommandRequest, command string) (*CommandResult, error) {
	if strings.TrimSpace(sess.config.CDP) == "" {
		return nil, fmt.Errorf("attach requires --cdp=chrome or --cdp=<url>")
	}
	page, err := d.activePage(sess)
	if err != nil {
		return nil, err
	}
	if len(req.Args) > 0 && strings.TrimSpace(req.Args[0]) != "" {
		page, err = d.ensureNavigationPage(sess, page)
		if err != nil {
			return nil, err
		}
		if err := d.navigateWithCookies(sess, page, req.Args[0]); err != nil {
			return nil, err
		}
	}
	d.syncPages(sess)
	return resultWithPage(name, command, sess)
}

func (d *daemonService) cmdAttach(name string, sess *liveSession, req CommandRequest) (*CommandResult, error) {
	return d.cmdAttachLike(name, sess, req, "attach")
}

func (d *daemonService) cmdGoto(name string, sess *liveSession, req CommandRequest) (*CommandResult, error) {
	if len(req.Args) == 0 {
		return nil, fmt.Errorf("goto requires url")
	}
	page, err := d.activePage(sess)
	if err != nil {
		return nil, err
	}
	page, err = d.ensureNavigationPage(sess, page)
	if err != nil {
		return nil, err
	}
	if err := d.navigateWithCookies(sess, page, req.Args[0]); err != nil {
		return nil, err
	}
	d.syncPages(sess)
	return resultWithPage(name, req.Command, sess)
}

func (d *daemonService) cmdNav(name string, sess *liveSession, req CommandRequest, direction string) (*CommandResult, error) {
	page, err := d.activePage(sess)
	if err != nil {
		return nil, err
	}
	switch direction {
	case "back":
		_, err = page.GoBack(playwright.PageGoBackOptions{WaitUntil: playwright.WaitUntilStateCommit})
	case "forward":
		_, err = page.GoForward(playwright.PageGoForwardOptions{WaitUntil: playwright.WaitUntilStateCommit})
	}
	if err != nil {
		return nil, err
	}
	d.syncPages(sess)
	return resultWithPage(name, req.Command, sess)
}

func (d *daemonService) cmdReload(name string, sess *liveSession, req CommandRequest) (*CommandResult, error) {
	page, err := d.activePage(sess)
	if err != nil {
		return nil, err
	}
	if _, err := page.Reload(playwright.PageReloadOptions{WaitUntil: playwright.WaitUntilStateCommit}); err != nil {
		return nil, err
	}
	d.syncPages(sess)
	return resultWithPage(name, req.Command, sess)
}

func (d *daemonService) cmdClose(name string) (*CommandResult, error) {
	if err := d.closeSession(name); err != nil {
		return nil, err
	}
	return &CommandResult{Session: name, Command: "close", Message: "session closed"}, nil
}

func (d *daemonService) cmdTabList(name string, sess *liveSession) (*CommandResult, error) {
	d.syncPages(sess)
	return &CommandResult{
		Session: name,
		Command: "tab-list",
		Tabs:    tabsFromSession(sess),
		Page:    pageInfo(sess.pages[sess.active], sess.active),
	}, nil
}

func (d *daemonService) cmdTabNew(name string, sess *liveSession, req CommandRequest) (*CommandResult, error) {
	page, err := sess.context.NewPage()
	if err != nil {
		return nil, err
	}
	if len(req.Args) > 0 && strings.TrimSpace(req.Args[0]) != "" {
		if err := d.navigateWithCookies(sess, page, req.Args[0]); err != nil {
			return nil, err
		}
	}
	d.syncPages(sess)
	sess.active = len(sess.pages) - 1
	return resultWithPage(name, "tab-new", sess)
}

func (d *daemonService) navigateWithCookies(sess *liveSession, page playwright.Page, target string) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil
	}
	if err := d.injectChromeCookiesForURL(sess, target); err != nil {
		return err
	}
	startedAt := time.Now()
	if _, err := page.Goto(target, playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded}); err != nil {
		d.logNavigationDiagnostic(sess, "goto", target, time.Since(startedAt))
		return rewriteNavigationError(target, err)
	}
	d.logNavigationDiagnostic(sess, "goto", target, time.Since(startedAt))
	if err := d.injectChromeCookies(sess, page); err != nil {
		return err
	}
	return nil
}

func rewriteNavigationError(target string, err error) error {
	if err == nil {
		return nil
	}
	if !strings.Contains(strings.ToLower(err.Error()), "too many redirects") {
		return err
	}
	target = strings.TrimSpace(target)
	if target == "" {
		return fmt.Errorf("navigation failed: remote site entered a redirect loop: %w", err)
	}
	return fmt.Errorf("navigation failed for %s: remote site entered a redirect loop: %w", target, err)
}

func (d *daemonService) cmdTabClose(name string, sess *liveSession, req CommandRequest) (*CommandResult, error) {
	index := sess.active
	if len(req.Args) > 0 {
		parsed, err := strconv.Atoi(req.Args[0])
		if err != nil {
			return nil, fmt.Errorf("tab-close index must be integer")
		}
		index = parsed
	}
	if index < 0 || index >= len(sess.pages) {
		return nil, fmt.Errorf("tab index out of range")
	}
	if err := sess.pages[index].Close(); err != nil {
		return nil, err
	}
	d.syncPages(sess)
	return resultWithPage(name, "tab-close", sess)
}

func (d *daemonService) cmdTabSelect(name string, sess *liveSession, req CommandRequest) (*CommandResult, error) {
	if len(req.Args) == 0 {
		return nil, fmt.Errorf("tab-select requires index")
	}
	index, err := strconv.Atoi(req.Args[0])
	if err != nil {
		return nil, fmt.Errorf("tab-select index must be integer")
	}
	if index < 0 || index >= len(sess.pages) {
		return nil, fmt.Errorf("tab index out of range")
	}
	sess.active = index
	if err := sess.pages[index].BringToFront(); err != nil {
		return nil, err
	}
	d.syncPages(sess)
	return resultWithPage(name, "tab-select", sess)
}

func (d *daemonService) cmdLocatorAction(name string, sess *liveSession, req CommandRequest) (*CommandResult, error) {
	page, err := d.activePage(sess)
	if err != nil {
		return nil, err
	}
	locator, err := d.resolveLocator(page, name, firstArg(req))
	if err != nil {
		return nil, err
	}
	switch req.Command {
	case "click":
		err = locator.Click()
	case "dblclick":
		err = locator.Dblclick()
	case "fill":
		if len(req.Args) < 2 {
			return nil, fmt.Errorf("fill requires target and text")
		}
		err = locator.Fill(req.Args[1])
	case "type":
		if len(req.Args) == 1 {
			err = page.Keyboard().InsertText(req.Args[0])
		} else {
			err = locator.Fill(req.Args[1])
		}
	case "hover":
		err = locator.Hover()
	case "check":
		err = locator.Check()
	case "uncheck":
		err = locator.Uncheck()
	case "select":
		if len(req.Args) < 2 {
			return nil, fmt.Errorf("select requires target and value")
		}
		_, err = locator.SelectOption(playwright.SelectOptionValues{Values: &[]string{req.Args[1]}})
	}
	if err != nil {
		return nil, err
	}
	d.syncPages(sess)
	return resultWithPage(name, req.Command, sess)
}

func (d *daemonService) cmdKeyboard(name string, sess *liveSession, req CommandRequest, mode string) (*CommandResult, error) {
	page, err := d.activePage(sess)
	if err != nil {
		return nil, err
	}
	if len(req.Args) == 0 {
		return nil, fmt.Errorf("%s requires key", req.Command)
	}
	switch mode {
	case "press":
		err = page.Keyboard().Press(req.Args[0])
	case "down":
		err = page.Keyboard().Down(req.Args[0])
	case "up":
		err = page.Keyboard().Up(req.Args[0])
	}
	if err != nil {
		return nil, err
	}
	return resultWithPage(name, req.Command, sess)
}

func (d *daemonService) cmdMouse(name string, sess *liveSession, req CommandRequest) (*CommandResult, error) {
	page, err := d.activePage(sess)
	if err != nil {
		return nil, err
	}
	mouse := page.Mouse()
	switch req.Command {
	case "mousemove":
		if len(req.Args) < 2 {
			return nil, fmt.Errorf("mousemove requires x y")
		}
		x, err := strconv.ParseFloat(req.Args[0], 64)
		if err != nil {
			return nil, err
		}
		y, err := strconv.ParseFloat(req.Args[1], 64)
		if err != nil {
			return nil, err
		}
		err = mouse.Move(x, y)
		if err != nil {
			return nil, err
		}
	case "mousedown":
		err = mouse.Down()
	case "mouseup":
		err = mouse.Up()
	case "mousewheel":
		if len(req.Args) < 2 {
			return nil, fmt.Errorf("mousewheel requires dx dy")
		}
		dx, err := strconv.ParseFloat(req.Args[0], 64)
		if err != nil {
			return nil, err
		}
		dy, err := strconv.ParseFloat(req.Args[1], 64)
		if err != nil {
			return nil, err
		}
		err = mouse.Wheel(dx, dy)
		if err != nil {
			return nil, err
		}
	}
	return resultWithPage(name, req.Command, sess)
}

func (d *daemonService) cmdDrag(name string, sess *liveSession, req CommandRequest) (*CommandResult, error) {
	if len(req.Args) < 2 {
		return nil, fmt.Errorf("drag requires start and end target")
	}
	page, err := d.activePage(sess)
	if err != nil {
		return nil, err
	}
	start, err := d.resolveLocator(page, name, req.Args[0])
	if err != nil {
		return nil, err
	}
	end, err := d.resolveLocator(page, name, req.Args[1])
	if err != nil {
		return nil, err
	}
	if err := start.DragTo(end); err != nil {
		return nil, err
	}
	return resultWithPage(name, req.Command, sess)
}

func (d *daemonService) cmdUpload(name string, sess *liveSession, req CommandRequest) (*CommandResult, error) {
	page, err := d.activePage(sess)
	if err != nil {
		return nil, err
	}
	selector := strings.TrimSpace(req.Flags["selector"])
	if selector == "" {
		selector = `input[type="file"]`
	}
	if len(req.Args) == 0 {
		return nil, fmt.Errorf("upload requires one or more file paths")
	}
	files := make([]string, 0, len(req.Args))
	for _, item := range req.Args {
		files = append(files, absChild(".", item))
	}
	if err := page.SetInputFiles(selector, files); err != nil {
		return nil, err
	}
	return &CommandResult{Session: name, Command: req.Command, OutputPaths: files, Page: pageInfo(page, sess.active)}, nil
}

func (d *daemonService) cmdSnapshot(name string, sess *liveSession, req CommandRequest) (*CommandResult, error) {
	page, err := d.activePage(sess)
	if err != nil {
		return nil, err
	}
	target := ""
	if len(req.Args) > 0 {
		target = req.Args[0]
	}
	path := strings.TrimSpace(req.Flags["filename"])
	if path == "" {
		path = snapshotPath(d.opts.StateDir, name)
	}
	info, err := captureSnapshot(page, path, target, intFlag(req.Flags, "depth", 0))
	if err != nil {
		return nil, err
	}
	return &CommandResult{
		Session:  name,
		Command:  req.Command,
		Page:     pageInfo(page, sess.active),
		Snapshot: info,
		Message:  info.Description,
	}, nil
}

func (d *daemonService) cmdScreenshot(name string, sess *liveSession, req CommandRequest) (*CommandResult, error) {
	page, err := d.activePage(sess)
	if err != nil {
		return nil, err
	}
	path := strings.TrimSpace(req.Flags["filename"])
	if path == "" {
		path = filepath.Join(sessionDir(d.opts.StateDir, name), "screenshot.png")
	}
	if len(req.Args) > 0 && strings.TrimSpace(req.Args[0]) != "" {
		locator, err := d.resolveLocator(page, name, req.Args[0])
		if err != nil {
			return nil, err
		}
		if _, err := locator.Screenshot(playwright.LocatorScreenshotOptions{Path: &path}); err != nil {
			return nil, err
		}
	} else {
		if _, err := page.Screenshot(playwright.PageScreenshotOptions{Path: &path}); err != nil {
			return nil, err
		}
	}
	return &CommandResult{Session: name, Command: req.Command, OutputPath: path, Page: pageInfo(page, sess.active)}, nil
}

func (d *daemonService) cmdPDF(name string, sess *liveSession, req CommandRequest) (*CommandResult, error) {
	page, err := d.activePage(sess)
	if err != nil {
		return nil, err
	}
	path := strings.TrimSpace(req.Flags["filename"])
	if path == "" {
		path = filepath.Join(sessionDir(d.opts.StateDir, name), "page.pdf")
	}
	if _, err := page.PDF(playwright.PagePdfOptions{Path: &path}); err != nil {
		return nil, err
	}
	return &CommandResult{Session: name, Command: req.Command, OutputPath: path, Page: pageInfo(page, sess.active)}, nil
}

func (d *daemonService) cmdEval(name string, sess *liveSession, req CommandRequest) (*CommandResult, error) {
	if len(req.Args) == 0 {
		return nil, fmt.Errorf("eval requires javascript expression")
	}
	page, err := d.activePage(sess)
	if err != nil {
		return nil, err
	}
	expression, timeoutMS := wrapEvalExpressionWithTimeout(req.Args[0], sess.config.TimeoutMS)
	var value any
	if len(req.Args) > 1 {
		locator, err := d.resolveLocator(page, name, req.Args[1])
		if err != nil {
			return nil, err
		}
		value, err = locator.Evaluate(expression, map[string]any{
			"expression": req.Args[0],
			"timeoutMs":  timeoutMS,
		})
		if err != nil {
			return nil, err
		}
	} else {
		value, err = page.Evaluate(expression, map[string]any{
			"expression": req.Args[0],
			"timeoutMs":  timeoutMS,
		})
		if err != nil {
			return nil, err
		}
	}
	return &CommandResult{Session: name, Command: req.Command, Value: value, Page: pageInfo(page, sess.active)}, nil
}

func wrapEvalExpressionWithTimeout(expression string, timeoutMS float64) (string, float64) {
	if timeoutMS <= 0 {
		timeoutMS = defaultActionMS
	}
	return `(input) => {
const expression = String(input && input.expression || "");
const timeoutMs = Number(input && input.timeoutMs || 0);
const timeoutError = () => {
  const err = new Error("Timeout " + timeoutMs + "ms exceeded.");
  err.name = "TimeoutError";
  return err;
};
const timer = new Promise((_, reject) => {
  setTimeout(() => reject(timeoutError()), timeoutMs);
});
let result;
try {
  result = globalThis.eval(expression);
} catch (error) {
  throw error;
}
return Promise.race([
  Promise.resolve(result),
  timer,
]);
}`, timeoutMS
}

func (d *daemonService) cmdResize(name string, sess *liveSession, req CommandRequest) (*CommandResult, error) {
	if len(req.Args) < 2 {
		return nil, fmt.Errorf("resize requires width and height")
	}
	width, err := strconv.Atoi(req.Args[0])
	if err != nil {
		return nil, err
	}
	height, err := strconv.Atoi(req.Args[1])
	if err != nil {
		return nil, err
	}
	page, err := d.activePage(sess)
	if err != nil {
		return nil, err
	}
	if err := page.SetViewportSize(width, height); err != nil {
		return nil, err
	}
	sess.config.Width = width
	sess.config.Height = height
	return resultWithPage(name, req.Command, sess)
}

func (d *daemonService) cmdStateSave(name string, sess *liveSession, req CommandRequest) (*CommandResult, error) {
	path := firstArg(req)
	if path == "" {
		path = filepath.Join(sessionDir(d.opts.StateDir, name), "storage_state.json")
	}
	_, err := sess.context.StorageState(path)
	if err != nil {
		return nil, err
	}
	return &CommandResult{Session: name, Command: req.Command, StatePath: path}, nil
}

func (d *daemonService) cmdStateLoad(name string, sess *liveSession, req CommandRequest) (*CommandResult, error) {
	if len(req.Args) == 0 {
		return nil, fmt.Errorf("state-load requires filename")
	}
	path := req.Args[0]
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := writeTextFile(filepath.Join(sessionDir(d.opts.StateDir, name), "storage_state.loaded.json"), string(data)); err != nil {
		return nil, err
	}
	return &CommandResult{Session: name, Command: req.Command, StatePath: path, Message: "state file stored for next session reopen"}, nil
}

func (d *daemonService) cmdCookieList(name string, sess *liveSession, req CommandRequest) (*CommandResult, error) {
	page, err := d.activePage(sess)
	if err != nil {
		return nil, err
	}
	cookies, err := sess.context.Cookies(page.URL())
	if err != nil {
		return nil, err
	}
	out := make([]CookieInfo, 0, len(cookies))
	for _, item := range cookies {
		out = append(out, cookieInfo(item))
	}
	return &CommandResult{Session: name, Command: req.Command, Cookies: out, Page: pageInfo(page, sess.active)}, nil
}

func (d *daemonService) cmdCookieGet(name string, sess *liveSession, req CommandRequest) (*CommandResult, error) {
	if len(req.Args) == 0 {
		return nil, fmt.Errorf("cookie-get requires name")
	}
	result, err := d.cmdCookieList(name, sess, req)
	if err != nil {
		return nil, err
	}
	for _, item := range result.Cookies {
		if item.Name == req.Args[0] {
			result.Value = item
			return result, nil
		}
	}
	return nil, fmt.Errorf("cookie not found: %s", req.Args[0])
}

func (d *daemonService) cmdCookieSet(name string, sess *liveSession, req CommandRequest) (*CommandResult, error) {
	if len(req.Args) < 2 {
		return nil, fmt.Errorf("cookie-set requires name and value")
	}
	page, err := d.activePage(sess)
	if err != nil {
		return nil, err
	}
	url := page.URL()
	cookie := playwright.OptionalCookie{
		Name:  req.Args[0],
		Value: req.Args[1],
		URL:   &url,
	}
	if err := sess.context.AddCookies([]playwright.OptionalCookie{cookie}); err != nil {
		return nil, err
	}
	return d.cmdCookieGet(name, sess, CommandRequest{Session: name, Command: "cookie-get", Args: []string{req.Args[0]}})
}

func (d *daemonService) cmdCookieDelete(name string, sess *liveSession, req CommandRequest) (*CommandResult, error) {
	if len(req.Args) == 0 {
		return nil, fmt.Errorf("cookie-delete requires name")
	}
	if err := sess.context.ClearCookies(playwright.BrowserContextClearCookiesOptions{Name: req.Args[0]}); err != nil {
		return nil, err
	}
	return &CommandResult{Session: name, Command: req.Command, Message: "cookie deleted"}, nil
}

func (d *daemonService) cmdCookieClear(name string, sess *liveSession) (*CommandResult, error) {
	if err := sess.context.ClearCookies(); err != nil {
		return nil, err
	}
	return &CommandResult{Session: name, Command: "cookie-clear", Message: "cookies cleared"}, nil
}

func (d *daemonService) cmdStorageList(name string, sess *liveSession, req CommandRequest) (*CommandResult, error) {
	page, err := d.activePage(sess)
	if err != nil {
		return nil, err
	}
	storeName := storageObjectName(req.Command)
	value, err := page.Evaluate(fmt.Sprintf(`() => Object.fromEntries(Object.entries(%s))`, storeName))
	if err != nil {
		return nil, err
	}
	return &CommandResult{Session: name, Command: req.Command, Storage: asStringMap(value), Page: pageInfo(page, sess.active)}, nil
}

func (d *daemonService) cmdStorageGet(name string, sess *liveSession, req CommandRequest) (*CommandResult, error) {
	if len(req.Args) == 0 {
		return nil, fmt.Errorf("%s requires key", req.Command)
	}
	page, err := d.activePage(sess)
	if err != nil {
		return nil, err
	}
	storeName := storageObjectName(req.Command)
	value, err := page.Evaluate(fmt.Sprintf(`(key) => %s.getItem(key)`, storeName), req.Args[0])
	if err != nil {
		return nil, err
	}
	return &CommandResult{Session: name, Command: req.Command, Value: value, Page: pageInfo(page, sess.active)}, nil
}

func (d *daemonService) cmdStorageSet(name string, sess *liveSession, req CommandRequest) (*CommandResult, error) {
	if len(req.Args) < 2 {
		return nil, fmt.Errorf("%s requires key and value", req.Command)
	}
	page, err := d.activePage(sess)
	if err != nil {
		return nil, err
	}
	storeName := storageObjectName(req.Command)
	_, err = page.Evaluate(fmt.Sprintf(`([key, value]) => { %s.setItem(key, value); return value }`, storeName), []string{req.Args[0], req.Args[1]})
	if err != nil {
		return nil, err
	}
	return &CommandResult{Session: name, Command: req.Command, Value: req.Args[1], Page: pageInfo(page, sess.active)}, nil
}

func (d *daemonService) cmdStorageDelete(name string, sess *liveSession, req CommandRequest) (*CommandResult, error) {
	if len(req.Args) == 0 {
		return nil, fmt.Errorf("%s requires key", req.Command)
	}
	page, err := d.activePage(sess)
	if err != nil {
		return nil, err
	}
	storeName := storageObjectName(req.Command)
	_, err = page.Evaluate(fmt.Sprintf(`(key) => %s.removeItem(key)`, storeName), req.Args[0])
	if err != nil {
		return nil, err
	}
	return &CommandResult{Session: name, Command: req.Command, Message: "deleted", Page: pageInfo(page, sess.active)}, nil
}

func (d *daemonService) cmdStorageClear(name string, sess *liveSession, req CommandRequest) (*CommandResult, error) {
	page, err := d.activePage(sess)
	if err != nil {
		return nil, err
	}
	storeName := storageObjectName(req.Command)
	_, err = page.Evaluate(fmt.Sprintf(`() => %s.clear()`, storeName))
	if err != nil {
		return nil, err
	}
	return &CommandResult{Session: name, Command: req.Command, Message: "cleared", Page: pageInfo(page, sess.active)}, nil
}

func (d *daemonService) cmdRequests(name string, sess *liveSession) (*CommandResult, error) {
	return &CommandResult{Session: name, Command: "requests", Requests: append([]RequestInfo(nil), sess.requests...)}, nil
}

func (d *daemonService) cmdRequest(name string, sess *liveSession, req CommandRequest) (*CommandResult, error) {
	if len(req.Args) == 0 {
		return nil, fmt.Errorf("request requires index")
	}
	index, err := strconv.Atoi(req.Args[0])
	if err != nil {
		return nil, err
	}
	if index <= 0 || index > len(sess.requests) {
		return nil, fmt.Errorf("request index out of range")
	}
	return &CommandResult{Session: name, Command: "request", Value: sess.requests[index-1]}, nil
}

func (d *daemonService) cmdConsole(name string, sess *liveSession, req CommandRequest) (*CommandResult, error) {
	level := strings.ToLower(strings.TrimSpace(firstArg(req)))
	if level == "" || level == "info" {
		return &CommandResult{Session: name, Command: req.Command, Console: append([]ConsoleEntry(nil), sess.consoleLog...)}, nil
	}
	filtered := make([]ConsoleEntry, 0, len(sess.consoleLog))
	for _, item := range sess.consoleLog {
		if strings.EqualFold(item.Type, level) {
			filtered = append(filtered, item)
		}
	}
	return &CommandResult{Session: name, Command: req.Command, Console: filtered}, nil
}

func (d *daemonService) cmdDialog(name string, sess *liveSession, req CommandRequest, accept bool) (*CommandResult, error) {
	if sess.dialog == nil {
		return nil, fmt.Errorf("no dialog pending")
	}
	if accept {
		if len(req.Args) > 0 {
			if err := sess.dialog.Accept(req.Args[0]); err != nil {
				return nil, err
			}
		} else if err := sess.dialog.Accept(); err != nil {
			return nil, err
		}
	} else if err := sess.dialog.Dismiss(); err != nil {
		return nil, err
	}
	sess.dialog = nil
	return &CommandResult{Session: name, Command: req.Command, Message: "dialog handled"}, nil
}

func (d *daemonService) resolveLocator(page playwright.Page, sessionName, target string) (playwright.Locator, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, fmt.Errorf("target is required")
	}
	if strings.HasPrefix(target, "e") && snapshotExists(snapshotPath(d.opts.StateDir, sessionName)) {
		ref, err := resolveSnapshotRef(snapshotPath(d.opts.StateDir, sessionName), target)
		if err == nil {
			if ref.ID != "" {
				return page.Locator("#" + ref.ID), nil
			}
			if ref.Name != "" && ref.Tag != "" {
				return page.Locator(fmt.Sprintf(`%s[name="%s"]`, ref.Tag, ref.Name)), nil
			}
			if ref.Href != "" {
				return page.Locator(fmt.Sprintf(`[href="%s"]`, ref.Href)), nil
			}
			if ref.Text != "" {
				return page.GetByText(ref.Text), nil
			}
		}
	}
	if strings.HasPrefix(target, "#") || strings.HasPrefix(target, ".") || strings.ContainsAny(target, "[>:") {
		return page.Locator(target), nil
	}
	return page.GetByText(target), nil
}

func cookieInfo(value playwright.Cookie) CookieInfo {
	item := CookieInfo{
		Name:     value.Name,
		Value:    value.Value,
		Domain:   value.Domain,
		Path:     value.Path,
		Expires:  value.Expires,
		HTTPOnly: value.HttpOnly,
		Secure:   value.Secure,
	}
	if value.SameSite != nil {
		item.SameSite = sameSiteString(value.SameSite)
	}
	if value.PartitionKey != nil {
		item.Partition = *value.PartitionKey
	}
	return item
}

func firstArg(req CommandRequest) string {
	if len(req.Args) == 0 {
		return ""
	}
	return strings.TrimSpace(req.Args[0])
}

func storageObjectName(command string) string {
	switch {
	case strings.HasPrefix(command, "localstorage-"):
		return "window.localStorage"
	case strings.HasPrefix(command, "sessionstorage-"):
		return "window.sessionStorage"
	default:
		return "window.localStorage"
	}
}

func asStringMap(value any) map[string]any {
	if value == nil {
		return map[string]any{}
	}
	if out, ok := value.(map[string]any); ok {
		return out
	}
	return map[string]any{"value": value}
}
