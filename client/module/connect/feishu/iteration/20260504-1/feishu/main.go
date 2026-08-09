// 长连接接收回调 Demo：参考飞书开放平台「处理回调」文档中的 WebSocket 方式。
// 需在开发者后台将回调订阅方式设为「使用长连接接收回调」，并订阅「拉取链接预览数据」等回调。
package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher/callback"
	larkws "github.com/larksuite/oapi-sdk-go/v3/ws"
)

var (
	errMissingAppID     = errors.New("missing FEISHU_APP_ID")
	errMissingAppSecret = errors.New("missing FEISHU_APP_SECRET")
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.Lshortfile)

	appID := os.Getenv("FEISHU_APP_ID")
	appSecret := os.Getenv("FEISHU_APP_SECRET")
	if appID == "" {
		log.Fatal(errMissingAppID)
	}
	if appSecret == "" {
		log.Fatal(errMissingAppSecret)
	}

	// 长连接模式下 EventDispatcher 的两个参数必须为空字符串（与 HTTP Webhook 的验签密钥不同）
	eventHandler := dispatcher.NewEventDispatcher("", "").
		OnP2CardURLPreviewGet(handleURLPreview).
		OnP2CardActionTrigger(handleCardAction)

	// LogLevelDebug 会输出 SDK 内 WebSocket 建连、帧、分发等全部 Debug/Info 级别日志
	cli := larkws.NewClient(appID, appSecret,
		larkws.WithEventHandler(eventHandler),
		larkws.WithLogLevel(larkcore.LogLevelDebug),
	)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Println("starting Feishu WS client (long connection for callbacks)...")
	if err := cli.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("client stopped with error: %v", err)
	}
	log.Println("client shutdown complete")
}

// 拉取链接预览数据 url.preview.get：根据用户粘贴的 URL 返回内联预览或卡片（需在开放平台配置链接预览域名等）
func handleURLPreview(ctx context.Context, event *callback.URLPreviewGetEvent) (*callback.URLPreviewGetResponse, error) {
	_ = ctx
	log.Printf("[url.preview.get] request_id=%q uri=%q raw_http_body=%s", event.RequestId(), event.RequestURI, string(event.Body))
	log.Printf("[url.preview.get] parsed event: %s", larkcore.Prettify(event))

	previewURL := ""
	if event != nil && event.Event != nil && event.Event.Context != nil {
		previewURL = event.Event.Context.URL
	}

	// 内联预览示例：实际业务可替换为拉取 OG 元数据、调用内部 API 等（须在 3 秒内返回）
	resp := &callback.URLPreviewGetResponse{
		Inline: &callback.Inline{
			Title: previewURL,
			I18nTitle: map[string]string{
				"zh_cn": "链接预览（Demo）",
				"en_us": "Link preview (demo)",
			},
			URL: &callback.URL{
				Web: previewURL,
			},
		},
	}
	log.Printf("[url.preview.get] response: %s", larkcore.Prettify(resp))
	return resp, nil
}

// 卡片回传交互 card.action.trigger
func handleCardAction(ctx context.Context, event *callback.CardActionTriggerEvent) (*callback.CardActionTriggerResponse, error) {
	_ = ctx
	log.Printf("[card.action.trigger] request_id=%q uri=%q raw_http_body=%s", event.RequestId(), event.RequestURI, string(event.Body))
	log.Printf("[card.action.trigger] parsed event: %s", larkcore.Prettify(event))
	return nil, nil
}
