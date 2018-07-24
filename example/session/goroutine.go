package session

import (
	"context"
	"fmt"
	"runtime"

	"github.com/MixinNetwork/ocean.one/example/config"
	"github.com/bugsnag/bugsnag-go"
)

func Go(f func(), c context.Context) {
	pc, file, line, _ := runtime.Caller(1)
	funcName := runtime.FuncForPC(pc).Name()
	go func(ctx context.Context, file string, line int, funcName string) {
		if ctx != nil && Request(ctx) != nil {
			defer bugsnag.Recover(fmt.Errorf("[RECOVER] %s:%d", file, line), bugsnag.SeverityInfo, bugsnag.ErrorClass{funcName}, Request(ctx))
		} else {
			defer bugsnag.Recover(fmt.Errorf("[RECOVER] %s:%d", file, line), bugsnag.SeverityInfo, bugsnag.ErrorClass{funcName})
		}
		if config.GoroutineLogEnabled && ctx != nil {
			Logger(ctx).Debugf("[GOROUTINE+] %s:%d:%s", file, line, funcName)
			defer Logger(ctx).Debugf("[GOROUTINE-] %s:%d:%s", file, line, funcName)
		}
		f()
	}(c, file, line, funcName)
}
