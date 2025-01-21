package redisx

import (
	"context"
	"github.com/QuantumShiftX/golib/stores/redisx/config"
	"github.com/redis/go-redis/v9"
)

var Engine redis.UniversalClient

func Must(c config.Config) {
	Engine = NewEngine(c)
}

func NewEngine(c config.Config) redis.UniversalClient {
	rdb := redis.NewUniversalClient(&redis.UniversalOptions{
		Addrs:      c.Addrs,
		Username:   c.Username,
		Password:   c.Password,
		MasterName: c.MasterName,
		DB:         c.DB,
	})

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		panic(err)
	}

	if c.Debug {
		rdb.AddHook(DebugHook{})
	}

	if c.Trace {
		rdb.AddHook(TraceHook{})
	}

	return rdb
}
