package swap

import (
	"github.com/MixinNetwork/go-number"
)

var (
	liquiditySlippage = number.FromString("0.001")
)

type Pair struct {
	X number.Decimal
	Y number.Decimal
}

type Pool struct {
	X         number.Decimal
	Y         number.Decimal
	Liquidity number.Decimal
	FeeRate   number.Decimal
	f         Formula
}

func (p *Pool) ProvideLiquidity(x, y number.Decimal) (number.Decimal, error) {
	x = x.RoundFloor(decimals)
	y = y.RoundFloor(decimals)
	if x.Sign() <= 0 || y.Sign() <= 0 {
		return zero, ErrInvalidParams
	}

	ip := p.f.Price(x, y)
	pp := p.f.Price(p.X, p.Y)
	if ip.Sub(pp).Div(pp).Cmp(liquiditySlippage) > 0 {
		return zero, ErrInvalidLiquidityPrice
	}

	p.X = p.X.Add(x)
	p.Y = p.Y.Add(y)
	liquidity := p.Liquidity.Mul(y.Div(p.Y))
	p.Liquidity = p.Liquidity.Add(liquidity)

	return liquidity, nil
}

func (p *Pool) RemoveLiquidity(liquidity number.Decimal) (*Pair, error) {
	liquidity = liquidity.RoundFloor(decimals)
	if p.Liquidity.Cmp(liquidity) < 0 {
		return nil, ErrInvalidParams
	}

	share := liquidity.Div(p.Liquidity).RoundFloor(decimals)
	pair := &Pair{
		X: p.X.Mul(share),
		Y: p.Y.Mul(share),
	}
	p.X = p.X.Sub(pair.X)
	p.Y = p.Y.Sub(pair.Y)
	p.Liquidity = p.Liquidity.Sub(liquidity)

	return pair, nil
}

func (p *Pool) Swap(amount number.Decimal, quote bool) (*Output, error) {
	amount = amount.RoundFloor(decimals)
	if amount.Sign() <= 0 {
		return nil, ErrInvalidParams
	}

	fee := amount.Mul(p.FeeRate)
	amount = amount.Sub(fee).RoundFloor(decimals)
	if amount.Sign() <= 0 {
		return nil, ErrInvalidParams
	}

	out, err := p.f.Swap(p, &Input{amount, quote})
	if err != nil {
		return nil, err
	}
	if quote {
		p.Y = p.Y.Add(fee)
	} else {
		p.X = p.X.Add(fee)
	}
	return out, nil
}
