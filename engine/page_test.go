package engine

import (
	"testing"

	"github.com/MixinMessenger/go-number"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func TestPage(t *testing.T) {
	assert := assert.New(t)

	page := NewPage("ask")
	assert.Nil(page)

	page = NewPage(PageSideAsk)
	assert.Equal("ASK", page.Side)

	entries := page.List(0)
	assert.Len(entries, 0)

	id, _ := uuid.NewV4()
	o1 := &Order{
		Id:              id.String(),
		Side:            page.Side,
		Type:            OrderTypeLimit,
		Price:           100,
		FilledPrice:     0,
		RemainingAmount: number.FromString("1"),
	}
	page.Put(o1)

	entries = page.List(0)
	assert.Len(entries, 1)
	e := entries[0]
	assert.Equal("1", e.Amount.Persist())
	assert.Equal(uint64(100), e.Price)

	id, _ = uuid.NewV4()
	o3 := &Order{
		Id:              id.String(),
		Side:            page.Side,
		Type:            OrderTypeLimit,
		Price:           300,
		FilledPrice:     0,
		RemainingAmount: number.FromString("2"),
	}
	page.Put(o3)

	id, _ = uuid.NewV4()
	o1 = &Order{
		Id:              id.String(),
		Side:            page.Side,
		Type:            OrderTypeLimit,
		Price:           100,
		FilledPrice:     0,
		RemainingAmount: number.FromString("3"),
	}
	page.Put(o1)

	id, _ = uuid.NewV4()
	o2 := &Order{
		Id:              id.String(),
		Side:            page.Side,
		Type:            OrderTypeLimit,
		Price:           200,
		FilledPrice:     0,
		RemainingAmount: number.FromString("4"),
	}
	page.Put(o2)

	entries = page.List(0)
	assert.Len(entries, 3)
	e = entries[0]
	assert.Equal("4", e.Amount.Persist())
	assert.Equal(uint64(100), e.Price)
	e = entries[1]
	assert.Equal("4", e.Amount.Persist())
	assert.Equal(uint64(200), e.Price)
	e = entries[2]
	assert.Equal("2", e.Amount.Persist())
	assert.Equal(uint64(300), e.Price)

	page.Iterate(func(order *Order) (number.Decimal, bool) {
		matchedAmount := number.FromString("0.5")
		order.FilledAmount = order.FilledAmount.Add(matchedAmount)
		order.RemainingAmount = order.RemainingAmount.Sub(matchedAmount)
		return matchedAmount, true
	})

	entries = page.List(0)
	assert.Len(entries, 3)
	e = entries[0]
	assert.Equal("3.5", e.Amount.Persist())
	assert.Equal(uint64(100), e.Price)
	e = entries[1]
	assert.Equal("4", e.Amount.Persist())
	assert.Equal(uint64(200), e.Price)
	e = entries[2]
	assert.Equal("2", e.Amount.Persist())
	assert.Equal(uint64(300), e.Price)

	page.Remove(o1)
	entries = page.List(0)
	assert.Len(entries, 3)
	e = entries[0]
	assert.Equal("0.5", e.Amount.Persist())
	assert.Equal(uint64(100), e.Price)
	e = entries[1]
	assert.Equal("4", e.Amount.Persist())
	assert.Equal(uint64(200), e.Price)
	e = entries[2]
	assert.Equal("2", e.Amount.Persist())
	assert.Equal(uint64(300), e.Price)

	page.Remove(o3)
	entries = page.List(0)
	assert.Len(entries, 3)
	e = entries[0]
	assert.Equal("0.5", e.Amount.Persist())
	assert.Equal(uint64(100), e.Price)
	e = entries[1]
	assert.Equal("4", e.Amount.Persist())
	assert.Equal(uint64(200), e.Price)
	e = entries[2]
	assert.Equal("0", e.Amount.Persist())
	assert.Equal(uint64(300), e.Price)

	page.Iterate(func(order *Order) (number.Decimal, bool) {
		matchedAmount := number.FromString("0.5")
		order.FilledAmount = order.FilledAmount.Add(matchedAmount)
		order.RemainingAmount = order.RemainingAmount.Sub(matchedAmount)
		return matchedAmount, order.Price == 200
	})

	entries = page.List(0)
	assert.Len(entries, 3)
	e = entries[0]
	assert.Equal("0", e.Amount.Persist())
	assert.Equal(uint64(100), e.Price)
	e = entries[1]
	assert.Equal("3.5", e.Amount.Persist())
	assert.Equal(uint64(200), e.Price)
	e = entries[2]
	assert.Equal("0", e.Amount.Persist())
	assert.Equal(uint64(300), e.Price)

	page.Remove(o2)
	entries = page.List(0)
	assert.Len(entries, 3)
	e = entries[0]
	assert.Equal("0", e.Amount.Persist())
	assert.Equal(uint64(100), e.Price)
	e = entries[1]
	assert.Equal("0", e.Amount.Persist())
	assert.Equal(uint64(200), e.Price)
	e = entries[2]
	assert.Equal("0", e.Amount.Persist())
	assert.Equal(uint64(300), e.Price)
}
