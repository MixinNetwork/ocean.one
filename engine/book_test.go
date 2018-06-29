package engine

import (
	"context"
	"testing"
	"time"

	"github.com/MixinMessenger/go-number"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

type DummyTrade struct {
	Amount           number.Decimal
	TakerId          string
	TakerAmount      number.Decimal
	TakerFilledPrice uint64
	MakerId          string
	MakerAmount      number.Decimal
	MakerFilledPrice uint64
}

func TestBook(t *testing.T) {
	ctx := context.Background()
	assert := assert.New(t)

	matched := make([]*DummyTrade, 0)
	cancelled := make([]*Order, 0)
	book := NewBook(func(taker, maker *Order, amount number.Decimal) {
		matched = append(matched, &DummyTrade{
			Amount:           amount,
			TakerId:          taker.Id,
			TakerAmount:      taker.RemainingAmount,
			TakerFilledPrice: taker.FilledPrice,
			MakerId:          maker.Id,
			MakerAmount:      maker.RemainingAmount,
			MakerFilledPrice: maker.FilledPrice,
		})
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
		Price:           100,
		FilledPrice:     0,
		RemainingAmount: number.FromString("10"),
	}
	book.AttachOrderEvent(ctx, bo1_1, OrderActionCreate)

	id, _ = uuid.NewV4()
	bo1_2 := &Order{
		Id:              id.String(),
		Side:            PageSideBid,
		Type:            OrderTypeLimit,
		Price:           100,
		FilledPrice:     0,
		RemainingAmount: number.FromString("20"),
	}
	book.AttachOrderEvent(ctx, bo1_2, OrderActionCreate)

	id, _ = uuid.NewV4()
	bo1_3 := &Order{
		Id:              id.String(),
		Side:            PageSideBid,
		Type:            OrderTypeLimit,
		Price:           100,
		FilledPrice:     0,
		RemainingAmount: number.FromString("30"),
	}
	book.AttachOrderEvent(ctx, bo1_3, OrderActionCreate)
	time.Sleep(100 * time.Millisecond)
	assert.Equal("60", book.bids.entries[100].Amount.Persist())
	assert.Equal(3, book.bids.entries[100].list.Size())

	book.AttachOrderEvent(ctx, bo1_2, OrderActionCancel)
	time.Sleep(100 * time.Millisecond)
	assert.Len(cancelled, 1)
	assert.Equal(bo1_2.Id, cancelled[0].Id)
	assert.Equal("40", book.bids.entries[100].Amount.Persist())
	assert.Equal(2, book.bids.entries[100].list.Size())

	id, _ = uuid.NewV4()
	bo2_1 := &Order{
		Id:              id.String(),
		Side:            PageSideBid,
		Type:            OrderTypeLimit,
		Price:           200,
		FilledPrice:     0,
		RemainingAmount: number.FromString("10"),
	}
	book.AttachOrderEvent(ctx, bo2_1, OrderActionCreate)

	id, _ = uuid.NewV4()
	ao1_1 := &Order{
		Id:              id.String(),
		Side:            PageSideAsk,
		Type:            OrderTypeLimit,
		Price:           100,
		FilledPrice:     0,
		RemainingAmount: number.FromString("30"),
	}
	for i := 0; i < 2; i++ {
		book.AttachOrderEvent(ctx, ao1_1, OrderActionCreate)
		time.Sleep(100 * time.Millisecond)

		assert.Len(cancelled, 1)
		assert.Equal(bo1_2.Id, cancelled[0].Id)
		assert.Equal("20", book.bids.entries[100].Amount.Persist())
		assert.Equal(1, book.bids.entries[100].list.Size())
		assert.Len(book.asks.entries, 0)
		assert.Len(matched, 3)
		m0 := matched[0]
		assert.Equal("10", m0.Amount.Persist())
		assert.Equal(ao1_1.Id, m0.TakerId)
		assert.Equal("20", m0.TakerAmount.Persist())
		assert.Equal(uint64(200), m0.TakerFilledPrice)
		assert.Equal(bo2_1.Id, m0.MakerId)
		assert.Equal("0", m0.MakerAmount.Persist())
		assert.Equal(uint64(200), m0.MakerFilledPrice)
		m1 := matched[1]
		assert.Equal("10", m1.Amount.Persist())
		assert.Equal(ao1_1.Id, m1.TakerId)
		assert.Equal("10", m1.TakerAmount.Persist())
		assert.Equal(uint64(150), m1.TakerFilledPrice)
		assert.Equal(bo1_1.Id, m1.MakerId)
		assert.Equal("0", m1.MakerAmount.Persist())
		assert.Equal(uint64(100), m1.MakerFilledPrice)
		m2 := matched[2]
		assert.Equal("10", m2.Amount.Persist())
		assert.Equal(ao1_1.Id, m2.TakerId)
		assert.Equal("0", m2.TakerAmount.Persist())
		assert.Equal(uint64(133), m2.TakerFilledPrice)
		assert.Equal(bo1_3.Id, m2.MakerId)
		assert.Equal("20", m2.MakerAmount.Persist())
		assert.Equal(uint64(100), m2.MakerFilledPrice)
	}

	id, _ = uuid.NewV4()
	ao1_2 := &Order{
		Id:              id.String(),
		Side:            PageSideAsk,
		Type:            OrderTypeLimit,
		Price:           100,
		FilledPrice:     0,
		RemainingAmount: number.FromString("30"),
	}
	book.AttachOrderEvent(ctx, ao1_2, OrderActionCreate)
	time.Sleep(100 * time.Millisecond)

	assert.Len(cancelled, 1)
	assert.Equal(bo1_2.Id, cancelled[0].Id)
	assert.Equal("0", book.bids.entries[100].Amount.Persist())
	assert.Equal(0, book.bids.entries[100].list.Size())
	assert.Len(book.asks.entries, 1)
	assert.Len(matched, 4)
	m3 := matched[3]
	assert.Equal("20", m3.Amount.Persist())
	assert.Equal(ao1_2.Id, m3.TakerId)
	assert.Equal("10", m3.TakerAmount.Persist())
	assert.Equal(uint64(100), m3.TakerFilledPrice)
	assert.Equal(bo1_3.Id, m3.MakerId)
	assert.Equal("0", m3.MakerAmount.Persist())
	assert.Equal(uint64(100), m3.MakerFilledPrice)

	id, _ = uuid.NewV4()
	ao2_1 := &Order{
		Id:              id.String(),
		Side:            PageSideAsk,
		Type:            OrderTypeLimit,
		Price:           200,
		FilledPrice:     0,
		RemainingAmount: number.FromString("10"),
	}
	book.AttachOrderEvent(ctx, ao2_1, OrderActionCreate)

	id, _ = uuid.NewV4()
	bo2_2 := &Order{
		Id:              id.String(),
		Side:            PageSideBid,
		Type:            OrderTypeMarket,
		Price:           200,
		FilledPrice:     0,
		RemainingAmount: number.FromString("100"),
	}
	book.AttachOrderEvent(ctx, bo2_2, OrderActionCreate)
	time.Sleep(100 * time.Millisecond)

	assert.Equal("0", book.bids.entries[100].Amount.Persist())
	assert.Equal(0, book.bids.entries[100].list.Size())
	assert.Equal("0", book.bids.entries[200].Amount.Persist())
	assert.Equal(0, book.bids.entries[200].list.Size())
	assert.Equal("0", book.asks.entries[100].Amount.Persist())
	assert.Equal(0, book.asks.entries[100].list.Size())
	assert.Equal("0", book.asks.entries[200].Amount.Persist())
	assert.Equal(0, book.asks.entries[200].list.Size())
	assert.Len(cancelled, 2)
	assert.Equal(bo1_2.Id, cancelled[0].Id)
	assert.Equal(bo2_2.Id, cancelled[1].Id)
	assert.Equal("80", cancelled[1].RemainingAmount.Persist())
	assert.Equal("20", cancelled[1].FilledAmount.Persist())
	assert.Equal(uint64(150), cancelled[1].FilledPrice)
	assert.Len(matched, 6)
	m4 := matched[4]
	assert.Equal("10", m4.Amount.Persist())
	assert.Equal(bo2_2.Id, m4.TakerId)
	assert.Equal("90", m4.TakerAmount.Persist())
	assert.Equal(uint64(100), m4.TakerFilledPrice)
	assert.Equal(ao1_2.Id, m4.MakerId)
	assert.Equal("0", m4.MakerAmount.Persist())
	assert.Equal(uint64(100), m4.MakerFilledPrice)
	m5 := matched[5]
	assert.Equal("10", m5.Amount.Persist())
	assert.Equal(bo2_2.Id, m5.TakerId)
	assert.Equal("80", m5.TakerAmount.Persist())
	assert.Equal(uint64(150), m5.TakerFilledPrice)
	assert.Equal(ao2_1.Id, m5.MakerId)
	assert.Equal("0", m5.MakerAmount.Persist())
	assert.Equal(uint64(200), m5.MakerFilledPrice)
}
