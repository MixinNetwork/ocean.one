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
	typ  string
	data map[string]interface{}
}

type Queue struct {
	market   string
	sequence int64
	events   chan *Event
}

func NewQueue(ctx context.Context, market string) *Queue {
	return &Queue{
		market:   market,
		sequence: int64(0),
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
	e.data["event"] = e.typ
	e.data["sequence"] = queue.sequence
	queue.sequence = queue.sequence + 1

	switch e.typ {
	case EventTypeOrderOpen, EventTypeOrderMatch, EventTypeOrderCancel:
		key := fmt.Sprintf("%s-ORDER-EVENTS", queue.market)
		_, err := Redis(ctx).RPush(key, e.data).Result()
		return err
	default:
		key := fmt.Sprintf("%s-%s", queue.market, e.typ)
		_, err := Redis(ctx).Set(key, e.data, 0).Result()
		if err != nil {
			return err
		}
		key = fmt.Sprintf("%s-ORDER-EVENTS", queue.market)
		_, err = Redis(ctx).RPush(key, map[string]interface{}{
			"event":    "HEARTBEAT",
			"sequence": e.data["sequence"],
		}).Result()
		return err
	}
}

func (queue *Queue) AttachEvent(ctx context.Context, typ string, data map[string]interface{}) {
	queue.events <- &Event{
		typ:  typ,
		data: data,
	}
}
