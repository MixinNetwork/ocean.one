package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/MixinNetwork/go-number"
	"github.com/MixinNetwork/ocean.one/cache"
	"github.com/MixinNetwork/ocean.one/persistence"
	"github.com/MixinNetwork/ocean.one/swap"
	"github.com/gofrs/uuid"
)

type PoolQueue struct {
	store   *persistence.Pool
	pool    *swap.Pool
	actions chan *persistence.SwapAction
	queue   *cache.Queue
}

type SwapAction struct {
	Base   uuid.UUID
	Quote  uuid.UUID
	Action byte
	Extra  uuid.UUID
}

func (ex *Exchange) PollSwapActions(ctx context.Context) {
	checkpoint, limit := time.Time{}, 500
	for {
		actions, err := persistence.ListPendingSwapActions(ctx, checkpoint, limit)
		if err != nil {
			log.Println("ListPendingSwapActions", err)
			time.Sleep(PollInterval)
			continue
		}
		for _, a := range actions {
			pq := ex.pools[a.Key()]
			pq.actions <- a
			checkpoint = a.CreatedAt
		}
		if len(actions) < limit {
			time.Sleep(PollInterval)
		}
	}
}

func (ex *Exchange) AttachPool(ctx context.Context, p *persistence.Pool) {
	ex.pools[p.Key()] = &PoolQueue{
		store:   p,
		pool:    p.Swap(),
		actions: make(chan *persistence.SwapAction, 1024),
		queue:   cache.NewQueue(ctx, "SWAP-"+p.Key()),
	}
	go ex.LoopPoolQueue(ctx, ex.pools[p.Key()])
}

func (ex *Exchange) LoopPoolQueue(ctx context.Context, pq *PoolQueue) {
	for {
		select {
		case <-ctx.Done():
		case a := <-pq.actions:
			amount := number.FromString(a.Amount)
			switch a.Action {
			case persistence.SwapAdd:
				if a.Expired() {
					ex.ensureSwapAdd(ctx, pq, a, nil, number.Zero())
				} else if aa := pq.cache.GetAnotherAsset(ctx, a); aa == nil {
					pq.cache.AddAsset(ctx, a)
				} else {
					var err error
					var liq number.Decimal
					aAmount := number.FromString(aa.Amount)
					if a.AssetId == pq.BaseAssetId {
						liq, err = pq.pool.ProvideLiquidity(amount, aAmount)
					} else {
						liq, err = pq.pool.ProvideLiquidity(aAmount, amount)
					}
					ex.ensureSwapAdd(ctx, pq, a, aa, liq)
				}
			case persistence.SwapExpire:
				ex.ensureSwapAdd(ctx, pq, a, nil, number.Zero())
			case persistence.SwapRemove:
				pair, _ := pq.pool.RemoveLiquidity(amount)
				ex.ensureSwapRemove(ctx, pq, a, pair)
			case persistence.SwapTrade:
				output, _, _ := pq.pool.Swap(amount, a.AssetId == pq.store.QuoteAssetId)
				ex.ensureSwapTrade(ctx, pq, a, output)
			}
		}
	}
}

func (ex *Exchange) ensureSwapTrade(ctx context.Context, pq *PoolQueue, a *persistence.SwapAction, out *swap.Output) {
	for {
		err := ex.doSwapTrade(ctx, pq, a, out)
		if err == nil {
			break
		}
		log.Println("ensureSwapTrade", err)
		time.Sleep(100 * time.Millisecond)
	}
}

func (ex *Exchange) doSwapTrade(ctx context.Context, pq *PoolQueue, a *persistence.SwapAction, out *swap.Output) error {
	if out == nil || out.Amount.IsZero() {
		return ex.refundSwapAction(ctx, a)
	}
	// in spanner transaction
	// delete swap actions
	// create transfers for
	// pool to user asset in out
	// send out websocket event
	return nil
}

func (ex *Exchange) ensureSwapRemove(ctx context.Context, pq *PoolQueue, a *persistence.SwapAction, pair *swap.Pair) {
	for {
		err := ex.doSwapRemove(ctx, pq, a, pair)
		if err == nil {
			break
		}
		log.Println("ensureSwapRemove", err)
		time.Sleep(100 * time.Millisecond)
	}
}

func (ex *Exchange) doSwapRemove(ctx context.Context, pq *PoolQueue, a *persistence.SwapAction, pair *swap.Pair) error {
	if pair == nil || pair.X.IsZero() || pair.Y.IsZero() {
		return ex.refundSwapAction(ctx, a)
	}
	// in spanner transaction
	// delete swap actions
	// create transfers for
	// 1. pool to user asset x
	// 2. pool to user asset y
	// 3. liquidity to pool asset token
	// send out websocket event
	return nil
}

func (ex *Exchange) decryptSwapAction(s *Snapshot, payload []byte) (*SwapAction, error) {
	if len(payload) != 33 && len(payload) != 49 {
		return nil, fmt.Errorf("invalid swap payload length %d", len(payload))
	}
	base, err := uuid.FromBytes(payload[:16])
	if err != nil {
		return nil, err
	}
	quote, err := uuid.FromBytes(payload[16:32])
	if err != nil {
		return nil, err
	}
	swap := &SwapAction{
		Base:  base,
		Quote: quote,
	}
	if len(payload) == 32 {
		swap.Action = 2
		return swap, nil
	}

	swap.Action = payload[32]
	if swap.Action != 0 && swap.Action != 1 {
		return nil, fmt.Errorf("invalid swap action %d", swap.Action)
	}
	if len(payload) == 33 {
		return swap, nil
	}
	if base.String() >= quote.String() {
		return nil, fmt.Errorf("invalid swap pair %s:%s", base, quote)
	}

	extra, err := uuid.FromBytes(payload[33:])
	if err != nil {
		return nil, err
	}
	swap.Extra = extra
	return swap, nil
}

func (ex *Exchange) handleSwapAction(ctx context.Context, s *Snapshot, swap *SwapAction) error {
	pk := swap.Base.String() + "-" + swap.Quote.String()
	if ex.pools[pk] == nil {
		pp, err := persistence.MakePool(ctx, swap.Base.String(), swap.Quote.String())
		if err != nil {
			return err
		}
		ex.AttachPool(ctx, pp)
		ex.swaps[pp.PoolId] = pp.Payload()
	}
	pq := ex.pools[pk]
	action := &persistence.SwapAction{
		ActionId:     s.SnapshotId,
		PoolId:       pq.store.PoolId,
		BaseAssetId:  swap.Base.String(),
		QuoteAssetId: swap.Quote.String(),
		AssetId:      s.Asset.AssetId,
		Amount:       s.Amount,
		BrokerId:     s.UserId,
		UserId:       s.OpponentId,
		TraceId:      s.TraceId,
	}
	switch swap.Action {
	case 0:
		if s.Asset.AssetId != action.BaseAssetId && s.Asset.AssetId != action.QuoteAssetId {
			return ex.refundSnapshot(ctx, s)
		}
		action.Action = persistence.SwapTrade
	case 1:
		if s.Asset.AssetId != action.BaseAssetId && s.Asset.AssetId != action.QuoteAssetId {
			return ex.refundSnapshot(ctx, s)
		}
		action.Action = persistence.SwapAdd
	case 2:
		if s.Asset.AssetId != action.PoolId {
			return ex.refundSnapshot(ctx, s)
		}
		action.Action = persistence.SwapRemove
	}
	return persistence.WriteSwapAction(ctx, action)
}

func (ex *Exchange) refundSwapAction(ctx context.Context, a *persistence.SwapAction) error {
	s := &Snapshot{
		UserId:     a.BrokerId,
		OpponentId: a.UserId,
		Amount:     a.Amount,
		TraceId:    a.TraceId,
	}
	s.Asset.AssetId = a.AssetId
	return ex.refundSnapshot(ctx, s)
}
