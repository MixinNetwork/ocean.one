package engine

import (
	"testing"

	"github.com/MixinNetwork/go-number"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func TestPageAsk(t *testing.T) {
	assert := assert.New(t)

	page := NewPage("ask")
	assert.Nil(page)

	page = NewPage(PageSideAsk)
	assert.Equal("ASK", page.Side)

	entries := page.List(0, false)
	assert.Len(entries, 0)

	id, _ := uuid.NewV4()
	o1 := &Order{
		Id:              id.String(),
		Side:            page.Side,
		Type:            OrderTypeLimit,
		Price:           number.NewInteger(10000, 2),
		RemainingAmount: number.NewInteger(10, 1),
		FilledAmount:    number.NewInteger(0, 1),
	}
	page.Put(o1)

	entries = page.List(0, false)
	assert.Len(entries, 1)
	e := entries[0]
	assert.Equal("1", e.Amount.Persist())
	assert.Equal(int64(10000), e.Price.Value())

	id, _ = uuid.NewV4()
	o3 := &Order{
		Id:              id.String(),
		Side:            page.Side,
		Type:            OrderTypeLimit,
		Price:           number.NewInteger(30000, 2),
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
		RemainingAmount: number.NewInteger(40, 1),
		FilledAmount:    number.NewInteger(0, 1),
	}
	page.Put(o2)

	entries = page.List(0, false)
	assert.Len(entries, 3)
	e = entries[0]
	assert.Equal("4", e.Amount.Persist())
	assert.Equal(int64(10000), e.Price.Value())
	e = entries[1]
	assert.Equal("4", e.Amount.Persist())
	assert.Equal(int64(20000), e.Price.Value())
	e = entries[2]
	assert.Equal("2", e.Amount.Persist())
	assert.Equal(int64(30000), e.Price.Value())

	page.Iterate(func(order *Order) (number.Integer, number.Integer, bool) {
		matchedAmount := number.NewInteger(5, 1)
		order.FilledAmount = order.FilledAmount.Add(matchedAmount)
		order.RemainingAmount = order.RemainingAmount.Sub(matchedAmount)
		return matchedAmount, number.NewInteger(5, 1), true
	})

	entries = page.List(0, false)
	assert.Len(entries, 3)
	e = entries[0]
	assert.Equal("3.5", e.Amount.Persist())
	assert.Equal(int64(10000), e.Price.Value())
	e = entries[1]
	assert.Equal("4", e.Amount.Persist())
	assert.Equal(int64(20000), e.Price.Value())
	e = entries[2]
	assert.Equal("2", e.Amount.Persist())
	assert.Equal(int64(30000), e.Price.Value())

	page.Remove(o1)
	entries = page.List(0, false)
	assert.Len(entries, 3)
	e = entries[0]
	assert.Equal("0.5", e.Amount.Persist())
	assert.Equal(int64(10000), e.Price.Value())
	e = entries[1]
	assert.Equal("4", e.Amount.Persist())
	assert.Equal(int64(20000), e.Price.Value())
	e = entries[2]
	assert.Equal("2", e.Amount.Persist())
	assert.Equal(int64(30000), e.Price.Value())

	page.Remove(o3)
	entries = page.List(0, false)
	assert.Len(entries, 3)
	e = entries[0]
	assert.Equal("0.5", e.Amount.Persist())
	assert.Equal(int64(10000), e.Price.Value())
	e = entries[1]
	assert.Equal("4", e.Amount.Persist())
	assert.Equal(int64(20000), e.Price.Value())
	e = entries[2]
	assert.Equal("0", e.Amount.Persist())
	assert.Equal(int64(30000), e.Price.Value())

	page.Iterate(func(order *Order) (number.Integer, number.Integer, bool) {
		matchedAmount := number.NewInteger(5, 1)
		order.FilledAmount = order.FilledAmount.Add(matchedAmount)
		order.RemainingAmount = order.RemainingAmount.Sub(matchedAmount)
		return matchedAmount, number.NewInteger(5, 1), order.Price.Decimal().IntPart() == 200
	})

	entries = page.List(0, false)
	assert.Len(entries, 3)
	e = entries[0]
	assert.Equal("0", e.Amount.Persist())
	assert.Equal(int64(10000), e.Price.Value())
	e = entries[1]
	assert.Equal("3.5", e.Amount.Persist())
	assert.Equal(int64(20000), e.Price.Value())
	e = entries[2]
	assert.Equal("0", e.Amount.Persist())
	assert.Equal(int64(30000), e.Price.Value())

	page.Remove(o2)
	entries = page.List(0, false)
	assert.Len(entries, 3)
	e = entries[0]
	assert.Equal("0", e.Amount.Persist())
	assert.Equal(int64(10000), e.Price.Value())
	e = entries[1]
	assert.Equal("0", e.Amount.Persist())
	assert.Equal(int64(20000), e.Price.Value())
	e = entries[2]
	assert.Equal("0", e.Amount.Persist())
	assert.Equal(int64(30000), e.Price.Value())
}

func TestPageBid(t *testing.T) {
	assert := assert.New(t)

	page := NewPage("bid")
	assert.Nil(page)

	page = NewPage(PageSideBid)
	assert.Equal("BID", page.Side)

	entries := page.List(0, false)
	assert.Len(entries, 0)

	id, _ := uuid.NewV4()
	o1 := &Order{
		Id:              id.String(),
		Side:            page.Side,
		Type:            OrderTypeLimit,
		Price:           number.NewInteger(10000, 2),
		RemainingAmount: number.NewInteger(0, 1),
		FilledAmount:    number.NewInteger(0, 1),
		RemainingFunds:  number.NewInteger(100000, 3),
		FilledFunds:     number.NewInteger(0, 3),
	}
	page.Put(o1)

	entries = page.List(0, false)
	assert.Len(entries, 1)
	e := entries[0]
	assert.Equal("1", e.Amount.Persist())
	assert.Equal("100", e.Funds.Persist())
	assert.Equal(int64(10000), e.Price.Value())

	id, _ = uuid.NewV4()
	o3 := &Order{
		Id:              id.String(),
		Side:            page.Side,
		Type:            OrderTypeLimit,
		Price:           number.NewInteger(30000, 2),
		RemainingAmount: number.NewInteger(0, 1),
		FilledAmount:    number.NewInteger(0, 1),
		RemainingFunds:  number.NewInteger(600000, 3),
		FilledFunds:     number.NewInteger(0, 3),
	}
	page.Put(o3)

	id, _ = uuid.NewV4()
	o1 = &Order{
		Id:              id.String(),
		Side:            page.Side,
		Type:            OrderTypeLimit,
		Price:           number.NewInteger(10000, 2),
		RemainingAmount: number.NewInteger(0, 1),
		FilledAmount:    number.NewInteger(0, 1),
		RemainingFunds:  number.NewInteger(300000, 3),
		FilledFunds:     number.NewInteger(0, 3),
	}
	page.Put(o1)

	id, _ = uuid.NewV4()
	o2 := &Order{
		Id:              id.String(),
		Side:            page.Side,
		Type:            OrderTypeLimit,
		Price:           number.NewInteger(20000, 2),
		RemainingAmount: number.NewInteger(0, 1),
		FilledAmount:    number.NewInteger(0, 1),
		RemainingFunds:  number.NewInteger(800000, 3),
		FilledFunds:     number.NewInteger(0, 3),
	}
	page.Put(o2)

	entries = page.List(0, false)
	assert.Len(entries, 3)
	e = entries[0]
	assert.Equal("2", e.Amount.Persist())
	assert.Equal("600", e.Funds.Persist())
	assert.Equal(int64(30000), e.Price.Value())
	e = entries[1]
	assert.Equal("4", e.Amount.Persist())
	assert.Equal("800", e.Funds.Persist())
	assert.Equal(int64(20000), e.Price.Value())
	e = entries[2]
	assert.Equal("4", e.Amount.Persist())
	assert.Equal("400", e.Funds.Persist())
	assert.Equal(int64(10000), e.Price.Value())

	page.Iterate(func(order *Order) (number.Integer, number.Integer, bool) {
		matchedFunds := number.NewInteger(50000, 3)
		order.FilledFunds = order.FilledFunds.Add(matchedFunds)
		order.RemainingFunds = order.RemainingFunds.Sub(matchedFunds)
		return number.NewInteger(5, 1), matchedFunds, true
	})

	entries = page.List(0, false)
	assert.Len(entries, 3)
	e = entries[0]
	assert.Equal("1.83333333333333333333333333333333", e.Amount.Persist())
	assert.Equal("550", e.Funds.Persist())
	assert.Equal(int64(30000), e.Price.Value())
	e = entries[1]
	assert.Equal("4", e.Amount.Persist())
	assert.Equal("800", e.Funds.Persist())
	assert.Equal(int64(20000), e.Price.Value())
	e = entries[2]
	assert.Equal("4", e.Amount.Persist())
	assert.Equal("400", e.Funds.Persist())
	assert.Equal(int64(10000), e.Price.Value())

	page.Remove(o1)
	entries = page.List(0, false)
	assert.Len(entries, 3)
	e = entries[0]
	assert.Equal("1.83333333333333333333333333333333", e.Amount.Persist())
	assert.Equal("550", e.Funds.Persist())
	assert.Equal(int64(30000), e.Price.Value())
	e = entries[1]
	assert.Equal("4", e.Amount.Persist())
	assert.Equal("800", e.Funds.Persist())
	assert.Equal(int64(20000), e.Price.Value())
	e = entries[2]
	assert.Equal("1", e.Amount.Persist())
	assert.Equal("100", e.Funds.Persist())
	assert.Equal(int64(10000), e.Price.Value())

	page.Remove(o3)
	entries = page.List(0, false)
	assert.Len(entries, 3)
	e = entries[0]
	assert.Equal("0", e.Amount.Persist())
	assert.Equal("0", e.Funds.Persist())
	assert.Equal(int64(30000), e.Price.Value())
	e = entries[1]
	assert.Equal("4", e.Amount.Persist())
	assert.Equal("800", e.Funds.Persist())
	assert.Equal(int64(20000), e.Price.Value())
	e = entries[2]
	assert.Equal("1", e.Amount.Persist())
	assert.Equal("100", e.Funds.Persist())
	assert.Equal(int64(10000), e.Price.Value())

	page.Iterate(func(order *Order) (number.Integer, number.Integer, bool) {
		matchedFunds := number.NewInteger(50000, 3)
		order.FilledFunds = order.FilledFunds.Add(matchedFunds)
		order.RemainingFunds = order.RemainingFunds.Sub(matchedFunds)
		return number.NewInteger(5, 1), matchedFunds, order.Price.Decimal().IntPart() == 100
	})

	entries = page.List(0, false)
	assert.Len(entries, 3)
	e = entries[0]
	assert.Equal("0", e.Amount.Persist())
	assert.Equal("0", e.Funds.Persist())
	assert.Equal(int64(30000), e.Price.Value())
	e = entries[1]
	assert.Equal("3.75", e.Amount.Persist())
	assert.Equal("750", e.Funds.Persist())
	assert.Equal(int64(20000), e.Price.Value())
	e = entries[2]
	assert.Equal("0.5", e.Amount.Persist())
	assert.Equal("50", e.Funds.Persist())
	assert.Equal(int64(10000), e.Price.Value())

	page.Remove(o2)
	entries = page.List(0, false)
	assert.Len(entries, 3)
	e = entries[0]
	assert.Equal("0", e.Amount.Persist())
	assert.Equal("0", e.Funds.Persist())
	assert.Equal(int64(30000), e.Price.Value())
	e = entries[1]
	assert.Equal("0", e.Amount.Persist())
	assert.Equal("0", e.Funds.Persist())
	assert.Equal(int64(20000), e.Price.Value())
	e = entries[2]
	assert.Equal("0.5", e.Amount.Persist())
	assert.Equal("50", e.Funds.Persist())
	assert.Equal(int64(10000), e.Price.Value())
}
