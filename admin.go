package main

import (
	"context"
	"crypto/md5"
	"encoding/base64"
	"io"
	"log"
	"time"

	"github.com/MixinNetwork/go-number"
	"github.com/MixinNetwork/ocean.one/config"
	"github.com/MixinNetwork/ocean.one/engine"
	"github.com/MixinNetwork/ocean.one/persistence"
	"github.com/fox-one/mixin-sdk-go"
	"github.com/gofrs/uuid/v5"
	"github.com/ugorji/go/codec"
)

func (ex *Exchange) adminCollectionOmniUSDTFromAllBrokers(ctx context.Context, recipientId string) error {
	omni := "815b0b1a-2764-3736-8faa-42d694fa620a"
	for _, u := range ex.brokers {
		ks := &mixin.Keystore{
			ClientID:   u.BrokerId,
			SessionID:  u.SessionId,
			PrivateKey: u.SessionKey,
			PinToken:   u.PINToken,
		}
		client, err := mixin.NewFromKeystore(ks)
		if err != nil {
			return err
		}
		asset, err := client.ReadAsset(ctx, omni)
		log.Println(asset, err)
		if err != nil {
			return err
		}
		if asset.Balance.IsZero() {
			continue
		}
		in := &mixin.TransferInput{
			AssetID:    omni,
			OpponentID: recipientId,
			Amount:     asset.Balance,
			TraceID:    mixin.UniqueConversationID("ADMIN|COLLECT|OMNI", u.BrokerId),
		}
		snap, err := client.Transfer(ctx, in, u.DecryptedPIN)
		log.Println(snap, err)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ex *Exchange) adminSendCancelOrderTransactionForOmniUSDT(ctx context.Context, a *persistence.Action) error {
	if a.Action != engine.OrderActionCreate {
		return nil
	}
	if a.Order.QuoteAssetId != "815b0b1a-2764-3736-8faa-42d694fa620a" {
		return nil
	}

	out := make([]byte, 140)
	encoder := codec.NewEncoderBytes(&out, ex.codec)
	data := OrderAction{O: uuid.Must(uuid.FromString(a.Order.OrderId))}
	err := encoder.Encode(data)
	if err != nil {
		return err
	}
	memo := base64.StdEncoding.EncodeToString(out)
	if len(memo) > 140 {
		log.Panicln(memo)
	}
	traceId := getAdminSettlementId(a.Order.OrderId, "ADMIN|CANCEL")
	for {
		err := ex.sendTransfer(ctx, config.ClientId, "0fdf3e21-428e-3fb2-a357-0f0a8886ec5c", "de5a6414-c181-3ecc-b401-ce375d08c399", number.FromString("1"), traceId, memo)
		if err == nil {
			break
		}
		log.Println("admin.sendTransfer => ", err)
		time.Sleep(time.Second)
	}
	return nil
}

func getAdminSettlementId(id, modifier string) string {
	h := md5.New()
	io.WriteString(h, id)
	io.WriteString(h, modifier)
	sum := h.Sum(nil)
	sum[6] = (sum[6] & 0x0f) | 0x30
	sum[8] = (sum[8] & 0x3f) | 0x80
	return uuid.FromBytesOrNil(sum).String()
}
