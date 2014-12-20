package main

import (
	"log"
	"time"

	"github.com/arjantop/cuirass"
	"github.com/arjantop/saola/httpservice"
	"github.com/arjantop/vaquita"
	"github.com/garyburd/redigo/redis"
	"github.com/protogalaxy/service-device-presence/service"
)

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

	config := vaquita.NewEmptyMapConfig()
	propertyFactory := vaquita.NewPropertyFactory(config)
	properties := service.NewBucketProperties(propertyFactory)
	exec := cuirass.NewExecutor(config)

	endpoint := httpservice.NewEndpoint()
	endpoint.PUT("/status/:deviceType/:deviceId", service.NewSetDeviceStatus(exec, properties, redisPool))
	endpoint.GET("/users/:userId", service.NewGetUserDevices(exec, properties, redisPool))

	log.Fatal(httpservice.Serve(":10000", endpoint))
}
