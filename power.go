package cache

import (
	"sync"
	"time"
)

var emptyDuration time.Duration

type PowerCache struct {
	ValueLoader                ValueLoader
	ExpiresAfterAccessDuration time.Duration
	ExpiresAfterWriteDuration  time.Duration
	PeriodicMaintenance        time.Duration
	MaxKeys                    int
	MaxWeight                  int64
	MaxSize                    int64
	DefaultValueWeight         int64

	mu           sync.Mutex
	values       map[string]interface{}
	tstamp       map[string]time.Time
	weight       map[string]int64
	cacheSizeEst int64
	nextClean    time.Time

	statLoadCount int64
	statLoadDur   time.Duration
	statHits      int64
	statReqs      int64
	statEvictions int64
}

func (c *PowerCache) Initialize() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values = make(map[string]interface{})
	c.tstamp = make(map[string]time.Time)
	c.weight = make(map[string]int64)
	if c.DefaultValueWeight == 0 {
		c.DefaultValueWeight = 1
	}
	if c.PeriodicMaintenance != emptyDuration {
		c.nextClean = time.Now().Add(c.PeriodicMaintenance)
	}

	c.statLoadCount = 0
	c.statHits = 0
	c.statReqs = 0
	c.statEvictions = 0
}

func (c *PowerCache) Length() int {
	return len(c.values)
}

func (c *PowerCache) cleanUpIfNeccissary() {
	c.mu.Lock()
	shouldClean := false
	//Do periodic maintenence if this is a time based cache
	if c.PeriodicMaintenance != emptyDuration {
		if c.nextClean.After(time.Now()) {
			shouldClean = true
		}
	}
	//If maxkeys is set and we are at (or possibly approaching) the limit, clean
	if c.MaxKeys != 0 && len(c.values) >= c.MaxKeys {
		shouldClean = true
	}
	//If maxsize is set and we are at (or possibly approaching) the limit, clean
	if c.MaxSize != 0 && c.cacheSizeEst >= c.MaxSize {
		shouldClean = true
	}
	c.mu.Unlock()
	//Clean
	if shouldClean {
		c.CleanUp()
	}
}

func (c *PowerCache) Put(key string, value interface{}) {
	c.cleanUpIfNeccissary()
	c.mu.Lock()
	defer c.mu.Unlock()
	c.values[key] = value
	if c.ExpiresAfterWriteDuration == emptyDuration && c.ExpiresAfterAccessDuration == emptyDuration {
		c.tstamp[key] = time.Now()
	}
	if c.ExpiresAfterWriteDuration != emptyDuration {
		c.tstamp[key] = time.Now().Add(c.ExpiresAfterWriteDuration)
	}
	if c.ExpiresAfterAccessDuration != emptyDuration {
		c.tstamp[key] = time.Now().Add(c.ExpiresAfterAccessDuration)
	}
	//Put in the weight
	c.weight[key] = c.DefaultValueWeight
	//TODO Use the weight calculator function if it's available
}

func (c *PowerCache) loadWithValueLoader(key string, valueLoader ValueLoader) (interface{}, error) {
	start := time.Now()
	value, err := valueLoader(key)
	if err != nil {
		return nil, err
	}
	loaddur := time.Now().Sub(start)
	//Update Average Load Duration
	c.mu.Lock()
	c.statLoadCount++
	c.statLoadDur = (c.statLoadDur + loaddur) / time.Duration(c.statLoadCount)
	c.mu.Unlock()
	c.Put(key, value)
	return value, nil
}

func (c *PowerCache) Refresh(key string) {
	c.loadWithValueLoader(key, c.ValueLoader)
}

func (c *PowerCache) Load(key string) (interface{}, error) {
	return c.GetWithValueLoader(key, c.ValueLoader)
}

func (c *PowerCache) GetIfPresent(key string) (interface{}, error) {
	if c.isKeyExpired(key) {
		c.Invalidate(key)
	}
	c.mu.Lock()
	c.statReqs++
	c.mu.Unlock()
	if v, ok := c.values[key]; ok {
		if c.ExpiresAfterWriteDuration == emptyDuration && c.ExpiresAfterAccessDuration == emptyDuration {
			c.tstamp[key] = time.Now()
		} else if c.ExpiresAfterAccessDuration != emptyDuration {
			c.tstamp[key] = time.Now().Add(c.ExpiresAfterAccessDuration)
		}
		c.mu.Lock()
		c.statHits++
		c.mu.Unlock()
		return v, nil
	} else {
		return nil, ErrNotPresent
	}
}

func (c *PowerCache) Get(key string) (interface{}, error) {
	return c.GetWithValueLoader(key, c.ValueLoader)
}

func (c *PowerCache) GetWithValueLoader(key string, valueLoader ValueLoader) (interface{}, error) {
	v, err := c.GetIfPresent(key)
	if err == nil {
		return v, err
	}
	return c.loadWithValueLoader(key, valueLoader)
}

func (c *PowerCache) isKeyExpired(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ExpiresAfterAccessDuration != emptyDuration {
		if a, ok := c.tstamp[key]; ok {
			if time.Now().After(a.Add(c.ExpiresAfterAccessDuration)) {
				return true
			}
		}
	}
	if c.ExpiresAfterWriteDuration != emptyDuration {
		if w, ok := c.tstamp[key]; ok {
			if time.Now().After(w.Add(c.ExpiresAfterWriteDuration)) {
				return true
			}
		}
	}
	return false
}

func (c *PowerCache) Invalidate(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.values, key)
	delete(c.tstamp, key)
	delete(c.weight, key)
	c.statEvictions++
}

func (c *PowerCache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.statEvictions += int64(len(c.values))
	c.values = make(map[string]interface{})
	c.tstamp = make(map[string]time.Time)
	c.weight = make(map[string]int64)
}

//The CacheMap CleanUp function has a few different eviction behaviors
//- Time Based Eviction
// - If the key is expired it will be evicted
// - If maxKeys is set then the algorithm will search for a value to evict
//  - It will stop if it finds at least one expired key to evict
//  - If not it will settle with the LRU (oldest key)
// - If maxSize is set then the algorithm will search for a value to evict
//  - It will stop if it finds at least one expired key to evict
//  - If not it will find the oldest and largest key to remove by calculating a weight
//- Size Based Eviction
// - Will try to find oldest and largest key to remove by calculating a weight
func (c *PowerCache) CleanUp() {
	c.mu.Lock()
	defer c.mu.Unlock()
	var aKey string
	var aWeight int64
	var aTstamp time.Time
	for k, _ := range c.values {
		if c.ExpiresAfterWriteDuration != emptyDuration ||
			c.ExpiresAfterAccessDuration != emptyDuration {

			if c.tstamp[k].Before(time.Now()) {
				delete(c.values, k)
				delete(c.tstamp, k)
				delete(c.weight, k)
				c.statEvictions++
				if c.MaxSize != 0 || c.MaxKeys != 0 {
					break
				} else {
					continue
				}
			}
		}

		if aKey == "" {
			aKey = k
			aWeight = c.weight[k]
			aTstamp = c.tstamp[k]
			continue
		}

		//Find out relative weights
		mWeight := aWeight
		if mWeight < c.weight[k] {
			mWeight = c.weight[k]
		}
		awf := float64(aWeight) / float64(mWeight)
		bwf := float64(c.weight[k]) / float64(mWeight)
		//fmt.Println("Weight: ", awf, bwf)
		now := time.Now()
		adf := 0.0
		bdf := 0.0

		if c.ExpiresAfterAccessDuration != emptyDuration || c.ExpiresAfterAccessDuration != emptyDuration {
			//In this case if this key were expired it would have been removed
			mTstamp := aTstamp
			if mTstamp.Before(c.tstamp[k]) {
				mTstamp = c.tstamp[k]
			}
			adf = float64(aTstamp.Sub(now)) / float64(mTstamp.Sub(now))
			bdf = float64(c.tstamp[k].Sub(now)) / float64(mTstamp.Sub(now))
			//fmt.Println("Expires: ", adf, bdf)
		} else {
			//In this case all the expires will just be a timestamp of write
			//Therefor the smaller the better
			//We will do durations of both parties
			ad := now.Sub(aTstamp)
			bd := now.Sub(c.tstamp[k])
			md := ad
			if ad < bd {
				md = bd
			}
			adf = float64(ad) / float64(md)
			bdf = float64(bd) / float64(md)
			//Now do the multiplactive inverse of each
			adf = 1.0 / adf
			bdf = 1.0 / bdf
			//fmt.Println("Accessed: ", adf, bdf)
		}

		//Calculate the scores, higher wins
		ascore := awf*0.5 + adf*0.5
		bscore := bwf*0.5 + bdf*0.5

		if bscore < ascore {
			aKey = k
			aWeight = c.weight[k]
			aTstamp = c.tstamp[k]
		}
	}
	//Now I've gone through, none were immediate canidates for cleaning, so we'll go with our worst guy
	//fmt.Println("Cleaning: ", aKey)
	delete(c.values, aKey)
	delete(c.tstamp, aKey)
	delete(c.weight, aKey)
	c.statEvictions++

	//Now set the time for the next cleaning
	if c.PeriodicMaintenance != emptyDuration {
		c.nextClean = time.Now().Add(c.PeriodicMaintenance)
	}
}

func (c *PowerCache) SetExpiresAt(key string, expires time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	//TODO Make sure that the cache is set up for future expires
	//TODO Make sure that the expires is in the future
	c.tstamp[key] = expires
}

func (c *PowerCache) SetExpiresIn(key string, expiresIn time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	//TODO Make sure thtat the cache is set up for future expires
	//TODO Make sure that the expires is in the future
	c.tstamp[key] = time.Now().Add(expiresIn)
}

func (c *PowerCache) SetWeight(key string, weight int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	//TODO Make sure the weight is not above the maxweight
	c.weight[key] = weight
}

func (c *PowerCache) HitRate() float64 {
	if c.statReqs == 0 {
		return 0.0
	}
	return float64(c.statHits) / float64(c.statReqs)
}

func (c *PowerCache) AverageLoadPenalty() time.Duration {
	return c.statLoadDur
}

func (c *PowerCache) EvictionCount() int64 {
	return c.statEvictions
}
