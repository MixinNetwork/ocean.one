package session

import (
	"context"
	"runtime"

	"github.com/MixinNetwork/ocean.one/example/config"
)

func Go(f func(), c context.Context) {
	pc, file, line, _ := runtime.Caller(1)
	funcName := runtime.FuncForPC(pc).Name()
	go func(ctx context.Context, file string, line int, funcName string) {
		defer func() {
			rcv := recover()
			if rcv == nil {
				return
			}
			Logger(ctx).Debugf("[GOROUTINE+] %s:%d:%s => PANIC %v", file, line, funcName, rcv)
		}()
		if config.GoroutineLogEnabled && ctx != nil {
			Logger(ctx).Debugf("[GOROUTINE+] %s:%d:%s", file, line, funcName)
			defer Logger(ctx).Debugf("[GOROUTINE-] %s:%d:%s", file, line, funcName)
		}
		f()
	}(c, file, line, funcName)
}
