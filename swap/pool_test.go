package swap

import (
	"testing"

	"github.com/MixinNetwork/go-number"
	"github.com/stretchr/testify/assert"
)

func TestPool(t *testing.T) {
	assert := assert.New(t)

	pool := &Pool{
		Fee: &Fee{
			PoolRate:  number.FromString("0.002"),
			ExtraRate: number.FromString("0.001"),
		},
		f: &ConstantProductFormula{},
	}

	liquidity, err := pool.ProvideLiquidity(number.FromString("2100"), number.FromString("100"))
	assert.Nil(err)
	assert.Equal("2100", pool.X.String())
	assert.Equal("100", pool.Y.String())
	assert.Equal("458.25856949", pool.Liquidity.String())
	assert.Equal("458.25756949", liquidity.String())

	liquidity, err = pool.ProvideLiquidity(number.FromString("2100"), number.FromString("100"))
	assert.Nil(err)
	assert.Equal("4200", pool.X.String())
	assert.Equal("200", pool.Y.String())
	assert.Equal("916.51713898", pool.Liquidity.String())
	assert.Equal("458.25856949", liquidity.String())

	liquidity, err = pool.ProvideLiquidity(number.FromString("2100"), number.FromString("100"))
	assert.Nil(err)
	assert.Equal("6300", pool.X.String())
	assert.Equal("300", pool.Y.String())
	assert.Equal("1374.77570847", pool.Liquidity.String())
	assert.Equal("458.25856949", liquidity.String())

	pair, err := pool.RemoveLiquidity(number.FromString("100"))
	assert.Nil(err)
	assert.Equal("5841.7434305", pool.X.String())
	assert.Equal("278.1782586", pool.Y.String())
	assert.Equal("1274.77570847", pool.Liquidity.String())
	assert.Equal("458.2565695", pair.X.String())
	assert.Equal("21.8217414", pair.Y.String())

	pair, err = pool.RemoveLiquidity(number.FromString("1274.77570847"))
	assert.NotNil(err)
	pair, err = pool.RemoveLiquidity(number.FromString("1274.77470847"))
	assert.Nil(err)
	assert.Equal("0.00458257", pool.X.String())
	assert.Equal("0.00021822", pool.Y.String())
	assert.Equal("0.001", pool.Liquidity.String())
	assert.Equal("5841.73884793", pair.X.String())
	assert.Equal("278.17804038", pair.Y.String())

	liquidity, err = pool.ProvideLiquidity(number.FromString("2100"), number.FromString("100"))
	assert.Nil(err)
	assert.Equal("2100.00458257", pool.X.String())
	assert.Equal("100.00021822", pool.Y.String())
	assert.Equal("458.25413903", pool.Liquidity.String())
	assert.Equal("458.25313903", liquidity.String())

	out, extra, err := pool.Swap(number.FromString("1"), true)
	assert.Nil(err)
	assert.Equal("0.001", extra.String())
	assert.Equal("2079.27426341", pool.X.String())
	assert.Equal("100.99921822", pool.Y.String())
	assert.Equal("458.25413903", pool.Liquidity.String())
	assert.Equal("20.73031916", out.Amount.String())

	pair, err = pool.RemoveLiquidity(number.FromString("458.25413903"))
	assert.NotNil(err)
	pair, err = pool.RemoveLiquidity(number.FromString("458.25313903"))
	assert.Nil(err)
	assert.Equal("0.00453739", pool.X.String())
	assert.Equal("0.00022041", pool.Y.String())
	assert.Equal("0.001", pool.Liquidity.String())
	assert.Equal("2079.26972602", pair.X.String())
	assert.Equal("100.99899781", pair.Y.String())
}
