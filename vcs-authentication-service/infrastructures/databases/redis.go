package databases

import (
	"github.com/redis/go-redis/v9"
	"github.com/PhucNguyen204/vcs-authentication-service/pkg/env"
)

type IRedisFactory interface {
	ConnectRedis() *redis.Client
}

type redisFactory struct {
	Address  string
	Password string
	Db       int
}

func NewRedisFactory(env env.RedisEnv) IRedisFactory {
	return &redisFactory{
		Address:  env.RedisAddress,
		Password: env.RedisPassword,
		Db:       env.RedisDb,
	}
}

func (f *redisFactory) ConnectRedis() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     f.Address,
		Password: f.Password,
		DB:       f.Db,
	})
}
