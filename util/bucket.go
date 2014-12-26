package util

import (
	"strconv"
	"time"

	"github.com/101loops/clock"
)

type Bucket struct {
	prefix     string
	bucketSize time.Duration
	timestamp  time.Time
}

func (b Bucket) String() string {
	return b.prefix + ":" + strconv.FormatInt(b.timestamp.Unix()/int64(b.bucketSize.Seconds()), 10)
}

func CurrentBucket(c clock.Clock, prefix string, bucketSize time.Duration) Bucket {
	return BucketWithOffset(c, prefix, bucketSize, 0)
}

func BucketWithOffset(c clock.Clock, prefix string, bucketSize time.Duration, offset int) Bucket {
	if bucketSize <= time.Second {
		bucketSize = time.Second
	}
	return Bucket{
		prefix:     prefix,
		bucketSize: bucketSize,
		timestamp:  c.Now().Add(time.Duration(offset) * bucketSize),
	}
}

func BucketRange(c clock.Clock, prefix string, bucketSize time.Duration, offsetStart int, offsetEnd int) []Bucket {
	buckets := make([]Bucket, 0, offsetEnd-offsetStart)
	for i := offsetStart; i <= offsetEnd; i++ {
		buckets = append(buckets, BucketWithOffset(c, prefix, bucketSize, i))
	}
	return buckets
}
