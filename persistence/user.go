package persistence

import (
	"context"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/dgrijalva/jwt-go"
	"github.com/gofrs/uuid"
	"google.golang.org/api/iterator"
)

type User struct {
	UserId    string
	PublicKey string
}

func UpdateUserPublicKey(ctx context.Context, userId, publicKey string) error {
	pkix, err := hex.DecodeString(publicKey)
	if err != nil {
		return nil
	}
	_, err = x509.ParsePKIXPublicKey(pkix)
	if err != nil {
		return nil
	}

	_, err = Spanner(ctx).ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		it := txn.ReadUsingIndex(ctx, "users", "users_by_public_key", spanner.Key{publicKey}, []string{"user_id"})
		defer it.Stop()

		_, err := it.Next()
		if err == iterator.Done {
		} else if err != nil {
			return err
		} else {
			return nil
		}

		return txn.BufferWrite([]*spanner.Mutation{spanner.InsertOrUpdateMap("users", map[string]interface{}{
			"user_id":    userId,
			"public_key": publicKey,
		})})
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

		pkix, err := hex.DecodeString(publicKey)
		if err != nil {
			return nil, err
		}
		return x509.ParsePKIXPublicKey(pkix)
	})

	if err != nil && strings.Contains(err.Error(), "spanner") {
		return "", err
	}
	if err == nil && token.Valid {
		return userId, nil
	}
	return "", nil
}

func UserOrder(ctx context.Context, orderId, userId string) (*Order, error) {
	it := Spanner(ctx).Single().Query(ctx, spanner.Statement{
		SQL:    "SELECT * FROM orders WHERE order_id=@order_id AND user_id=@user_id",
		Params: map[string]interface{}{"order_id": orderId, "user_id": userId},
	})
	defer it.Stop()

	row, err := it.Next()
	if err == iterator.Done {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	var o Order
	err = row.ToStruct(&o)
	return &o, err
}

func UserOrders(ctx context.Context, userId string, market, state string, offset time.Time, order string, limit int) ([]*Order, error) {
	txn := Spanner(ctx).ReadOnlyTransaction()
	defer txn.Close()

	if limit > 100 {
		limit = 100
	}
	cmp := "<"
	if order != "DESC" {
		order = "ASC"
		cmp = ">"
	}

	base, quote := getBaseQuote(market)
	query := "SELECT order_id FROM orders@{FORCE_INDEX=orders_by_user_state_created_%s} WHERE user_id=@user_id AND created_at%s=@offset AND state=@state"
	query = fmt.Sprintf(query, strings.ToLower(order), cmp)
	params := map[string]interface{}{"user_id": userId, "offset": offset, "state": state}
	if base != "" && quote != "" {
		query = query + " AND base_asset_id=@base AND quote_asset_id=@quote"
		params["base"], params["quote"] = base, quote
	}
	query = query + " ORDER BY user_id,state,created_at " + order
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
	if order == "DESC" {
		sort.Slice(orders, func(i, j int) bool { return orders[i].CreatedAt.After(orders[j].CreatedAt) })
	} else {
		sort.Slice(orders, func(i, j int) bool { return orders[i].CreatedAt.Before(orders[j].CreatedAt) })
	}
	return orders, nil
}
