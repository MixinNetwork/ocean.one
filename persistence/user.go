package persistence

import (
	"context"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/dgrijalva/jwt-go"
	"github.com/satori/go.uuid"
	"google.golang.org/api/iterator"
)

type User struct {
	UserId    string
	PublicKey string
}

func UpdateUserPublicKey(ctx context.Context, userId, publicKey string) error {
	if _, err := hex.DecodeString(publicKey); err != nil {
		return nil
	}

	_, err := Spanner(ctx).ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		it := txn.ReadUsingIndex(ctx, "users", "users_by_public_key", spanner.Key{publicKey}, []string{"user_id"})
		defer it.Stop()

		_, err := it.Next()
		if err == iterator.Done {
		} else if err != nil {
			return err
		} else {
			return nil
		}

		txn.BufferWrite([]*spanner.Mutation{spanner.InsertOrUpdateMap("users", map[string]interface{}{
			"user_id":    userId,
			"public_key": publicKey,
		})})
		return nil
	})

	return err
}

func Authenticate(ctx context.Context, jwtToken string) (string, error) {
	var userId string
	token, err := jwt.Parse(jwtToken, func(token *jwt.Token) (interface{}, error) {
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return nil, nil
		}
		_, ok = token.Method.(*jwt.SigningMethodECDSA)
		if !ok {
			return nil, nil
		}
		id, _ := uuid.FromString(fmt.Sprint(claims["uid"]))
		if id.String() == uuid.Nil.String() {
			return nil, nil
		} else {
			userId = id.String()
		}

		it := Spanner(ctx).Single().Read(ctx, "users", spanner.Key{userId}, []string{"public_key"})
		defer it.Stop()
		row, err := it.Next()
		if err == iterator.Done {
			return nil, nil
		} else if err != nil {
			return nil, err
		}
		var publicKey string
		err = row.Columns(&publicKey)
		if err != nil {
			return nil, err
		}

		return hex.DecodeString(publicKey)
	})

	if err != nil && strings.Contains(err.Error(), "spanner") {
		return "", err
	}
	if err == nil && token.Valid {
		return userId, nil
	}
	return "", nil
}

func UserOrders(ctx context.Context, userId string, market, state string, offset time.Time, limit int) ([]*Order, error) {
	txn := Spanner(ctx).ReadOnlyTransaction()
	defer txn.Close()

	if limit > 100 {
		limit = 100
	}

	base, quote := getBaseQuote(market)
	if base == "" || quote == "" {
		return nil, nil
	}

	query := "SELECT order_id FROM orders@{FORCE_INDEX=orders_by_user_created_desc} WHERE user_id=@user_id AND created_at<@offset AND state=@state"
	params := map[string]interface{}{"user_id": userId, "offset": offset, "state": state}
	if base != "" && quote != "" {
		query = query + " AND base_asset_id=@base AND quote_asset_id=@quote"
		params["base"], params["quote"] = base, quote
	}
	query = query + " ORDER BY user_id,created_at DESC"
	query = fmt.Sprintf("%s LIMIT %d", query, limit)

	iit := txn.Query(ctx, spanner.Statement{query, params})
	defer iit.Stop()

	var orderIds []string
	for {
		row, err := iit.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			return nil, err
		}
		var id string
		err = row.Columns(&id)
		if err != nil {
			return nil, err
		}
		orderIds = append(orderIds, id)
	}

	oit := txn.Query(ctx, spanner.Statement{
		SQL:    "SELECT * FROM orders WHERE order_id IN UNNEST(@order_ids)",
		Params: map[string]interface{}{"order_ids": orderIds},
	})
	defer oit.Stop()

	var orders []*Order
	for {
		row, err := oit.Next()
		if err == iterator.Done {
			break
		} else if err != nil {
			return orders, err
		}
		var o Order
		err = row.ToStruct(&o)
		if err != nil {
			return orders, err
		}
		orders = append(orders, &o)
	}
	sort.Slice(orders, func(i, j int) bool { return orders[i].CreatedAt.After(orders[j].CreatedAt) })
	return orders, nil
}
