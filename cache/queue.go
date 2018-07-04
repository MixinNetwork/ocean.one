package cache

import (
	"context"
	"time"
)

type eventType struct {
	key  string
	data map[string]interface{}
}

type Queue struct {
	sequence int64
	events   chan *eventType
}

func NewQueue(ctx context.Context) *Queue {
	return &Queue{
		sequence: time.Now().Unix(),
	}
}

func (queue *Queue) Loop(ctx context.Context) {
	for {
		select {
		case e := <-queue.events:
			queue.sequence = queue.sequence + 1
			e.data["sequence"] = queue.sequence
		}
	}
}

func (queue *Queue) AttachEvent(ctx context.Context, key string, data map[string]interface{}) {
	queue.events <- &eventType{
		key:  key,
		data: data,
	}
}
