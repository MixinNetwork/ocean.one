package cache

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/gorilla/websocket"
	"github.com/satori/go.uuid"
)

const (
	readWait       = 10 * time.Second
	writeWait      = 10 * time.Second
	pingPeriod     = 5 * time.Second
	maxMessageSize = 1024
)

type BlazeMessage struct {
	Id     string                 `json:"id"`
	Action string                 `json:"action"`
	Params map[string]interface{} `json:"params,omitempty"`
	Data   interface{}            `json:"data,omitempty"`
	Error  string                 `json:"error,omitempty"`
}

type Client struct {
	hub            *Hub
	conn           *websocket.Conn
	cid            string
	receive        chan *BlazeMessage
	hubChannel     chan *EventResponse
	clientResponse chan []byte
	hubResponse    chan []byte
	cancel         context.CancelFunc
}

func NewClient(ctx context.Context, hub *Hub, conn *websocket.Conn, id string, cancel context.CancelFunc) (*Client, error) {
	client := &Client{
		hub:            hub,
		conn:           conn,
		cid:            id,
		receive:        make(chan *BlazeMessage, 64),
		hubChannel:     make(chan *EventResponse, 81920),
		hubResponse:    make(chan []byte, 1024),
		clientResponse: make(chan []byte, 64),
		cancel:         cancel,
	}
	return client, nil
}

func (client *Client) WritePump(ctx context.Context) error {
	defer client.conn.Close()
	go client.loopHubChannel(ctx)

	pingTicker := time.NewTicker(pingPeriod)
	defer pingTicker.Stop()

	for {
		select {
		case msg := <-client.clientResponse:
			err := writeGzipToConn(ctx, client.conn, msg)
			if err != nil {
				return err
			}
		case msg := <-client.hubResponse:
			err := writeGzipToConn(ctx, client.conn, msg)
			if err != nil {
				return err
			}
		case <-pingTicker.C:
			err := client.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err != nil {
				return err
			}
			err = client.conn.WriteMessage(websocket.PingMessage, []byte{})
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (client *Client) loopHubChannel(ctx context.Context) error {
	defer client.conn.Close()

	for {
		select {
		case e := <-client.hubChannel:
			switch e.Source {
			case "LIST_PENDING_EVENTS":
				time.Sleep(100 * time.Millisecond)
				err := client.sendPendingEvents(ctx, e.Channel)
				if err != nil {
					return err
				}
			case "EMIT_EVENT":
				data, _ := json.Marshal(BlazeMessage{
					Id:     uuid.Nil.String(),
					Action: "EMIT_EVENT",
					Data: map[string]interface{}{
						"source": e.Source,
						"event":  e.Event,
					},
				})
				err := client.pipeHubResponse(ctx, data)
				if err != nil {
					return err
				}
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (client *Client) sendPendingEvents(ctx context.Context, channel string) error {
	events, err := ListPendingEvents(ctx, channel)
	if err != nil {
		return err
	}
	for _, e := range events {
		data, _ := json.Marshal(BlazeMessage{
			Id:     uuid.Nil.String(),
			Action: "EMIT_EVENT",
			Data: map[string]interface{}{
				"source": "LIST_PENDING_EVENTS",
				"event":  e,
			},
		})
		err = client.pipeHubResponse(ctx, data)
		if err != nil {
			return err
		}
	}
	return nil
}

func writeGzipToConn(ctx context.Context, conn *websocket.Conn, msg []byte) error {
	err := conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err != nil {
		return err
	}
	wsWriter, err := conn.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return err
	}
	gzWriter, err := gzip.NewWriterLevel(wsWriter, 3)
	if err != nil {
		return err
	}
	if _, err := gzWriter.Write(msg); err != nil {
		return err
	}

	if err := gzWriter.Close(); err != nil {
		return err
	}
	if err := wsWriter.Close(); err != nil {
		return err
	}
	return nil
}

func (client *Client) ReadPump(ctx context.Context) error {
	defer client.conn.Close()
	go client.loopReceiveMessage(ctx)

	client.conn.SetReadLimit(maxMessageSize)
	client.conn.SetReadDeadline(time.Now().Add(readWait))
	client.conn.SetPongHandler(func(string) error {
		return client.conn.SetReadDeadline(time.Now().Add(readWait))
	})

	for {
		err := client.conn.SetReadDeadline(time.Now().Add(readWait))
		if err != nil {
			return err
		}
		messageType, wsReader, err := client.conn.NextReader()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway) {
				return err
			}
			log.Printf("EXPECTED CLOSE %s %s\n", client.cid, err.Error())
			return nil
		}
		if messageType != websocket.BinaryMessage {
			err = client.error(ctx, "message type must be binary")
		} else {
			err = client.parseMessage(ctx, wsReader)
		}
		if err != nil {
			return err
		}
	}
}

func (client *Client) loopReceiveMessage(ctx context.Context) error {
	defer client.conn.Close()

	for {
		select {
		case msg := <-client.receive:
			err := client.handleMessage(ctx, msg)
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (client *Client) handleMessage(ctx context.Context, msg *BlazeMessage) error {
	var err error
	market := fmt.Sprint(msg.Params["market"])
	switch msg.Action {
	case "SUBSCRIBE_BOOK":
		err = client.hub.SubscribePendingEvents(ctx, market, client.cid)
	case "UNSUBSCRIBE_BOOK":
		err = client.hub.UnsubscribePendingEvents(ctx, market, client.cid)
	case "SUBSCRIBE_TICKER":
	case "UNSUBSCRIBE_TICKER":
	}
	return client.ack(ctx, msg.Action, msg.Id, err)
}

func (client *Client) parseMessage(ctx context.Context, wsReader io.Reader) error {
	var message BlazeMessage
	gzReader, err := gzip.NewReader(wsReader)
	if err != nil {
		return client.error(ctx, err.Error())
	}
	defer gzReader.Close()
	if err = json.NewDecoder(gzReader).Decode(&message); err != nil {
		return client.error(ctx, err.Error())
	}

	select {
	case client.receive <- &message:
	case <-time.After(writeWait):
		return errors.New("timeout to pipe receive message")
	}
	return nil
}

func (client *Client) error(ctx context.Context, err string) error {
	data, _ := json.Marshal(BlazeMessage{
		Id:     uuid.Nil.String(),
		Action: "ERROR",
		Error:  err,
	})
	return client.pipeClientResponse(ctx, data)
}

func (client *Client) ack(ctx context.Context, action, id string, err error) error {
	msg := &BlazeMessage{Action: action, Id: id}
	if err != nil {
		msg.Error = err.Error()
	} else {
		msg.Data = map[string]string{"status": "received"}
	}
	data, _ := json.Marshal(msg)
	return client.pipeClientResponse(ctx, data)
}

func (client *Client) pipeClientResponse(ctx context.Context, msg []byte) error {
	select {
	case client.clientResponse <- msg:
	case <-time.After(writeWait):
		return errors.New("timeout to pipe client response")
	}
	return nil
}

func (client *Client) pipeHubResponse(ctx context.Context, msg []byte) error {
	select {
	case client.hubResponse <- msg:
	case <-time.After(writeWait):
		return errors.New("timeout to pipe hub response")
	}
	return nil
}

func (client *Client) pipeHubChannel(ctx context.Context, msg *EventResponse) error {
	select {
	case client.hubChannel <- msg:
	case <-time.After(writeWait):
		return errors.New("timeout to pipe hub channel")
	}
	return nil
}
