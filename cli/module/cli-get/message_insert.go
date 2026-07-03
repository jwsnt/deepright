package main

import (
	"strings"
	"time"

	messageinsert "cli-get/messageinsert"
)

const messageInsertPublishLimit = 5

type messageInsertPublishItem struct {
	Mid     string `json:"mid"`
	Message string `json:"message"`
}

func loadPendingMessageInsertPublishItems(chatID string, limit int) ([]messageInsertPublishItem, error) {
	if strings.TrimSpace(chatID) == "" {
		return nil, nil
	}
	db, err := getDataDB()
	if err != nil {
		return nil, err
	}
	items, err := messageinsert.ListPending(db, chatID, limit)
	if err != nil {
		return nil, err
	}
	result := make([]messageInsertPublishItem, 0, len(items))
	for _, item := range items {
		result = append(result, messageInsertPublishItem{
			Mid:     strings.TrimSpace(item.Mid),
			Message: strings.TrimSpace(item.Message),
		})
	}
	return result, nil
}

func markUploadedMessageInsertPublishItems(chatID string, items []messageInsertPublishItem) error {
	if strings.TrimSpace(chatID) == "" || len(items) == 0 {
		return nil
	}
	db, err := getDataDB()
	if err != nil {
		return err
	}
	mids := make([]string, 0, len(items))
	for _, item := range items {
		if mid := strings.TrimSpace(item.Mid); mid != "" {
			mids = append(mids, mid)
		}
	}
	if len(mids) == 0 {
		return nil
	}
	_, err = messageinsert.MarkUploaded(db, chatID, mids, time.Now())
	return err
}
