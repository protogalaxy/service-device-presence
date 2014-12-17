package service

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"code.google.com/p/go.net/context"
	"github.com/101loops/clock"
	"github.com/arjantop/cuirass"
	"github.com/garyburd/redigo/redis"
	"github.com/julienschmidt/httprouter"
	"github.com/protogalaxy/service-device-presence/device"
	"github.com/protogalaxy/service-device-presence/util"
)

func NewRedisGetUserDevicesCommand(pool *redis.Pool, properties *BucketProperties, userId string) *cuirass.Command {
	return cuirass.NewCommand("RedisGetUserDevices", func(ctx context.Context) (r interface{}, err error) {
		c := make(chan error, 1)
		go func() {
			c <- func() error {
				conn := pool.Get()
				defer conn.Close()

				bucketKeys := util.BucketRange(clock.New(), userId, properties.BucketSize.Get(), -properties.NumberOfBuckets.Get(), 0)
				deviceList, err := redis.Strings(conn.Do("SUNION", toInterfaceSlice(bucketKeys)...))
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
	devices, _ = devices.([]*device.Device)
	return devices.([]*device.Device), err
}

func toInterfaceSlice(buckets []util.Bucket) []interface{} {
	result := make([]interface{}, 0, len(buckets))
	for _, b := range buckets {
		result = append(result, b.String())
	}
	return result
}

type GetUserDevicesService struct {
	redisPool  *redis.Pool
	properties *BucketProperties
	exec       *cuirass.Executor
}

func NewGetUserDevices(exec *cuirass.Executor, properties *BucketProperties, rp *redis.Pool) *GetUserDevicesService {
	return &GetUserDevicesService{
		redisPool:  rp,
		properties: properties,
		exec:       exec,
	}
}

func (h *GetUserDevicesService) ServeHTTP(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	userId := ps.ByName("userId")

	log.Printf("Getting devices for user %s", userId)

	cmd := NewRedisGetUserDevicesCommand(h.redisPool, h.properties, userId)
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
