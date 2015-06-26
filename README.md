Cache
###

Cache is an attempt to copy and improve the java guava cache library. It uses
a go map under the hood to store unstructured data. It defines a few interfaces
around caches, and then declares a single implementation.

Cache provides 2 of the 3 basic types of eviction:

1. Size-Based
1. Time-Based
1. ~~Reference-Based~~

Size-Based Eviction
---

The primary type of size based eviction is based on key count. You can create
a cache like so:

	c := cache.NewMaxKeysCache(1024)

Time-Based Eviction
---

Cache will expire keys after either write or access. 

### Write Expiration

Write expiration will keep track of when a key was created or replaced. This
type of cache would be useful if your data grows stale after a certain amount
of time.

	c := cache.NewExpiresAfterWriteCache(time.Minute * 5)

### Access Expiration

Access based expiration will keep track of the last time a key was either read
or written.

	c := cache.NewExpiresAfterAccessCache(time.Minute * 5)

Power Cache
---

Power cache is the underlying implementation. You can get ahold of the power
cache and then combine size and time based evictions to create things like a
least recently used cache.

	c := cache.NewPowerCache()
	c.MaxKeys = 100
	c.ExpiresAfterAccessDuration = time.Minute * 5
	c.PeriodicMaintenance = time.Hour

Value Loader
---

In order to get keys into the cache you can use a value loader. This can be
specified at the cache level, or at the function level. Assuming c is your
cache:

	c.ValueLoader = vl
	c.GetWithValueLoader("key", vl)
	
	func vl(key string) (interface{}, error){
		//TODO Get a key from slow source
		//TODO return value with nil error if ok
		return "Hello World", nil
		//TODO return nil value with error if bad
		return nil, cache.ErrNotPresent
	}

[google-guava]: https://code.google.com/p/guava-libraries/wiki/CachesExplained
