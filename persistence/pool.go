package persistence

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"cloud.google.com/go/spanner"
	"github.com/MixinNetwork/go-number"
	"github.com/MixinNetwork/ocean.one/swap"
	"github.com/gofrs/uuid"
	"google.golang.org/api/iterator"
)

const (
	PoolFormulaConstantProduct = "CONSTANT_PRODUCT"
)

type Pool struct {
	PoolId       string `spanner:"pool_id"`
	BaseAssetId  string `spanner:"base_asset_id"`
	QuoteAssetId string `spanner:"quote_asset_id"`
	BaseAmount   string `spanner:"base_amount"`
	QuoteAmount  string `spanner:"quote_amount"`
	Liquidity    string `spanner:"liquidity"`
	Formula      string `spanner:"formula"`
}

func (p *Pool) Swap() *swap.Pool {
	x := number.FromString(p.BaseAmount)
	y := number.FromString(p.QuoteAmount)
	liquidity := number.FromString(p.Liquidity)

	switch p.Formula {
	case PoolFormulaConstantProduct:
		return swap.BuildConstantProductPool(x, y, liquidity)
	default:
		panic("invalid formula")
	}
}

func (p *Pool) Key() string {
	return p.BaseAssetId + "-" + p.QuoteAssetId
}

func (p *Pool) Payload() []byte {
	bu, err := uuid.FromString(p.BaseAssetId)
	if err != nil {
		panic(err)
	}
	qu, err := uuid.FromString(p.QuoteAssetId)
	if err != nil {
		panic(err)
	}
	pb, pq := bu.Bytes(), qu.Bytes()
	return append(pb, pq...)
}

func AllPools(ctx context.Context) ([]*Pool, error) {
	it := Spanner(ctx).Single().Query(ctx, spanner.Statement{SQL: "SELECT * FROM pools"})
	defer it.Stop()

	var pools []*Pool
	for {
		row, err := it.Next()
		if err == iterator.Done {
			return pools, nil
		} else if err != nil {
			return pools, err
		}
		var p Pool
		err = row.ToStruct(&p)
		if err != nil {
			return pools, err
		}
		pools = append(pools, &p)
	}
}

func MakePool(ctx context.Context, base, quote string) (*Pool, error) {
	if base >= quote {
		return nil, fmt.Errorf("invalid swap pair %s:%s", base, quote)
	}
	err := validateAssetId(base)
	if err != nil {
		return nil, err
	}
	err = validateAssetId(quote)
	if err != nil {
		return nil, err
	}
	pool := &Pool{
		BaseAssetId:  base,
		QuoteAssetId: quote,
		BaseAmount:   number.Zero().String(),
		QuoteAmount:  number.Zero().String(),
		Liquidity:    number.Zero().String(),
		Formula:      PoolFormulaConstantProduct,
	}
	_, err = Spanner(ctx).ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		exist, err := checkSwapPoolExistence(ctx, txn, base, quote)
		if err != nil || exist {
			return err
		}
		poolId, err := readPoolLiquidityProviderAssetId(ctx, txn)
		if err != nil {
			return err
		} else if poolId == "" {
			return fmt.Errorf("no vailable pool liquidity provider assets right now %s:%s", base, quote)
		}
		pool.PoolId = poolId
		providerMutation := spanner.Delete("swap_tokens", spanner.Key{poolId})
		poolMutation, err := spanner.InsertStruct("swap_pools", pool)
		if err != nil {
			return err
		}
		return txn.BufferWrite([]*spanner.Mutation{providerMutation, poolMutation})
	})
	return pool, err
}

func readPoolLiquidityProviderAssetId(ctx context.Context, txn *spanner.ReadWriteTransaction) (string, error) {
	it := txn.Read(ctx, "swap_tokens", spanner.AllKeys(), []string{"asset_id"})
	defer it.Stop()

	row, err := it.Next()
	if err == iterator.Done {
		return "", nil
	} else if err != nil {
		return "", err
	}
	var id string
	err = row.Columns(&id)
	return id, err
}

func checkSwapPoolExistence(ctx context.Context, txn *spanner.ReadWriteTransaction, base, quote string) (bool, error) {
	it := txn.ReadUsingIndex(ctx, "swap_pools", "swap_pools_by_base_quote", spanner.Key{base, quote}, []string{"created_at"})
	defer it.Stop()

	_, err := it.Next()
	if err == iterator.Done {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func validateAssetId(id string) error {
	uid, err := uuid.FromString(id)
	if err != nil {
		return err
	}
	res, err := http.Get("https://api.mixin.one/network/assets/" + uid.String())
	if err != nil {
		return err
	}
	defer res.Body.Close()

	var body struct {
		Data struct {
			AssetId string `json:"asset_id"`
		} `json:"data"`
	}
	err = json.NewDecoder(res.Body).Decode(&body)
	if err != nil {
		return err
	}
	if body.Data.AssetId != id {
		return fmt.Errorf("invalid asset %s %s", id, body.Data.AssetId)
	}
	return nil
}
