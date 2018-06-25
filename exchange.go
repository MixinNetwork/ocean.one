package main

import (
	"context"
	"log"
	"time"

	"github.com/MixinMessenger/bot-api-go-client"
	"github.com/MixinMessenger/ocean.one/config"
	"github.com/MixinMessenger/ocean.one/engine"
	"github.com/MixinMessenger/ocean.one/persistence"
	"github.com/ugorji/go/codec"
)

type Exchange struct {
	books     map[string]*engine.Book
	codec     codec.Handle
	snapshots map[string]bool
}

func NewExchange() *Exchange {
	return &Exchange{
		codec:     new(codec.MsgpackHandle),
		books:     make(map[string]*engine.Book),
		snapshots: make(map[string]bool),
	}
}

func (ex *Exchange) Run(ctx context.Context) {
	go ex.PollMixinMessages(ctx)
	go ex.PollMixinNetwork(ctx)
	ex.PollOrderActions(ctx)
}

func (ex *Exchange) PollOrderActions(ctx context.Context) {
}

func (ex *Exchange) PollMixinNetwork(ctx context.Context) {
	checkpoint, limit := persistence.ReadActionCheckpoint(ctx).UTC(), 500
	for {
		snapshots, err := ex.requestMixinNetwork(ctx, checkpoint, limit)
		if err != nil {
			log.Println("PollMixinNetwork ERROR", err)
			time.Sleep(1 * time.Second)
			continue
		}
		for _, s := range snapshots {
			if ex.snapshots[s.SnapshotId] {
				continue
			}
			log.Println(s)
			err := ex.processSnapshot(ctx, s)
			if err != nil {
				log.Println("PollMixinNetwork processSnapshot ERROR", err)
				break
			}
			checkpoint = s.CreatedAt
			ex.snapshots[s.SnapshotId] = true
		}
		if len(snapshots) < limit {
			time.Sleep(1 * time.Second)
			continue
		}
	}
}

func (ex *Exchange) PollMixinMessages(ctx context.Context) {
	bot.Loop(ctx, ex, config.ClientId, config.SessionId, config.SessionKey)
}

func (ex *Exchange) OnMessage(ctx context.Context, mc *bot.MessageContext, msg bot.MessageView, userId string) error {
	log.Println(msg, userId)
	return nil
}
