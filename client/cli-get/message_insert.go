package main

import (
	"strings"
	"time"

	messageinsert "cli-get/messageinsert"
)

const messageInsertPublishLimit = 5

type messageInsertPublishItem struct {
	Tid     string `json:"tid"`
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
	db, err := getDataDB()
	if err != nil {
		return err
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
	_, err = messageinsert.MarkPublished(db, chatID, tids, time.Now())
	return err
}
