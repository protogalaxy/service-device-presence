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

package util

import (
	"strconv"
	"time"

	"github.com/protogalaxy/service-device-presence/Godeps/_workspace/src/github.com/101loops/clock"
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
