package feishusvc

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"testing"

	"connect/connectsvc"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

func TestTextRetryReusesIdempotencyKey(t *testing.T) {
	messageAPI := &stubMessageAPI{
		replyResps: []*larkim.ReplyMessageResp{
			{CodeError: larkcore.CodeError{Code: 230099}},
			{CodeError: larkcore.CodeError{Code: 230099}},
			{CodeError: larkcore.CodeError{Code: 0}, Data: &larkim.ReplyMessageRespData{MessageId: stringPtr("om_reply")}},
		},
	}
	sender := NewSender(
		stubMetaLoader{
			meta: &connectsvc.Meta{Name: DefaultName, ChatID: "oc_test_chat"},
			cfg:  Config{AppID: "app", AppSecret: "secret"},
		},
		log.New(io.Discard, "", 0),
		&bytes.Buffer{},
	)
	sender.apis = LarkAPISet{Message: messageAPI, Image: &stubImageAPI{}, File: &stubFileAPI{}}

	const idempotencyKey = "feishu-ack-idempotency-test"
	if _, err := sender.Send(context.Background(), SendInput{
		Message:        latestConnectRequestMessageJSON("om_origin"),
		Content:        "ack",
		IdempotencyKey: idempotencyKey,
	}); err != nil {
		t.Fatal(err)
	}
	if len(messageAPI.replyReqs) != maxSendRetries {
		t.Fatalf("reply attempts = %d, want %d", len(messageAPI.replyReqs), maxSendRetries)
	}
	for index, raw := range messageAPI.replyJSON {
		var body struct {
			UUID string `json:"uuid"`
		}
		if err := json.Unmarshal([]byte(raw), &body); err != nil {
			t.Fatalf("decode reply %d: %v; raw=%s", index+1, err, raw)
		}
		if got, want := body.UUID, childSendUUID(idempotencyKey, "text", 0); got != want {
			t.Fatalf("reply %d uuid = %q, want %q", index+1, got, want)
		}
	}
}

func TestChildSendUUIDIsStableAndSeparatesMessageKinds(t *testing.T) {
	const parent = "a28f7e3f-13cc-578b-b8b9-3e00cdff4f8d"
	text := childSendUUID(parent, "text", 0)
	if text == "" || text != childSendUUID(parent, "text", 0) {
		t.Fatalf("text child UUID must be stable: %q", text)
	}
	if image := childSendUUID(parent, "image", 0); image == text {
		t.Fatalf("image UUID must differ from text UUID: %q", image)
	}
	if nextImage := childSendUUID(parent, "image", 1); nextImage == childSendUUID(parent, "image", 0) {
		t.Fatalf("image UUID must differ by index: %q", nextImage)
	}
}
