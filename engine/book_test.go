package engine

import (
	"context"
	"testing"
	"time"

	"github.com/MixinNetwork/go-number"
	"github.com/MixinNetwork/ocean.one/cache"
	"github.com/MixinNetwork/ocean.one/config"
	"github.com/go-redis/redis"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

type DummyTrade struct {
	Amount           number.Integer
	TakerId          string
	TakerAmount      number.Integer
	TakerFunds       number.Integer
	TakerFilledPrice number.Integer
	MakerId          string
	MakerAmount      number.Integer
	MakerFunds       number.Integer
	MakerFilledPrice number.Integer
}

func TestBook(t *testing.T) {
	ctx := context.Background()
	ctx = testSetupRedis(ctx)
	assert := assert.New(t)

	matched := make([]*DummyTrade, 0)
	cancelled := make([]*Order, 0)
	book := NewBook(ctx, "market", func(taker, maker *Order, amount number.Integer) string {
		matched = append(matched, &DummyTrade{
			Amount:           amount,
			TakerId:          taker.Id,
			TakerAmount:      taker.RemainingAmount,
			TakerFunds:       taker.RemainingFunds,
			TakerFilledPrice: taker.FilledFunds.Div(taker.FilledAmount),
			MakerId:          maker.Id,
			MakerAmount:      maker.RemainingAmount,
			MakerFunds:       maker.RemainingFunds,
			MakerFilledPrice: maker.FilledFunds.Div(maker.FilledAmount),
		})
		return "TRADE-ID"
	}, func(order *Order) {
		cancelled = append(cancelled, order)
	})
	assert.NotNil(book)
	go book.Run(ctx)

	id, _ := uuid.NewV4()
	bo1_1 := &Order{
		Id:              id.String(),
		Side:            PageSideBid,
		Type:            OrderTypeLimit,
		Price:           number.NewInteger(10000, 2),
		RemainingAmount: number.NewInteger(0, 1),
		FilledAmount:    number.NewInteger(0, 1),
		RemainingFunds:  number.NewInteger(1000000, 3),
		FilledFunds:     number.NewInteger(0, 3),
	}
	book.AttachOrderEvent(ctx, bo1_1, OrderActionCreate)

	id, _ = uuid.NewV4()
	bo1_2 := &Order{
		Id:              id.String(),
		Side:            PageSideBid,
		Type:            OrderTypeLimit,
		Price:           number.NewInteger(10000, 2),
		RemainingAmount: number.NewInteger(0, 1),
		FilledAmount:    number.NewInteger(0, 1),
		RemainingFunds:  number.NewInteger(2000000, 3),
		FilledFunds:     number.NewInteger(0, 3),
	}
	book.AttachOrderEvent(ctx, bo1_2, OrderActionCreate)

	id, _ = uuid.NewV4()
	bo1_3 := &Order{
		Id:              id.String(),
		Side:            PageSideBid,
		Type:            OrderTypeLimit,
		Price:           number.NewInteger(10000, 2),
		RemainingAmount: number.NewInteger(0, 1),
		FilledAmount:    number.NewInteger(0, 1),
		RemainingFunds:  number.NewInteger(3000000, 3),
		FilledFunds:     number.NewInteger(0, 3),
	}
	book.AttachOrderEvent(ctx, bo1_3, OrderActionCreate)
	time.Sleep(100 * time.Millisecond)
	assert.Equal("6000", book.bids.entries[10000].Funds.Persist())
	assert.Equal(3, book.bids.entries[10000].list.Size())

	book.AttachOrderEvent(ctx, bo1_2, OrderActionCancel)
	time.Sleep(100 * time.Millisecond)
	assert.Len(cancelled, 1)
	assert.Equal(bo1_2.Id, cancelled[0].Id)
	assert.Equal("4000", book.bids.entries[10000].Funds.Persist())
	assert.Equal(2, book.bids.entries[10000].list.Size())

	id, _ = uuid.NewV4()
	bo2_1 := &Order{
		Id:              id.String(),
		Side:            PageSideBid,
		Type:            OrderTypeLimit,
		Price:           number.NewInteger(20000, 2),
		RemainingAmount: number.NewInteger(0, 1),
		FilledAmount:    number.NewInteger(0, 1),
		RemainingFunds:  number.NewInteger(2000000, 3),
		FilledFunds:     number.NewInteger(0, 3),
	}
	book.AttachOrderEvent(ctx, bo2_1, OrderActionCreate)

	id, _ = uuid.NewV4()
	ao1_1 := &Order{
		Id:              id.String(),
		Side:            PageSideAsk,
		Type:            OrderTypeLimit,
		Price:           number.NewInteger(10000, 2),
		RemainingAmount: number.NewInteger(300, 1),
		FilledAmount:    number.NewInteger(0, 1),
		RemainingFunds:  number.NewInteger(0, 3),
		FilledFunds:     number.NewInteger(0, 3),
	}
	for i := 0; i < 2; i++ {
		book.AttachOrderEvent(ctx, ao1_1, OrderActionCreate)
		time.Sleep(100 * time.Millisecond)

		assert.Len(cancelled, 1)
		assert.Equal(bo1_2.Id, cancelled[0].Id)
		assert.Equal("2000", book.bids.entries[10000].Funds.Persist())
		assert.Equal(1, book.bids.entries[10000].list.Size())
		assert.Len(book.asks.entries, 0)
		assert.Len(matched, 3)
		m0 := matched[0]
		assert.Equal("10", m0.Amount.Persist())
		assert.Equal(ao1_1.Id, m0.TakerId)
		assert.Equal("20", m0.TakerAmount.Persist())
		assert.Equal("200", m0.TakerFilledPrice.Persist())
		assert.Equal(bo2_1.Id, m0.MakerId)
		assert.Equal("0", m0.MakerAmount.Persist())
		assert.Equal("200", m0.MakerFilledPrice.Persist())
		m1 := matched[1]
		assert.Equal("10", m1.Amount.Persist())
		assert.Equal(ao1_1.Id, m1.TakerId)
		assert.Equal("10", m1.TakerAmount.Persist())
		assert.Equal("150", m1.TakerFilledPrice.Persist())
		assert.Equal(bo1_1.Id, m1.MakerId)
		assert.Equal("0", m1.MakerAmount.Persist())
		assert.Equal("100", m1.MakerFilledPrice.Persist())
		m2 := matched[2]
		assert.Equal("10", m2.Amount.Persist())
		assert.Equal(ao1_1.Id, m2.TakerId)
		assert.Equal("0", m2.TakerAmount.Persist())
		assert.Equal("133.33", m2.TakerFilledPrice.Persist())
		assert.Equal(bo1_3.Id, m2.MakerId)
		assert.Equal("2000", m2.MakerFunds.Persist())
		assert.Equal("100", m2.MakerFilledPrice.Persist())
	}

	id, _ = uuid.NewV4()
	ao1_2 := &Order{
		Id:              id.String(),
		Side:            PageSideAsk,
		Type:            OrderTypeLimit,
		Price:           number.NewInteger(10000, 2),
		RemainingAmount: number.NewInteger(300, 1),
		FilledAmount:    number.NewInteger(0, 1),
		RemainingFunds:  number.NewInteger(0, 3),
		FilledFunds:     number.NewInteger(0, 3),
	}
	book.AttachOrderEvent(ctx, ao1_2, OrderActionCreate)
	time.Sleep(100 * time.Millisecond)

	assert.Len(cancelled, 1)
	assert.Equal(bo1_2.Id, cancelled[0].Id)
	assert.Equal("0", book.bids.entries[10000].Funds.Persist())
	assert.Equal(0, book.bids.entries[10000].list.Size())
	assert.Len(book.asks.entries, 1)
	assert.Len(matched, 4)
	m3 := matched[3]
	assert.Equal("20", m3.Amount.Persist())
	assert.Equal(ao1_2.Id, m3.TakerId)
	assert.Equal("10", m3.TakerAmount.Persist())
	assert.Equal("100", m3.TakerFilledPrice.Persist())
	assert.Equal(bo1_3.Id, m3.MakerId)
	assert.Equal("0", m3.MakerAmount.Persist())
	assert.Equal("100", m3.MakerFilledPrice.Persist())

	id, _ = uuid.NewV4()
	ao2_1 := &Order{
		Id:              id.String(),
		Side:            PageSideAsk,
		Type:            OrderTypeLimit,
		Price:           number.NewInteger(20000, 2),
		RemainingAmount: number.NewInteger(100, 1),
		FilledAmount:    number.NewInteger(0, 1),
		RemainingFunds:  number.NewInteger(0, 3),
		FilledFunds:     number.NewInteger(0, 3),
	}
	book.AttachOrderEvent(ctx, ao2_1, OrderActionCreate)

	id, _ = uuid.NewV4()
	bo2_2 := &Order{
		Id:              id.String(),
		Side:            PageSideBid,
		Type:            OrderTypeMarket,
		Price:           number.NewInteger(0, 2),
		RemainingAmount: number.NewInteger(0, 1),
		FilledAmount:    number.NewInteger(0, 1),
		RemainingFunds:  number.NewInteger(20000000, 3),
		FilledFunds:     number.NewInteger(0, 3),
	}
	book.AttachOrderEvent(ctx, bo2_2, OrderActionCreate)
	time.Sleep(100 * time.Millisecond)

	assert.Equal("0", book.bids.entries[10000].Funds.Persist())
	assert.Equal(0, book.bids.entries[10000].list.Size())
	assert.Equal("0", book.bids.entries[20000].Funds.Persist())
	assert.Equal(0, book.bids.entries[20000].list.Size())
	assert.Equal("0", book.asks.entries[10000].Amount.Persist())
	assert.Equal(0, book.asks.entries[10000].list.Size())
	assert.Equal("0", book.asks.entries[20000].Amount.Persist())
	assert.Equal(0, book.asks.entries[20000].list.Size())
	assert.Len(cancelled, 2)
	assert.Equal(bo1_2.Id, cancelled[0].Id)
	assert.Equal(bo2_2.Id, cancelled[1].Id)
	assert.Equal("17000", cancelled[1].RemainingFunds.Persist())
	assert.Equal("20", cancelled[1].FilledAmount.Persist())
	assert.Equal("150", cancelled[1].FilledFunds.Div(cancelled[1].FilledAmount).Persist())
	assert.Len(matched, 6)
	m4 := matched[4]
	assert.Equal("10", m4.Amount.Persist())
	assert.Equal(bo2_2.Id, m4.TakerId)
	assert.Equal("19000", m4.TakerFunds.Persist())
	assert.Equal("100", m4.TakerFilledPrice.Persist())
	assert.Equal(ao1_2.Id, m4.MakerId)
	assert.Equal("0", m4.MakerAmount.Persist())
	assert.Equal("100", m4.MakerFilledPrice.Persist())
	m5 := matched[5]
	assert.Equal("10", m5.Amount.Persist())
	assert.Equal(bo2_2.Id, m5.TakerId)
	assert.Equal("17000", m5.TakerFunds.Persist())
	assert.Equal("150", m5.TakerFilledPrice.Persist())
	assert.Equal(ao2_1.Id, m5.MakerId)
	assert.Equal("0", m5.MakerAmount.Persist())
	assert.Equal("200", m5.MakerFilledPrice.Persist())
}

func testSetupRedis(ctx context.Context) context.Context {
	redisClient := redis.NewClient(&redis.Options{
		Addr:         config.RedisEngineCacheAddress,
		DB:           config.RedisEngineCacheDatabase,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolTimeout:  4 * time.Second,
		IdleTimeout:  60 * time.Second,
		PoolSize:     1024,
	})

	return cache.SetupRedis(ctx, redisClient)
}
