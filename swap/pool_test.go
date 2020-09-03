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
}
