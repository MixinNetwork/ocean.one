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
	Swap(p *Pool, in *Input) (*Output, error)
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

func (cpf *ConstantProductFormula) Swap(p *Pool, in *Input) (*Output, error) {
	if p.X.Sign() <= 0 || p.Y.Sign() <= 0 || in.Amount.Sign() <= 0 {
		return nil, ErrInvalidParams
	}

	k := p.X.Mul(p.Y)
	out := &Output{}

	if in.Quote {
		out.PriceInitial = cpf.Price(p.X, p.Y)
		p.Y = p.Y.Add(in.Amount)
		x := k.Div(p.Y).RoundCeil(decimals)
		out.Amount = p.X.Sub(x)
		p.X = x
		out.PriceFinal = cpf.Price(p.X, p.Y)
	} else {
		out.PriceInitial = cpf.Price(p.Y, p.X)
		p.X = p.X.Add(in.Amount)
		y := k.Div(p.X).RoundCeil(decimals)
		out.Amount = p.Y.Sub(y)
		p.Y = y
		out.PriceFinal = cpf.Price(p.Y, p.X)
	}
	slip := out.PriceFinal.Sub(out.PriceInitial).Abs().String()
	out.PriceSlippage = number.FromString(slip).Div(out.PriceInitial)
	out.PriceSlippage = out.PriceSlippage.RoundFloor(decimals)
	return out, nil
}
