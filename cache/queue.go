package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis"
)

const (
	EventTypeOrderOpen   = "ORDER_OPEN"
	EventTypeOrderMatch  = "ORDER_MATCH"
	EventTypeOrderCancel = "ORDER_CANCEL"
)

type Event struct {
	Market    string                 `json:"market"`
	Type      string                 `json:"event"`
	Sequence  int64                  `json:"sequence"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

type Queue struct {
	market   string
	sequence int64
	events   chan *Event
}

func ListPendingEvents(ctx context.Context, key string) ([]*Event, error) {
	var events []*Event
	slice, err := Redis(ctx).LRange(key, 0, -1).Result()
	if err != nil {
		return nil, err
	}
	for _, s := range slice {
		var e Event
		err = json.Unmarshal([]byte(s), &e)
		if err != nil {
			return nil, err
		}
		events = append(events, &e)
	}
	return events, nil
}

func NewQueue(ctx context.Context, market string) *Queue {
	return &Queue{
		market:   market,
		sequence: time.Now().Unix(),
		events:   make(chan *Event, 8192),
	}
}

func (queue *Queue) Loop(ctx context.Context) {
	for {
		select {
		case e := <-queue.events:
			err := queue.handleEvent(ctx, e)
			if err != nil {
				log.Println("cache queue loop error", err)
				time.Sleep(1 * time.Second)
			}
		}
	}
}

func (queue *Queue) handleEvent(ctx context.Context, e *Event) error {
	e.Sequence = queue.sequence
	queue.sequence = queue.sequence + 1
	data, err := json.Marshal(e)
	if err != nil {
		log.Panicln(err)
	}

	key := queue.market + "-ORDER-EVENTS"
	switch e.Type {
	case EventTypeOrderOpen, EventTypeOrderMatch, EventTypeOrderCancel:
		_, err := Redis(ctx).RPush(key, data).Result()
		if err != nil {
			return err
		}
	case "BOOK-T0":
		_, err := Redis(ctx).Pipelined(func(pipe redis.Pipeliner) error {
			pipe.Del(key)
			pipe.RPush(key, data)
			return nil
		})
		if err != nil {
			return err
		}
		data, _ = json.Marshal(Event{
			Market:    queue.market,
			Type:      "HEARTBEAT",
			Sequence:  e.Sequence,
			Timestamp: e.Timestamp,
		})
	default:
		return fmt.Errorf("unsupported queue type %s", e.Type)
	}

	_, err = Redis(ctx).Publish("ORDER-EVENTS", data).Result()
	return err

}

func (queue *Queue) AttachEvent(ctx context.Context, typ string, data map[string]interface{}) {
	queue.events <- &Event{
		Market:    queue.market,
		Type:      typ,
		Data:      data,
		Timestamp: time.Now().UTC(),
	}
}
