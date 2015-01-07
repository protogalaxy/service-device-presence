package service

import (
	"encoding/json"
	"net/http"

	"github.com/101loops/clock"
	"github.com/arjantop/cuirass"
	"github.com/arjantop/cuirass/util/contextutil"
	"github.com/arjantop/saola/httpservice"
	"github.com/arjantop/saola/redisservice"
	"github.com/protogalaxy/service-device-presence/device"
	"github.com/protogalaxy/service-device-presence/util"
	"golang.org/x/net/context"
)

func NewRedisSetDeviceStatusCommand(
	pool *redisservice.Pool,
	properties *BucketProperties,
	dev *device.Device,
	status *device.DeviceStatus) *cuirass.Command {

	return cuirass.NewCommand("RedisSetDeviceStatus", func(ctx context.Context) (interface{}, error) {
		err := contextutil.Do(ctx, func() error {
			conn := pool.Get()
			defer conn.Close()

			bucketKey := util.CurrentBucket(clock.New(), status.UserId, properties.BucketSize.Get())

			deviceString := dev.String()
			var err error
			if status.Status == device.StatusOnline {
				ttl := int(properties.Ttl().Seconds())
				conn.Send(ctx, "MULTI")
				conn.Send(ctx, "SADD", bucketKey, deviceString)
				conn.Send(ctx, "EXPIRE", bucketKey, ttl)
				conn.Send(ctx, "SET", deviceString, status.UserId)
				conn.Send(ctx, "EXPIRE", deviceString, ttl)
				_, err = conn.Do(ctx, "EXEC")
			} else {
				_, err = conn.Do(ctx, "DEL", deviceString)
			}
			return err
		})
		return nil, err
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
	redisPool  *redisservice.Pool
	properties *BucketProperties
	exec       *cuirass.Executor
}

func NewSetDeviceStatus(exec *cuirass.Executor, properties *BucketProperties, rp *redisservice.Pool) *SetDeviceStatusService {
	return &SetDeviceStatusService{
		redisPool:  rp,
		properties: properties,
		exec:       exec,
	}
}

func (h *SetDeviceStatusService) DoHTTP(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	logger := util.GetContextLogger(ctx)
	ps := httpservice.GetParams(ctx)
	dev := device.Device{
		DeviceType: ps.Get("deviceType"),
		DeviceId:   ps.Get("deviceId"),
	}

	decoder := json.NewDecoder(r.Body)
	var deviceStatus device.DeviceStatus
	err := decoder.Decode(&deviceStatus)
	if err != nil {
		return util.NewCustomError(http.StatusBadRequest, "problem decoding request body", err,
			util.Data{"device_type": dev.DeviceType, "device_id": dev.DeviceId})
	}

	logger.Info("Setting device status",
		"device", dev.String(),
		"user_id", deviceStatus.UserId,
		"device_status", deviceStatus.Status)

	cmd := NewRedisSetDeviceStatusCommand(h.redisPool, h.properties, &dev, &deviceStatus)
	err = ExecRedisSetDeviceStatusCommand(h.exec, ctx, cmd)
	if err != nil {
		return util.NewErrorResponse("problem setting device status", err,
			util.Data{"device_type": dev.DeviceType, "device_id": dev.DeviceId})
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{}\n"))
	return nil
}

func (h *SetDeviceStatusService) Do(ctx context.Context) error {
	return httpservice.Do(h, ctx)
}

func (h *SetDeviceStatusService) Name() string {
	return "setdevicestatus"
}
