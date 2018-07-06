package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

const registerWait = 10 * time.Second

type Subscription struct {
	channel string
	cid     string
}

type Member struct {
	client   *Client
	channels map[string]time.Time
}

type EventResponse struct {
	Channel string
	Source  string
	Event   *Event
}

type Hub struct {
	register    chan *Client
	unregister  chan *Client
	subscribe   chan *Subscription
	unsubscribe chan *Subscription
	response    chan *EventResponse
}

func NewHub() *Hub {
	return &Hub{
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		subscribe:   make(chan *Subscription, 64),
		unsubscribe: make(chan *Subscription, 64),
		response:    make(chan *EventResponse, 8192),
	}
}

func (hub *Hub) Run(ctx context.Context) error {
	go hub.loopPendingEvents(ctx)
	members := make(map[string]*Member)
	channels := make(map[string]map[string]time.Time)

	for {
		select {
		case client := <-hub.register:
			if _, found := members[client.cid]; !found {
				members[client.cid] = &Member{client, make(map[string]time.Time)}
			}
		case client := <-hub.unregister:
			if member, found := members[client.cid]; found {
				delete(members, client.cid)
				for channel, _ := range member.channels {
					delete(channels[channel], client.cid)
				}
				client.cancel()
			}
		case sub := <-hub.subscribe:
			if _, found := channels[sub.channel]; !found {
				channels[sub.channel] = make(map[string]time.Time)
			}
			if member, found := members[sub.cid]; found {
				if _, found := member.channels[sub.channel]; found {
					continue
				}
				channels[sub.channel][sub.cid] = time.Now()
				member.channels[sub.channel] = time.Now()
				err := member.client.pipeHubChannel(ctx, &EventResponse{
					Channel: sub.channel,
					Source:  "LIST_PENDING_EVENTS",
				})
				if err != nil {
					log.Println("hub subscribe", err)
					member.client.cancel()
				}
			}
		case sub := <-hub.unsubscribe:
			if member, found := members[sub.cid]; found {
				delete(member.channels, sub.channel)
			}
			if channel, found := channels[sub.channel]; found {
				delete(channel, sub.cid)
			}
		case resp := <-hub.response:
			clients, found := channels[resp.Channel]
			if !found {
				continue
			}
			for cid, _ := range clients {
				member, found := members[cid]
				if !found {
					continue
				}
				err := member.client.pipeHubChannel(ctx, resp)
				if err != nil {
					log.Println("hub response", err)
					member.client.cancel()
				}
			}
		}
	}
}

func (hub *Hub) Register(ctx context.Context, client *Client) error {
	select {
	case hub.register <- client:
	case <-time.After(registerWait):
		return fmt.Errorf("timeout to register client %s", client.cid)
	}
	return nil
}

func (hub *Hub) Unregister(client *Client) error {
	select {
	case hub.unregister <- client:
	case <-time.After(registerWait):
		return fmt.Errorf("timeout to unregister client %s", client.cid)
	}
	return nil
}

func (hub *Hub) SubscribePendingEvents(ctx context.Context, market, cid string) error {
	select {
	case hub.subscribe <- &Subscription{market + "-ORDER-EVENTS", cid}:
	case <-time.After(registerWait):
		return fmt.Errorf("timeout to subscribe pending events %s %s", market, cid)
	}
	return nil
}

func (hub *Hub) UnsubscribePendingEvents(ctx context.Context, market, cid string) error {
	select {
	case hub.unsubscribe <- &Subscription{market + "-ORDER-EVENTS", cid}:
	case <-time.After(registerWait):
		return fmt.Errorf("timeout to unsubscribe pending events %s %s", market, cid)
	}
	return nil
}

func (hub *Hub) loopPendingEvents(ctx context.Context) {
	pubsub := Redis(ctx).Subscribe("ORDER-EVENTS")

	for {
		msg, err := pubsub.ReceiveMessage()
		if err != nil {
			log.Println("loopPendingEvents", err)
			time.Sleep(300 * time.Millisecond)
			continue
		}
		var event Event
		err = json.Unmarshal([]byte(msg.Payload), &event)
		if err != nil {
			log.Panicln(err)
		}
		hub.response <- &EventResponse{event.Market + "-ORDER-EVENTS", "EMIT_EVENT", &event}
	}
}
