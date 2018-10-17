[![Build Status](https://travis-ci.org/dotcloud/go-redis-server.png)](https://travis-ci.org/dotcloud/go-redis-server)

Redis server protocol library
=============================

There are plenty of good client implementations of the redis protocol, but not many *server* implementations.

go-redis-server is a helper library for building server software capable of speaking the redis protocol. This could be
an alternate implementation of redis, a custom proxy to redis, or even a completely different backend capable of
"masquerading" its API as a redis database.


Sample code
------------

```go
package main

import (
	redis "github.com/prettyyjnic/go-redis-server"
)

type MyHandler struct {
	values map[string][]byte
}

func (h *MyHandler) GET(key string) ([]byte, error) {
	v := h.values[key]
	return v, nil
}

func (h *MyHandler) SET(key string, value []byte) error {
	h.values[key] = value
	return nil
}

func main() {
    srv, err := redis.NewServer(redis.DefaultConfig().Proto("unix").Host("/tmp/redis.sock").Handler(&MyHandler{}))
	if err != nil {
		panic(err)
	}
	if err := srv.ListenAndServe(); err != nil {
		panic(err)
	}
}
```

Copyright (c) dotCloud 2013
