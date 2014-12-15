package util

import (
	"fmt"
	"time"

	"github.com/101loops/clock"
)

const DefaultBucketSize = time.Minute

type Bucket struct {
	prefix     string
	bucketSize time.Duration
	timestamp  time.Time
}

func (b Bucket) String() string {
	return fmt.Sprintf("%s:%d", b.prefix, b.timestamp.Unix()/int64(b.bucketSize.Seconds()))
}

func CurrentBucket(c clock.Clock, prefix string, bucketSize time.Duration) Bucket {
	return BucketWithOffset(c, prefix, bucketSize, 0)
}

func BucketWithOffset(c clock.Clock, prefix string, bucketSize time.Duration, offset int) Bucket {
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
