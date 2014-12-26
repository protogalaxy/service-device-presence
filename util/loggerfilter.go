package util

import (
	"os"

	"github.com/arjantop/saola"
	"golang.org/x/net/context"
	"gopkg.in/inconshreveable/log15.v2"
)

func newLogger() log15.Logger {
	logger := log15.New()
	logger.SetHandler(log15.StreamHandler(os.Stdout, log15.LogfmtFormat()))
	return logger
}

func NewContextLoggerFilter() saola.Filter {
	logger := newLogger()
	return saola.FuncFilter(func(ctx context.Context, s saola.Service) error {
		return s.Do(WithContextLogger(ctx, logger))
	})
}

type key int

const contextLoggerKey key = 0

func WithContextLogger(ctx context.Context, logger log15.Logger) context.Context {
	return context.WithValue(ctx, contextLoggerKey, logger)
}

func GetContextLogger(ctx context.Context) log15.Logger {
	if logger, ok := ctx.Value(contextLoggerKey).(log15.Logger); ok {
		return logger
	}
	return newLogger()
}
