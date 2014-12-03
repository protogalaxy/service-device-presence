package service

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"code.google.com/p/go.net/context"
	"github.com/arjantop/cuirass"
	"github.com/garyburd/redigo/redis"
	"github.com/julienschmidt/httprouter"
	"github.com/protogalaxy/service-device-presence/device"
)

func NewRedisGetUserDevicesCommand(pool *redis.Pool, userId string) *cuirass.Command {
	return cuirass.NewCommand("RedisGetUserDevices", func(ctx context.Context) (r interface{}, err error) {
		c := make(chan error, 1)
		go func() {
			c <- func() error {
				conn := pool.Get()
				defer conn.Close()

				deviceList, err := redis.Strings(conn.Do("SUNION", fmt.Sprintf("%s:%s", userId, "bucket")))
				if err != nil {
					return err
				}

				devices := make([]*device.Device, 0)
				for _, deviceString := range deviceList {
					deviceExists, err := redis.Bool(conn.Do("EXISTS", deviceString))
					if err != nil {
						return err
					}
					if deviceExists {
						deviceParts := strings.SplitN(deviceString, ":", 2)
						if len(deviceParts) != 2 {
							log.Println("Invalid device string")
							continue
						}
						devices = append(devices, &device.Device{
							DeviceType: deviceParts[0],
							DeviceId:   deviceParts[1],
						})
					}
				}
				r = devices
				return nil
			}()
		}()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case err := <-c:
			return r, err
		}
	}).Build()
}

func ExecRedisGetUserDevicesCommand(
	exec *cuirass.Executor,
	ctx context.Context,
	cmd *cuirass.Command) ([]*device.Device, error) {

	devices, err := exec.Exec(ctx, cmd)
	return devices.([]*device.Device), err
}

type GetUserDevicesService struct {
	redisPool *redis.Pool
	exec      *cuirass.Executor
}

func NewGetUserDevices(exec *cuirass.Executor, rp *redis.Pool) *GetUserDevicesService {
	return &GetUserDevicesService{
		redisPool: rp,
		exec:      exec,
	}
}

func (h *GetUserDevicesService) ServeHTTP(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	userId := ps.ByName("userId")

	log.Printf("Getting devices for user %s", userId)

	cmd := NewRedisGetUserDevicesCommand(h.redisPool, userId)
	userDevices, err := ExecRedisGetUserDevicesCommand(h.exec, ctx, cmd)
	if err != nil {
		log.Println(err)
		http.Error(w, "error getting devices for user", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	result := struct {
		UserId  string           `json:"user_id"`
		Devices []*device.Device `json:"devices"`
	}{
		userId,
		userDevices,
	}
	err = encoder.Encode(&result)
	if err != nil {
		log.Println(err)
	}
}
