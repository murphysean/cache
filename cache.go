package cache

import (
	"errors"
	"time"
)

var (
	ErrNotPresent = errors.New("cache: Value not present")
)

type ValueLoader func(key string) (interface{}, error)
type Weigher func(key string, value interface{}) int64
type Comparer func(weighta, weightb int64, agea, ageb time.Duration) int64

type Cache interface {
	GetWithValueLoader(key string, valueLoader ValueLoader) (interface{}, error)
	GetIfPresent(key string) (interface{}, error)
	//GetAllPresent(keys []string) map[string]interface{}
	Put(key string, value interface{})
	//PutAll(map[string]interface{})
	Invalidate(key string)
	//InvalidateKeys(keys []string)
	InvalidateAll()
	//AsMap() map[string]interface{}
	CleanUp()
	//Size() int64
	//Stats()
}

type LoadingCache interface {
	Cache
	Get(key string) (interface{}, error)
	//GetAll(keys []string) map[string]interface{}
	Refresh(key string)
	Load(key string) (interface{}, error)
	//LoadAll(keys []string) error
	//Reload(key string, oldValue interface{}) error
}

type ExpiringCache interface {
	SetExpiresAt(key string, expires time.Time)
	SetExpiresIn(key string, expiresIn time.Duration)
}

type PriorityCache interface {
	SetWeight(key string, weight int64)
}

type StatsCache interface {
	HitRate() float64
	AverageLoadPenalty() time.Duration
	EvictionCount() int64
}

func NewExpiresAfterAccessCache(accessDuration time.Duration) Cache {
	c := new(PowerCache)
	c.ExpiresAfterAccessDuration = accessDuration
	c.PeriodicMaintenance = time.Minute * 5
	c.Initialize()
	return c
}

func NewExpiresAfterWriteCache(writeDuration time.Duration) Cache {
	c := new(PowerCache)
	c.ExpiresAfterWriteDuration = writeDuration
	c.PeriodicMaintenance = time.Minute * 5
	c.Initialize()
	return c
}

func NewMaxKeysCache(maxKeys int) Cache {
	c := new(PowerCache)
	c.MaxKeys = maxKeys
	c.Initialize()
	return c
}

func NewPowerCache() *PowerCache {
	c := new(PowerCache)
	c.Initialize()
	return c
}
