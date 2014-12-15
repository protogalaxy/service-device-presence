package main

import (
	"log"
	"net/http"
	"time"

	"github.com/arjantop/cuirass"
	"github.com/arjantop/vaquita"
	"github.com/garyburd/redigo/redis"
	"github.com/julienschmidt/httprouter"
	"github.com/protogalaxy/service-device-presence/service"
)

type HttpService interface {
	http.Handler
}

func DoPing(c redis.Conn) error {
	_, err := c.Do("PING")
	return err
}

func NewRedisPool() *redis.Pool {
	return &redis.Pool{
		MaxIdle:     20,
		IdleTimeout: time.Minute,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", "localhost:6379")
			if err != nil {
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			return DoPing(c)
		},
	}
}

func main() {
	redisPool := NewRedisPool()

	exec := cuirass.NewExecutor(vaquita.NewEmptyMapConfig())

	router := httprouter.New()
	router.PUT("/status/:deviceType/:deviceId", service.NewSetDeviceStatus(exec, redisPool).ServeHTTP)
	router.GET("/users/:userId", service.NewGetUserDevices(exec, redisPool).ServeHTTP)

	log.Fatal(http.ListenAndServe(":10000", router))
}
