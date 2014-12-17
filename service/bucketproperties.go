package service

import (
	"time"

	"github.com/arjantop/vaquita"
)

type BucketProperties struct {
	NumberOfBuckets vaquita.IntProperty
	BucketSize      vaquita.DurationProperty
}

func NewBucketProperties(f *vaquita.PropertyFactory) *BucketProperties {
	return &BucketProperties{
		NumberOfBuckets: f.GetIntProperty("protogalaxy.devicepresence.buckets.number", 5),
		BucketSize:      f.GetDurationProperty("protogalaxy.devicepresence.buckets.size", time.Minute, time.Second),
	}
}

func (p *BucketProperties) Ttl() time.Duration {
	return time.Duration(p.NumberOfBuckets.Get()) * p.BucketSize.Get()
}
