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
		Price:           number.NewInteger(10000, 2),
		FilledPrice:     number.NewInteger(0, 2),
		RemainingAmount: number.NewInteger(10, 1),
		FilledAmount:    number.NewInteger(0, 1),
	}
	page.Put(o1)

	entries = page.List(0)
	assert.Len(entries, 1)
	e := entries[0]
	assert.Equal("1", e.Amount.Persist())
	assert.Equal(int64(10000), e.Price)

	id, _ = uuid.NewV4()
	o3 := &Order{
		Id:              id.String(),
		Side:            page.Side,
		Type:            OrderTypeLimit,
		Price:           number.NewInteger(30000, 2),
		FilledPrice:     number.NewInteger(0, 2),
		RemainingAmount: number.NewInteger(20, 1),
		FilledAmount:    number.NewInteger(0, 1),
	}
	page.Put(o3)

	id, _ = uuid.NewV4()
	o1 = &Order{
		Id:              id.String(),
		Side:            page.Side,
		Type:            OrderTypeLimit,
		Price:           number.NewInteger(10000, 2),
		FilledPrice:     number.NewInteger(0, 2),
		RemainingAmount: number.NewInteger(30, 1),
		FilledAmount:    number.NewInteger(0, 1),
	}
	page.Put(o1)

	id, _ = uuid.NewV4()
	o2 := &Order{
		Id:              id.String(),
		Side:            page.Side,
		Type:            OrderTypeLimit,
		Price:           number.NewInteger(20000, 2),
		FilledPrice:     number.NewInteger(0, 2),
		RemainingAmount: number.NewInteger(40, 1),
		FilledAmount:    number.NewInteger(0, 1),
	}
	page.Put(o2)

	entries = page.List(0)
	assert.Len(entries, 3)
	e = entries[0]
	assert.Equal("4", e.Amount.Persist())
	assert.Equal(int64(10000), e.Price)
	e = entries[1]
	assert.Equal("4", e.Amount.Persist())
	assert.Equal(int64(20000), e.Price)
	e = entries[2]
	assert.Equal("2", e.Amount.Persist())
	assert.Equal(int64(30000), e.Price)

	page.Iterate(func(order *Order) (number.Integer, bool) {
		matchedAmount := number.NewInteger(5, 1)
		order.FilledAmount = order.FilledAmount.Add(matchedAmount)
		order.RemainingAmount = order.RemainingAmount.Sub(matchedAmount)
		return matchedAmount, true
	})

	entries = page.List(0)
	assert.Len(entries, 3)
	e = entries[0]
	assert.Equal("3.5", e.Amount.Persist())
	assert.Equal(int64(10000), e.Price)
	e = entries[1]
	assert.Equal("4", e.Amount.Persist())
	assert.Equal(int64(20000), e.Price)
	e = entries[2]
	assert.Equal("2", e.Amount.Persist())
	assert.Equal(int64(30000), e.Price)

	page.Remove(o1)
	entries = page.List(0)
	assert.Len(entries, 3)
	e = entries[0]
	assert.Equal("0.5", e.Amount.Persist())
	assert.Equal(int64(10000), e.Price)
	e = entries[1]
	assert.Equal("4", e.Amount.Persist())
	assert.Equal(int64(20000), e.Price)
	e = entries[2]
	assert.Equal("2", e.Amount.Persist())
	assert.Equal(int64(30000), e.Price)

	page.Remove(o3)
	entries = page.List(0)
	assert.Len(entries, 3)
	e = entries[0]
	assert.Equal("0.5", e.Amount.Persist())
	assert.Equal(int64(10000), e.Price)
	e = entries[1]
	assert.Equal("4", e.Amount.Persist())
	assert.Equal(int64(20000), e.Price)
	e = entries[2]
	assert.Equal("0", e.Amount.Persist())
	assert.Equal(int64(30000), e.Price)

	page.Iterate(func(order *Order) (number.Integer, bool) {
		matchedAmount := number.NewInteger(5, 1)
		order.FilledAmount = order.FilledAmount.Add(matchedAmount)
		order.RemainingAmount = order.RemainingAmount.Sub(matchedAmount)
		return matchedAmount, order.Price.Decimal().IntPart() == 200
	})

	entries = page.List(0)
	assert.Len(entries, 3)
	e = entries[0]
	assert.Equal("0", e.Amount.Persist())
	assert.Equal(int64(10000), e.Price)
	e = entries[1]
	assert.Equal("3.5", e.Amount.Persist())
	assert.Equal(int64(20000), e.Price)
	e = entries[2]
	assert.Equal("0", e.Amount.Persist())
	assert.Equal(int64(30000), e.Price)

	page.Remove(o2)
	entries = page.List(0)
	assert.Len(entries, 3)
	e = entries[0]
	assert.Equal("0", e.Amount.Persist())
	assert.Equal(int64(10000), e.Price)
	e = entries[1]
	assert.Equal("0", e.Amount.Persist())
	assert.Equal(int64(20000), e.Price)
	e = entries[2]
	assert.Equal("0", e.Amount.Persist())
	assert.Equal(int64(30000), e.Price)
}
