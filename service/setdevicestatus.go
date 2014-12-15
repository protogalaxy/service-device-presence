package service

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"code.google.com/p/go.net/context"
	"github.com/101loops/clock"
	"github.com/arjantop/cuirass"
	"github.com/garyburd/redigo/redis"
	"github.com/julienschmidt/httprouter"
	"github.com/protogalaxy/service-device-presence/device"
	"github.com/protogalaxy/service-device-presence/util"
)

func NewRedisSetDeviceStatusCommand(pool *redis.Pool, dev *device.Device, status *device.DeviceStatus) *cuirass.Command {
	return cuirass.NewCommand("RedisSetDeviceStatus", func(ctx context.Context) (interface{}, error) {
		c := make(chan error, 1)
		go func() {
			c <- func() error {
				conn := pool.Get()
				defer conn.Close()

				bucketKey := util.CurrentBucket(clock.Work, status.UserId, util.DefaultBucketSize)

				deviceString := dev.String()
				var err error
				if status.Status == device.StatusOnline {
					conn.Send("MULTI")
					conn.Send("SADD", bucketKey, deviceString)
					conn.Send("EXPIRE", bucketKey, 5*60)
					conn.Send("SET", deviceString, status.UserId)
					conn.Send("EXPIRE", deviceString, 5*60)
					_, err = conn.Do("EXEC")
				} else {
					_, err = conn.Do("DEL", deviceString)
				}
				return err
			}()
		}()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case err := <-c:
			return nil, err
		}
	}).Build()
}

func ExecRedisSetDeviceStatusCommand(
	exec *cuirass.Executor,
	ctx context.Context,
	cmd *cuirass.Command) error {

	_, err := exec.Exec(ctx, cmd)
	return err
}

type SetDeviceStatusService struct {
	redisPool *redis.Pool
	exec      *cuirass.Executor
}

func NewSetDeviceStatus(exec *cuirass.Executor, rp *redis.Pool) *SetDeviceStatusService {
	return &SetDeviceStatusService{
		redisPool: rp,
		exec:      exec,
	}
}

func (h *SetDeviceStatusService) ServeHTTP(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	dev := device.Device{ps.ByName("deviceType"), ps.ByName("deviceId")}

	decoder := json.NewDecoder(r.Body)
	var deviceStatus device.DeviceStatus
	err := decoder.Decode(&deviceStatus)
	if err != nil {
		log.Println(err)
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	log.Printf("Setting device status %s for user %s to %s", dev.String(), deviceStatus.UserId, deviceStatus.Status)

	cmd := NewRedisSetDeviceStatusCommand(h.redisPool, &dev, &deviceStatus)
	err = ExecRedisSetDeviceStatusCommand(h.exec, ctx, cmd)
	if err != nil {
		log.Println(err)
		http.Error(w, "error settind device name", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}\n"))
	if err != nil {
		log.Println(err)
	}
}
