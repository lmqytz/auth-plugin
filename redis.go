package main

import (
	"github.com/gomodule/redigo/redis"
	"time"
)

func NewRedis(conf RedisConf) *Redis {
	pool := &redis.Pool{
		MaxIdle:     conf.MaxIdle,
		IdleTimeout: time.Duration(conf.MaxIdleTime) * time.Second,
		MaxActive:   conf.MaxActive,
		Dial: func() (redis.Conn, error) {
			var options []redis.DialOption
			options = append(options, redis.DialConnectTimeout(time.Duration(conf.Timeout)*time.Second))
			conn, err := redis.Dial("tcp", conf.Address, options...)
			if conn == nil || err != nil {
				return nil, err
			}

			if conf.Password != "" {
				_, err = conn.Do("AUTH", conf.Password)
				if err != nil {
					conn.Close()
					return nil, err
				}
			}

			_, err = conn.Do("SELECT", conf.Db)
			if err != nil {
				conn.Close()
				return nil, err
			}

			return conn, nil
		},
	}

	if conf.HealthCheckInterval > 0 {
		pool.TestOnBorrow = func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Duration(conf.HealthCheckInterval)*time.Second {
				return nil
			}
			_, err := c.Do("PING")
			return err
		}
	}

	return &Redis{pool: pool}
}

type Redis struct {
	pool *redis.Pool
}

func (redis *Redis) Do(commandName string, args ...interface{}) (reply interface{}, err error) {
	conn := redis.pool.Get()
	defer func() {
		_ = conn.Close()
	}()

	return conn.Do(commandName, args...)
}
