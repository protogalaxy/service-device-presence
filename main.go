package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"os"
	"strings"
	"time"

	"github.com/protogalaxy/service-device-presence/Godeps/_workspace/src/github.com/Shopify/sarama"
	"github.com/protogalaxy/service-device-presence/Godeps/_workspace/src/github.com/garyburd/redigo/redis"
	"github.com/protogalaxy/service-device-presence/Godeps/_workspace/src/github.com/golang/glog"
	"github.com/protogalaxy/service-device-presence/Godeps/_workspace/src/google.golang.org/grpc"
	"github.com/protogalaxy/service-device-presence/devicepresence"
)

func DoPing(c redis.Conn) error {
	_, err := c.Do("PING")
	return err
}

var port = flag.Int("port", 9090, "port to listen on")

func ParseLinkEnv(name string) string {
	v := os.Getenv(name + "_PORT")
	if v == "" {
		glog.Fatalf("Missing environment variable %s_PORT", name)
	}
	connStr := strings.Split(v, "//")[1]
	return connStr
}

func main() {
	flag.Parse()
	rand.Seed(time.Now().UnixNano())

	redisPool := &redis.Pool{
		MaxIdle:     50,
		IdleTimeout: time.Minute,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", ParseLinkEnv("REDIS"))
			if err != nil {
				return nil, err
			}
			return c, err
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			return DoPing(c)
		},
	}
	defer redisPool.Close()

	cfg := sarama.NewConfig()
	cfg.ClientID = "service-device-presence"
	producer, err := sarama.NewSyncProducer([]string{ParseLinkEnv("KAFKA")}, cfg)
	if err != nil {
		glog.Fatalf("Unable to connect to kafka: %s", err)
	}
	defer func() {
		if err := producer.Close(); err != nil {
			glog.Errorf("Error closing producer: %s", err)
		}
	}()

	socket, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		glog.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	devicepresence.RegisterPresenceManagerServer(grpcServer, &devicepresence.Manager{
		Redis:  redisPool,
		Stream: producer,
	})
	grpcServer.Serve(socket)
}
