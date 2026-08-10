package launchsplash

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const DefaultDuration = 5 * time.Second

var (
	ErrUnsupportedPlatform = errors.New("launch splash is only supported on macOS")
	launchSplashLookPathFn = exec.LookPath
	launchSplashCommandFn  = exec.Command
	launchSplashGOOSFn     = func() string { return runtime.GOOS }
)

type Config struct {
	LogoPath string
	Duration time.Duration
}

func Start(cfg Config) error {
	cmd, err := buildCommand(cfg)
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() {
		_ = cmd.Wait()
	}()
	return nil
}

func Run(cfg Config) error {
	cmd, err := buildCommand(cfg)
	if err != nil {
		return err
	}
	output, err := cmd.CombinedOutput()
	if err == nil {
		return nil
	}
	text := strings.TrimSpace(string(output))
	if text == "" {
		return err
	}
	return fmt.Errorf("%w: %s", err, text)
}

func buildCommand(cfg Config) (*exec.Cmd, error) {
	if launchSplashGOOSFn() != "darwin" {
		return nil, ErrUnsupportedPlatform
	}
	normalized, err := normalizeConfig(cfg)
	if err != nil {
		return nil, err
	}
	osascriptPath, err := launchSplashLookPathFn("osascript")
	if err != nil {
		return nil, fmt.Errorf("locate osascript: %w", err)
	}
	cmd := launchSplashCommandFn(osascriptPath, "-l", "JavaScript", "-e", jxaScript)
	cmd.Env = append(os.Environ(),
		"DEEPRIGHT_SPLASH_LOGO="+normalized.LogoPath,
		fmt.Sprintf("DEEPRIGHT_SPLASH_DURATION_MS=%d", normalized.Duration.Milliseconds()),
	)
	return cmd, nil
}

func normalizeConfig(cfg Config) (Config, error) {
	cfg.LogoPath = strings.TrimSpace(cfg.LogoPath)
	if cfg.LogoPath == "" {
		return Config{}, errors.New("launch splash logo path is empty")
	}
	info, err := os.Stat(cfg.LogoPath)
	if err != nil {
		return Config{}, fmt.Errorf("stat launch splash logo: %w", err)
	}
	if info.IsDir() {
		return Config{}, errors.New("launch splash logo path points to a directory")
	}
	if abs, err := filepathAbsFn(cfg.LogoPath); err == nil {
		cfg.LogoPath = abs
	}
	if cfg.Duration <= 0 {
		cfg.Duration = DefaultDuration
	}
	return cfg, nil
}

var filepathAbsFn = filepath.Abs

const jxaScript = `
ObjC.import("Cocoa");

function env(name, fallbackValue) {
  var value = $.NSProcessInfo.processInfo.environment.objectForKey(name);
  if (value === undefined || value === null) {
    return fallbackValue;
  }
  var text = ObjC.unwrap(value);
  if (text === undefined || text === null || String(text) === "") {
    return fallbackValue;
  }
  return String(text);
}

function clamp(value, minValue, maxValue) {
  return Math.max(minValue, Math.min(maxValue, value));
}

function easeOutCubic(progress) {
  return 1 - Math.pow(1 - progress, 3);
}

var logoPath = env("DEEPRIGHT_SPLASH_LOGO", "");
if (!logoPath) {
  throw new Error("missing DEEPRIGHT_SPLASH_LOGO");
}

var durationMs = parseInt(env("DEEPRIGHT_SPLASH_DURATION_MS", "5000"), 10);
if (!(durationMs > 0)) {
  durationMs = 5000;
}

var app = $.NSApplication.sharedApplication;
app.setActivationPolicy($.NSApplicationActivationPolicyAccessory);

var image = $.NSImage.alloc.initWithContentsOfFile($(logoPath));
if (!image) {
  throw new Error("failed to load splash logo");
}
var imageSize = image.size;
var imageWidth = Number(imageSize.width);
var imageHeight = Number(imageSize.height);
if (!(imageWidth > 0) || !(imageHeight > 0)) {
  throw new Error("invalid splash logo dimensions");
}

var screen = $.NSScreen.mainScreen;
if (!screen && $.NSScreen.screens.count > 0) {
  screen = $.NSScreen.screens.objectAtIndex(0);
}
if (!screen) {
  throw new Error("failed to resolve main screen");
}

var visibleFrame = screen.visibleFrame;
var baseMaxWidth = clamp(Number(visibleFrame.size.width) * 0.18, 180, 320);
var baseMaxHeight = clamp(Number(visibleFrame.size.height) * 0.18, 140, 260);
var maxWidth = Math.min(Number(visibleFrame.size.width) * 0.8, baseMaxWidth * 3);
var maxHeight = Math.min(Number(visibleFrame.size.height) * 0.8, baseMaxHeight * 3);
var imageScale = Math.min(maxWidth / imageWidth, maxHeight / imageHeight);
if (!(imageScale > 0)) {
  imageScale = 1;
}
var rootWidth = Math.max(1, imageWidth * imageScale);
var rootHeight = Math.max(1, imageHeight * imageScale);
var startScale = 0.22;
var endScale = 1.0;

var window = $.NSWindow.alloc.initWithContentRectStyleMaskBackingDefer(
  $.NSMakeRect(0, 0, rootWidth, rootHeight),
  $.NSWindowStyleMaskBorderless,
  $.NSBackingStoreBuffered,
  false
);
window.setOpaque(false);
window.setBackgroundColor($.NSColor.clearColor);
window.setIgnoresMouseEvents(true);
window.setLevel($.NSStatusWindowLevel);
window.setCollectionBehavior(
  $.NSWindowCollectionBehaviorCanJoinAllSpaces |
  $.NSWindowCollectionBehaviorTransient |
  $.NSWindowCollectionBehaviorFullScreenAuxiliary
);

var rootView = $.NSView.alloc.initWithFrame($.NSMakeRect(0, 0, rootWidth, rootHeight));
window.setContentView(rootView);

var imageView = $.NSImageView.alloc.initWithFrame($.NSMakeRect(0, 0, rootWidth, rootHeight));
imageView.setImage(image);
imageView.setImageScaling($.NSImageScaleProportionallyUpOrDown);
rootView.addSubview(imageView);

function centeredFrame(scale) {
  var width = rootWidth * scale;
  var height = rootHeight * scale;
  return $.NSMakeRect(
    Number(visibleFrame.origin.x) + (Number(visibleFrame.size.width) - width) / 2,
    Number(visibleFrame.origin.y) + (Number(visibleFrame.size.height) - height) / 2,
    width,
    height
  );
}

function applyLayout(scale, alpha) {
  var currentRootWidth = rootWidth * scale;
  var currentRootHeight = rootHeight * scale;
  var frame = centeredFrame(scale);
  window.setFrameDisplay(frame, true);
  rootView.setFrame($.NSMakeRect(0, 0, currentRootWidth, currentRootHeight));
  imageView.setFrame($.NSMakeRect(0, 0, currentRootWidth, currentRootHeight));
  window.setAlphaValue(alpha);
}

applyLayout(startScale, 1);
window.orderFront(null);
app.activateIgnoringOtherApps(true);

var fps = 24;
var frameInterval = 1 / fps;
var totalFrames = Math.max(1, Math.round(durationMs / (frameInterval * 1000)));
for (var frameIndex = 0; frameIndex <= totalFrames; frameIndex++) {
  var raw = frameIndex / totalFrames;
  var eased = easeOutCubic(raw);
  var scale = startScale + (endScale - startScale) * eased;
  var fadeProgress = raw < 0.76 ? 0 : (raw - 0.76) / 0.24;
  var alpha = 1 - Math.pow(clamp(fadeProgress, 0, 1), 1.8);
  applyLayout(scale, clamp(alpha, 0, 1));
  $.NSRunLoop.currentRunLoop.runUntilDate($.NSDate.dateWithTimeIntervalSinceNow(frameInterval));
}

window.orderOut(null);
`
