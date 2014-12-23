package main

import (
	"log"
	"time"

	"github.com/arjantop/cuirass"
	"github.com/arjantop/cuirass/metricsstream"
	"github.com/arjantop/saola"
	"github.com/arjantop/saola/httpservice"
	"github.com/arjantop/vaquita"
	"github.com/garyburd/redigo/redis"
	"github.com/protogalaxy/service-device-presence/service"
	"github.com/protogalaxy/service-device-presence/util"
	"github.com/quipo/statsd"
)

import (
	"net/http"
	_ "net/http/pprof"
)

func DoPing(c redis.Conn) error {
	_, err := c.Do("PING")
	return err
}

func NewRedisPool() *redis.Pool {
	return &redis.Pool{
		MaxIdle:     50,
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

	statsdClient := statsd.NewStatsdClient("localhost:8125", "stats.service.devicepresence.")
	statsdClient.CreateSocket()

	endpoint := httpservice.NewEndpoint()

	endpoint.PUT("/status/:deviceType/:deviceId", saola.Apply(
		service.NewSetDeviceStatus(exec, properties, redisPool),
		util.NewContextLoggerFilter(),
		util.NewResponseStatsFilter(statsdClient),
		util.NewErrorResponseFilter(),
		util.NewErrorLoggerFilter()))

	endpoint.GET("/users/:userId", saola.Apply(
		service.NewGetUserDevices(exec, properties, redisPool),
		util.NewContextLoggerFilter(),
		util.NewResponseStatsFilter(statsdClient),
		util.NewErrorResponseFilter(),
		util.NewErrorLoggerFilter()))

	go func() {
		http.Handle("/cuirass.stream", metricsstream.NewMetricsStream(exec))
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	log.Fatal(httpservice.Serve(":10000", endpoint))
}
