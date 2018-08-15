package models

import (
	"testing"

	"github.com/MixinNetwork/ocean.one/example/uuid"
	"github.com/stretchr/testify/assert"
)

func TestMarketCRUD(t *testing.T) {
	ctx := setupTestContext()
	defer teardownTestContext(ctx)
	assert := assert.New(t)

	user := &User{UserId: uuid.NewV4().String()}
	fm, err := user.LikeMarket(ctx, uuid.NewV4().String(), uuid.NewV4().String())
	assert.NotNil(err)
	assert.Nil(fm)
	fm, err = user.LikeMarket(ctx, "c6d0c728-2624-429b-8e0d-d9d19b6592fa", "815b0b1a-2764-3736-8faa-42d694fa620a")
	assert.Nil(err)
	assert.NotNil(fm)
	set, err := readFavoriteMarkets(ctx, &User{UserId: uuid.NewV4().String()})
	assert.Nil(err)
	assert.Len(set, 0)
	set, err = readFavoriteMarkets(ctx, user)
	assert.Nil(err)
	assert.Len(set, 1)
	err = user.DislikeMarket(ctx, "c6d0c728-2624-429b-8e0d-d9d19b6592fa", "815b0b1a-2764-3736-8faa-42d694fa620a")
	assert.Nil(err)
	set, err = readFavoriteMarkets(ctx, user)
	assert.Nil(err)
	assert.Len(set, 0)
	fm, err = user.LikeMarket(ctx, "c6d0c728-2624-429b-8e0d-d9d19b6592fa", "815b0b1a-2764-3736-8faa-42d694fa620a")
	assert.Nil(err)
	assert.NotNil(fm)

	err = CreateOrUpdateMarket(ctx, "c6d0c728-2624-429b-8e0d-d9d19b6592fa", "815b0b1a-2764-3736-8faa-42d694fa620a", 0, 0, 0, 0, 0)
	assert.Nil(err)
	markets, err := ListActiveMarkets(ctx, user)
	assert.Nil(err)
	for _, m := range markets {
		assert.True(m.IsLikedBy)
	}
}
