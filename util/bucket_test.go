package util_test

import (
	"testing"
	"time"

	"github.com/101loops/clock"
	"github.com/protogalaxy/service-device-presence/util"
	"github.com/stretchr/testify/assert"
)

func TestCurrentBucket(t *testing.T) {
	c := clock.NewMock()
	c.Set(time.Unix(3600, 0))
	b := util.CurrentBucket(c, "prefix", time.Hour)
	assert.Equal(t, "prefix:1", b.String())

	c.Add(time.Hour)
	b = util.CurrentBucket(c, "prefix", time.Hour)
	assert.Equal(t, "prefix:2", b.String())
}

func TestCurrentBucketDifferentBucketSizes(t *testing.T) {
	c := clock.NewMock()
	c.Set(time.Unix(3600, 0))
	b := util.CurrentBucket(c, "p", time.Minute)
	assert.Equal(t, "p:60", b.String())

	b = util.CurrentBucket(c, "p", time.Second)
	assert.Equal(t, "p:3600", b.String())
}

func TestBucketWithOffset(t *testing.T) {
	c := clock.NewMock()
	c.Set(time.Unix(3600, 0))
	b := util.BucketWithOffset(c, "p", time.Second, 0)
	assert.Equal(t, "p:3600", b.String())

	b = util.BucketWithOffset(c, "p", time.Second, 1)
	assert.Equal(t, "p:3601", b.String())

	b = util.BucketWithOffset(c, "p", time.Second, -1)
	assert.Equal(t, "p:3599", b.String())
}

func TestBucketWithZeroOrNegativeDuration(t *testing.T) {
	c := clock.NewMock()
	c.Set(time.Unix(5, 0))
	b := util.CurrentBucket(c, "p", 0)
	assert.Equal(t, "p:5", b.String())

	b = util.CurrentBucket(c, "p", -1)
	assert.Equal(t, "p:5", b.String())
}

func TestBucketRange(t *testing.T) {
	c := clock.NewMock()
	c.Set(time.Unix(5, 0))
	buckets := util.BucketRange(c, "p", time.Second, -1, 1)
	assert.Equal(t, "p:4", buckets[0].String())
	assert.Equal(t, "p:5", buckets[1].String())
	assert.Equal(t, "p:6", buckets[2].String())
}
