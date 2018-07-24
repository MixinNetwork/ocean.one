package session

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"runtime"

	"github.com/bugsnag/bugsnag-go"
)

type Error struct {
	Status      int    `json:"status"`
	Code        int    `json:"code"`
	Description string `json:"description"`
	trace       string
}

func (sessionError Error) Error() string {
	str, err := json.Marshal(sessionError)
	if err != nil {
		log.Panicln(err)
	}
	return string(str)
}

func ParseError(err string) (Error, bool) {
	var sessionErr Error
	json.Unmarshal([]byte(err), &sessionErr)
	return sessionErr, sessionErr.Code > 0 && sessionErr.Description != ""
}

func BadRequestError(ctx context.Context) Error {
	description := "The request body can’t be pasred as valid data."
	return createError(ctx, http.StatusAccepted, http.StatusBadRequest, description, nil)
}

func NotFoundError(ctx context.Context) Error {
	description := "The endpoint is not found."
	return createError(ctx, http.StatusAccepted, http.StatusNotFound, description, nil)
}

func AuthorizationError(ctx context.Context) Error {
	description := "Unauthorized, maybe invalid token."
	return createError(ctx, http.StatusAccepted, 401, description, nil)
}

func TooManyRequestsError(ctx context.Context) Error {
	description := http.StatusText(http.StatusTooManyRequests)
	return createError(ctx, http.StatusAccepted, http.StatusTooManyRequests, description, nil)
}

func ServerError(ctx context.Context, err error) Error {
	description := http.StatusText(http.StatusInternalServerError)
	return createError(ctx, http.StatusInternalServerError, http.StatusInternalServerError, description, err)
}

func TransactionError(ctx context.Context, err error) Error {
	description := http.StatusText(http.StatusInternalServerError)
	return createError(ctx, http.StatusInternalServerError, 10001, description, err)
}

func BadDataError(ctx context.Context) Error {
	description := "The request data has invalid field."
	return createError(ctx, http.StatusAccepted, 10002, description, nil)
}

func PhoneInvalidFormatError(ctx context.Context, phone string) Error {
	description := fmt.Sprintf("Invalid phone number %s.", phone)
	return createError(ctx, http.StatusAccepted, 20110, description, nil)
}

func createError(ctx context.Context, status, code int, description string, err error) Error {
	pc, file, line, _ := runtime.Caller(2)
	funcName := runtime.FuncForPC(pc).Name()
	trace := fmt.Sprintf("[ERROR %d] %s\n%s:%d", code, description, file, line)
	if err != nil {
		if sessionError, ok := err.(Error); ok {
			trace = trace + "\n" + sessionError.trace
		} else {
			trace = trace + "\n" + err.Error()
		}
	}

	if ctx != nil {
		class := bugsnag.ErrorClass{fmt.Sprintf("%s$%d", funcName, code)}
		rawData := []interface{}{bugsnag.SeverityError, class}
		meta := bugsnag.MetaData{}
		if claims := AuthorizationInfo(ctx); claims != nil {
			userId := fmt.Sprint(claims["uid"])
			if _, found := claims["uid"]; !found {
				userId = fmt.Sprint(claims["sub"])
			}
			meta["claims"] = claims
			rawData = append(rawData, bugsnag.User{Id: userId})
		}
		if r := Request(ctx); r != nil {
			rawData = append(rawData, r)
			if RequestBody(ctx) != "" {
				meta["body"] = map[string]interface{}{"data": RequestBody(ctx)}
			}
		}
		rawData = append(rawData, meta)
		bugsnag.Notify(errors.New(trace), rawData...)
		if logger := Logger(ctx); logger != nil {
			logger.Error(trace)
		}
	}

	return Error{
		Status:      status,
		Code:        code,
		Description: description,
		trace:       trace,
	}
}
