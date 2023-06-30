package session

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
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

func ForbiddenError(ctx context.Context) Error {
	description := http.StatusText(http.StatusForbidden)
	return createError(ctx, http.StatusAccepted, http.StatusForbidden, description, nil)
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

func PhoneSMSDeliveryError(ctx context.Context, phone string, err error) Error {
	description := fmt.Sprintf("Failed to deliver SMS to %s.", phone)
	return createError(ctx, http.StatusAccepted, 10003, description, err)
}

func RecaptchaVerifyError(ctx context.Context) Error {
	description := fmt.Sprintf("Recaptcha is invalid.")
	return createError(ctx, http.StatusAccepted, 10004, description, nil)
}

func RecaptchaRequiredError(ctx context.Context) Error {
	description := fmt.Sprintf("Recaptcha is required.")
	return createError(ctx, http.StatusAccepted, 10005, description, nil)
}

func MixinNotConnectedError(ctx context.Context) Error {
	description := fmt.Sprintf("Mixin Messenger not connected.")
	return createError(ctx, http.StatusAccepted, 10006, description, nil)
}

func EmailSMSDeliveryError(ctx context.Context, email string, err error) Error {
	description := fmt.Sprintf("Failed to deliver email to %s.", email)
	return createError(ctx, http.StatusAccepted, 10007, description, err)
}

func PhoneInvalidFormatError(ctx context.Context, phone string) Error {
	description := fmt.Sprintf("Invalid phone number %s.", phone)
	return createError(ctx, http.StatusAccepted, 20110, description, nil)
}

func InsufficientKeyPoolError(ctx context.Context) Error {
	description := "Insufficient keys."
	return createError(ctx, http.StatusAccepted, 20111, description, nil)
}

func VerificationCodeInvalidError(ctx context.Context) Error {
	description := "Invalid verification code."
	return createError(ctx, http.StatusAccepted, 20113, description, nil)
}

func VerificationCodeExpiredError(ctx context.Context) Error {
	description := "Expired verification code."
	return createError(ctx, http.StatusAccepted, 20114, description, nil)
}

func EmailInvalidFormatError(ctx context.Context, email string) Error {
	description := fmt.Sprintf("Invalid email format %s.", email)
	return createError(ctx, http.StatusAccepted, 20115, description, nil)
}

func PasswordTooSimpleError(ctx context.Context) Error {
	description := "Password too simple, at least 8 characters required."
	return createError(ctx, http.StatusAccepted, 20118, description, nil)
}

func createError(ctx context.Context, status, code int, description string, err error) Error {
	pc, file, line, _ := runtime.Caller(2)
	funcName := runtime.FuncForPC(pc).Name()
	trace := fmt.Sprintf("[ERROR %d] %s\n%s:%d:%s", code, description, file, line, funcName)
	if err != nil {
		if sessionError, ok := err.(Error); ok {
			trace = trace + "\n" + sessionError.trace
		} else {
			trace = trace + "\n" + err.Error()
		}
	}

	return Error{
		Status:      status,
		Code:        code,
		Description: description,
		trace:       trace,
	}
}
