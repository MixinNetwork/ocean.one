package swap

import (
	"github.com/MixinNetwork/go-number"
)

var (
	zero     = number.FromString("0")
	decimals = int32(8)
)

type Formula interface {
	Price(x, y number.Decimal) number.Decimal
	Swap(x, y number.Decimal, in number.Decimal) (*Output, error)
}

type Input struct {
	Amount number.Decimal
	Quote  bool
}

type Output struct {
	Amount        number.Decimal
	PriceInitial  number.Decimal
	PriceFinal    number.Decimal
	PriceSlippage number.Decimal
}

type ConstantProductFormula struct{}

func (cpf *ConstantProductFormula) Price(x, y number.Decimal) number.Decimal {
	x = x.RoundFloor(decimals)
	y = y.RoundFloor(decimals)
	return x.Div(y).RoundFloor(decimals)
}

func (cpf *ConstantProductFormula) Swap(x, y number.Decimal, in number.Decimal) (*Output, error) {
	if x.Sign() <= 0 || y.Sign() <= 0 || in.Sign() <= 0 {
		return nil, ErrInvalidParams
	}

	k := x.Mul(y)
	out := &Output{}

	out.PriceInitial = cpf.Price(x, y)
	pY := y.Add(in)
	out.Amount = x.Sub(k.Div(pY).RoundCeil(decimals))
	pX := x.Sub(out.Amount)
	out.PriceFinal = cpf.Price(pX, pY)

	slip := out.PriceFinal.Sub(out.PriceInitial).Abs().String()
	out.PriceSlippage = number.FromString(slip).Div(out.PriceInitial)
	out.PriceSlippage = out.PriceSlippage.RoundFloor(decimals)
	return out, nil
}
