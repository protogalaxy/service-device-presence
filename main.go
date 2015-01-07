package main

import (
	"log"
	"math/rand"
	"time"

	"github.com/arjantop/cuirass"
	"github.com/arjantop/cuirass/metricsstream"
	"github.com/arjantop/saola"
	"github.com/arjantop/saola/httpservice"
	"github.com/arjantop/saola/redisservice"
	"github.com/arjantop/vaquita"
	"github.com/cactus/go-statsd-client/statsd"
	"github.com/garyburd/redigo/redis"
	"github.com/protogalaxy/service-device-presence/service"
	"github.com/protogalaxy/service-device-presence/stats"
	"github.com/protogalaxy/service-device-presence/util"
	"github.com/wadey/go-zipkin"
)

import (
	"net/http"
	_ "net/http/pprof"
)

func DoPing(c redis.Conn) error {
	_, err := c.Do("PING")
	return err
}

func main() {
	rand.Seed(time.Now().UnixNano())
	config := vaquita.NewEmptyMapConfig()
	config.SetProperty("cuirass.command.RedisSetDeviceStatus.execution.isolation.thread.timeoutInMilliseconds", "30")
	config.SetProperty("cuirass.command.RedisGetUserDevices.execution.isolation.thread.timeoutInMilliseconds", "50")

	propertyFactory := vaquita.NewPropertyFactory(config)
	properties := service.NewBucketProperties(propertyFactory)
	exec := cuirass.NewExecutor(config)

	statsdClient, _ := statsd.Dial("localhost:8125", "protogalaxy.service.devicepresence")
	defer statsdClient.Close()
	statsReceiver := stats.NewStatsdStatsReceiver(statsdClient, 0.01)

	scribe := []zipkin.SpanCollector{zipkin.NewScribeCollector("localhost:9410")}

	newRedisPool := &redisservice.Pool{
		Filter:      util.NewRedisTracingFilter(6379, scribe),
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
	defer newRedisPool.Close()

	endpoint := httpservice.NewEndpoint()

	endpoint.PUT("/status/:deviceType/:deviceId", saola.Apply(
		service.NewSetDeviceStatus(exec, properties, newRedisPool),
		httpservice.NewCancellationFilter(),
		util.NewContextLoggerFilter(),
		util.NewServerTracingFilter(10000, scribe),
		saola.NewStatsFilter(statsReceiver),
		httpservice.NewResponseStatsFilter(statsReceiver),
		util.NewErrorResponseFilter(),
		util.NewErrorLoggerFilter()))

	endpoint.GET("/users/:userId", saola.Apply(
		service.NewGetUserDevices(exec, properties, newRedisPool),
		httpservice.NewCancellationFilter(),
		util.NewContextLoggerFilter(),
		util.NewServerTracingFilter(10000, scribe),
		saola.NewStatsFilter(statsReceiver),
		httpservice.NewResponseStatsFilter(statsReceiver),
		util.NewErrorResponseFilter(),
		util.NewErrorLoggerFilter()))

	go func() {
		http.Handle("/cuirass.stream", metricsstream.NewMetricsStream(exec))
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	log.Fatal(httpservice.Serve(":10000", saola.Apply(
		endpoint,
		httpservice.NewStdRequestLogFilter())))
}
