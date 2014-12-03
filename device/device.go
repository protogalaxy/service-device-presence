package device

import "fmt"

type Status string

const (
	StatusOnline  = Status("online")
	StatusOffline = Status("offline")
)

func (s *Status) UnmarshalJSON(v []byte) error {
	switch string(v[1 : len(v)-1]) {
	case "online":
		*s = StatusOnline
		return nil
	case "offline":
		*s = StatusOffline
		return nil
	}
	return fmt.Errorf("unknown device status: %s", string(v))
}

type Device struct {
	DeviceType string `json:"device_type"`
	DeviceId   string `json:"device_id"`
}

func (d *Device) String() string {
	return fmt.Sprintf("%s:%s", d.DeviceType, d.DeviceId)
}

type DeviceStatus struct {
	UserId string `json:"user_id"`
	Status Status `json:"status"`
}
