package swap

import (
	"math"

	"github.com/MixinNetwork/go-number"
)

var (
	liquiditySlippage = number.FromString("0.001")
	liquidityMaximum  = number.FromString("1000000000")
	liquidityMinimum  = number.FromString("0.001")
)

type Pair struct {
	X number.Decimal
	Y number.Decimal
}

type Fee struct {
	PoolRate  number.Decimal
	ExtraRate number.Decimal
}

type Pool struct {
	X         number.Decimal
	Y         number.Decimal
	Liquidity number.Decimal
	Fee       *Fee
	f         Formula
}

func (p *Pool) ProvideLiquidity(x, y number.Decimal) (number.Decimal, error) {
	x = x.RoundFloor(decimals)
	y = y.RoundFloor(decimals)
	if x.Sign() <= 0 || y.Sign() <= 0 {
		return zero, ErrInvalidParams
	}

	if p.X.Sign() == 0 && p.Y.Sign() == 0 {
		liquidity := number.FromFloat(math.Sqrt(x.Mul(y).Float64()))
		liquidity = liquidity.RoundFloor(decimals)
		p.Liquidity = liquidity.Add(liquidityMinimum)
		p.X = x
		p.Y = y
		return liquidity, nil
	}

	ip := p.f.Price(x, y)
	pp := p.f.Price(p.X, p.Y)
	if ip.Sub(pp).Div(pp).Cmp(liquiditySlippage) > 0 {
		return zero, ErrInvalidLiquidityPrice
	}

	liquidity := p.Liquidity.Mul(x).Div(p.X)
	if lY := p.Liquidity.Mul(y).Div(p.Y); lY.Cmp(liquidity) < 0 {
		liquidity = lY
	}
	liquidity = liquidity.RoundFloor(decimals)
	p.Liquidity = p.Liquidity.Add(liquidity)
	p.X = p.X.Add(x)
	p.Y = p.Y.Add(y)

	return liquidity, nil
}

func (p *Pool) RemoveLiquidity(liquidity number.Decimal) (*Pair, error) {
	liquidity = liquidity.RoundFloor(decimals)
	if p.Liquidity.Cmp(liquidity.Add(liquidityMinimum)) < 0 {
		return nil, ErrInvalidParams
	}

	pair := &Pair{
		X: p.X.Mul(liquidity).Div(p.Liquidity).RoundFloor(decimals),
		Y: p.Y.Mul(liquidity).Div(p.Liquidity).RoundFloor(decimals),
	}
	p.X = p.X.Sub(pair.X)
	p.Y = p.Y.Sub(pair.Y)
	p.Liquidity = p.Liquidity.Sub(liquidity)

	if p.X.Sign() == 0 || p.Y.Sign() == 0 || p.Liquidity.Sign() == 0 {
		return nil, ErrLiquidityEmpty
	}
	return pair, nil
}

func (p *Pool) Swap(amount number.Decimal, quote bool) (*Output, number.Decimal, error) {
	amount = amount.RoundFloor(decimals)
	if amount.Sign() <= 0 {
		return nil, zero, ErrInvalidParams
	}

	poolFee := amount.Mul(p.Fee.PoolRate)
	extraFee := amount.Mul(p.Fee.ExtraRate)
	totalFee := poolFee.Add(extraFee)
	amount = amount.Sub(totalFee).RoundFloor(decimals)
	if amount.Sign() <= 0 {
		return nil, zero, ErrInvalidParams
	}

	if quote {
		out, err := p.f.Swap(p.X, p.Y, amount)
		if err != nil {
			return nil, zero, err
		}
		p.X = p.X.Sub(out.Amount)
		p.Y = p.Y.Add(amount).Add(poolFee)
		return out, extraFee, nil
	} else {
		out, err := p.f.Swap(p.Y, p.X, amount)
		if err != nil {
			return nil, zero, err
		}
		p.X = p.X.Add(amount).Add(poolFee)
		p.Y = p.Y.Sub(out.Amount)
		return out, extraFee, nil
	}
}
