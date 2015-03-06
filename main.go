package main

import (
	"flag"
	"math/rand"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/golang/glog"
	"github.com/protogalaxy/service-device-presence/devicepresence"
	"google.golang.org/grpc"
)

import (
	"net"
	_ "net/http/pprof"
)

func DoPing(c redis.Conn) error {
	_, err := c.Do("PING")
	return err
}

func main() {
	flag.Parse()
	rand.Seed(time.Now().UnixNano())

	redisPool := &redis.Pool{
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
	defer redisPool.Close()

	socket, err := net.Listen("tcp", ":9090")
	if err != nil {
		glog.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	devicepresence.RegisterPresenceManagerServer(grpcServer, &devicepresence.Manager{
		Redis: redisPool,
	})
	grpcServer.Serve(socket)
}
