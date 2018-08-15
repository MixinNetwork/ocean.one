package models

import (
	"context"

	"cloud.google.com/go/spanner"
	"github.com/MixinNetwork/ocean.one/example/session"
	"google.golang.org/api/iterator"
)

type FavoriteMarket struct {
	UserId string
	Base   string
	Quote  string
}

var favoriteMarketsColumnsFull = []string{"user_id", "base", "quote"}

func (fm *FavoriteMarket) valuesFull() []interface{} {
	return []interface{}{fm.UserId, fm.Base, fm.Quote}
}

func (user *User) LikeMarket(ctx context.Context, base, quote string) (*FavoriteMarket, error) {
	if !validateMarket(base, quote) {
		return nil, session.BadDataError(ctx)
	}

	fm := &FavoriteMarket{
		UserId: user.UserId,
		Base:   base,
		Quote:  quote,
	}
	err := session.Database(ctx).Apply(ctx, []*spanner.Mutation{
		spanner.Insert("favorite_markets", favoriteMarketsColumnsFull, fm.valuesFull()),
	}, "favorite_markets", "INSERT", "LikeMarket")
	if err != nil {
		return nil, session.TransactionError(ctx, err)
	}
	return fm, nil
}

func (user *User) DislikeMarket(ctx context.Context, base, quote string) error {
	if !validateMarket(base, quote) {
		return session.BadDataError(ctx)
	}
	err := session.Database(ctx).Apply(ctx, []*spanner.Mutation{
		spanner.Delete("favorite_markets", spanner.Key{user.UserId, base, quote}),
	}, "favorite_markets", "INSERT", "LikeMarket")
	if err != nil {
		return session.TransactionError(ctx, err)
	}
	return nil
}

func readFavoriteMarkets(ctx context.Context, current *User) (map[string]bool, error) {
	if current == nil {
		return map[string]bool{}, nil
	}

	stmt := spanner.Statement{
		SQL:    "SELECT base,quote FROM favorite_markets WHERE user_id=@user_id LIMIT 100",
		Params: map[string]interface{}{"user_id": current.UserId},
	}
	it := session.Database(ctx).Query(ctx, stmt, "favorite_markets", "readFavoriteMarkets")
	defer it.Stop()

	set := make(map[string]bool, 0)
	for {
		row, err := it.Next()
		if err == iterator.Done {
			return set, nil
		} else if err != nil {
			return nil, session.TransactionError(ctx, err)
		}
		var base, quote string
		if err := row.Columns(&base, &quote); err != nil {
			return set, session.TransactionError(ctx, err)
		}
		set[base+quote] = true
	}
}

func validateMarket(base, quote string) bool {
	var bases []string
	switch quote {
	case "815b0b1a-2764-3736-8faa-42d694fa620a":
		bases = usdtMarkets
	case "c6d0c728-2624-429b-8e0d-d9d19b6592fa":
		bases = btcMarkets
	case "c94ac88f-4671-3976-b60a-09064f1811e8":
		bases = xinMarkets
	default:
		return false
	}
	for _, m := range bases {
		if base == m {
			return true
		}
	}
	return false
}
