package cache

import (
	"context"
	"fmt"
	"log"
	"time"
)

const (
	EventTypeOrderOpen   = "ORDER_OPEN"
	EventTypeOrderMatch  = "ORDER_MATCH"
	EventTypeOrderCancel = "ORDER_CANCEL"
)

type Event struct {
	market string                 `json:"market"`
	typ    string                 `json:"event"`
	seq    int64                  `json:"sequence"`
	data   map[string]interface{} `json:"data"`
	time   time.Time              `json:"timestamp"`
}

type Queue struct {
	market   string
	sequence int64
	events   chan *Event
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
	queue.sequence += 1
	e.seq = queue.sequence

	switch e.typ {
	case EventTypeOrderOpen, EventTypeOrderMatch, EventTypeOrderCancel:
		key := fmt.Sprintf("%s-ORDER-EVENTS", queue.market)
		_, err := Redis(ctx).RPush(key, e).Result()
		if err != nil {
			return err
		}
		_, err = Redis(ctx).Publish("ORDER-EVENTS", e).Result()
		return err
	case "BOOK-T0":
		key := fmt.Sprintf("%s-%s", queue.market, e.typ)
		_, err := Redis(ctx).Set(key, e, 0).Result()
		if err != nil {
			return err
		}
		key = fmt.Sprintf("%s-ORDER-EVENTS", queue.market)
		_, err = Redis(ctx).Del(key).Result()
		if err != nil {
			return err
		}
		data := map[string]interface{}{
			"event":    "HEARTBEAT",
			"sequence": e.seq,
		}
		_, err = Redis(ctx).RPush(key, data).Result()
		if err != nil {
			return err
		}
		_, err = Redis(ctx).Publish("ORDER-EVENTS", data).Result()
		return err
	}

	return fmt.Errorf("unsupported queue type %s", e.typ)
}

func (queue *Queue) AttachEvent(ctx context.Context, typ string, data map[string]interface{}) {
	queue.events <- &Event{
		market: queue.market,
		typ:    typ,
		data:   data,
		time:   time.Now().UTC(),
	}
}
