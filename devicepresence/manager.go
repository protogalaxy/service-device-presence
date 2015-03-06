// Copyright (C) 2015 The Protogalaxy Project
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

//go:generate protoc --go_out=plugins=grpc:. -I ../protos ../protos/devicepresence.proto

package devicepresence

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/101loops/clock"
	"github.com/garyburd/redigo/redis"
	"github.com/protogalaxy/service-device-presence/util"
	"golang.org/x/net/context"
)

const (
	BucketSize = time.Minute
	NumBuckets = 5
)

type Manager struct {
	Redis *redis.Pool
}

func (m *Manager) SetStatus(ctx context.Context, req *StatusRequest) (*StatusReply, error) {
	if err := validateRequest(req); err != nil {
		return nil, err
	}

	conn := m.Redis.Get()
	defer conn.Close()

	bucketKey := util.CurrentBucket(clock.New(), req.Device.UserId, BucketSize)

	deviceKey := makeKey(req.Device)
	var err error
	if req.Device.Status == Device_ONLINE {
		conn.Send("MULTI")
		conn.Send("SADD", bucketKey, deviceKey)
		conn.Send("EXPIRE", bucketKey, int(BucketSize))
		conn.Send("SET", deviceKey, req.Device.UserId)
		conn.Send("EXPIRE", deviceKey, int(deviceTTL(BucketSize, NumBuckets)))
		_, err = conn.Do("EXEC")
	} else {
		_, err = conn.Do("DEL", deviceKey)
	}

	if err != nil {
		return nil, err
	}

	var res StatusReply
	return &res, nil
}

func validateRequest(req *StatusRequest) error {
	if req.Device == nil {
		return errors.New("missing device")
	}
	if req.Device.Id == "" {
		return errors.New("missing device id")
	}
	if req.Device.UserId == "" {
		return errors.New("missing user id")
	}
	return nil
}

func makeKey(d *Device) string {
	return fmt.Sprintf("%s:%s", d.Type, d.Id)
}

func deviceTTL(bs time.Duration, n int) time.Duration {
	return time.Duration(n) * bs
}

func (m *Manager) GetDevices(req *DevicesRequest, stream PresenceManager_GetDevicesServer) error {
	if req.UserId == "" {
		return errors.New("missing user id")
	}
	conn := m.Redis.Get()
	defer conn.Close()

	bucketKeys := util.BucketRange(clock.New(), req.UserId, BucketSize, -NumBuckets, 0)
	deviceList, err := redis.Strings(conn.Do("SUNION", toInterfaceSlice(bucketKeys)...))
	if err != nil {
		return err
	}

	for _, deviceKey := range deviceList {
		userID, err := redis.String(conn.Do("GET", deviceKey))
		if err != nil && err != redis.ErrNil {
			return fmt.Errorf("getting device status: %s", err)
		}

		if userID != "" {
			device, err := parseDevice(userID, deviceKey)
			if err != nil {
				return fmt.Errorf("parsing device: %s", err)
			}

			if err := stream.Send(device); err != nil {
				return err
			}
		}
	}
	return nil
}

func toInterfaceSlice(buckets []util.Bucket) []interface{} {
	result := make([]interface{}, 0, len(buckets))
	for _, b := range buckets {
		result = append(result, b.String())
	}
	return result
}

func parseDevice(userID, deviceKey string) (*Device, error) {
	deviceParts := strings.SplitN(deviceKey, ":", 2)
	if len(deviceParts) != 2 {
		return nil, errors.New("invalid device string")
	}

	dt, err := parseDeviceType(deviceParts[0])
	if err != nil {
		return nil, err
	}

	device := &Device{
		Id:     deviceParts[1],
		Type:   dt,
		UserId: userID,
		Status: Device_ONLINE,
	}
	return device, nil
}

func parseDeviceType(t string) (Device_Type, error) {
	if dt, ok := Device_Type_value[t]; ok {
		return Device_Type(dt), nil
	}
	return 0, errors.New("invalid device type")
}
