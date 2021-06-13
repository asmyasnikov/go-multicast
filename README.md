# go-multicast
[<img src="https://api.travis-ci.org/asmyasnikov/go-multicast.svg?branch=master">](https://travis-ci.org/github/asmyasnikov/go-multicast)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/asmyasnikov/go-multicast)](https://pkg.go.dev/github.com/asmyasnikov/go-multicast)
[![GoDoc](https://godoc.org/github.com/asmyasnikov/go-multicast?status.svg)](https://godoc.org/github.com/asmyasnikov/go-multicast)
[![Go Report Card](https://goreportcard.com/badge/github.com/asmyasnikov/go-multicast)](https://goreportcard.com/badge/github.com/asmyasnikov/go-multicast)
[![codecov](https://codecov.io/gh/asmyasnikov/go-multicast/branch/master/graph/badge.svg)](https://codecov.io/gh/asmyasnikov/go-multicast)
![tests](https://github.com/asmyasnikov/go-multicast/workflows/tests/badge.svg?branch=master)
![lint](https://github.com/asmyasnikov/go-multicast/workflows/lint/badge.svg?branch=master)

package provide multicast sending messages to all connected clients

Usage:

```go
package main

import (
	"context"
	"github.com/asmyasnikov/go-multicast"
	"github.com/asmyasnikov/go-multicast/helpers/websocket"
	"net/http"
	"fmt"
	"time"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m := multicast.New(
		ctx,
		func(err error) {
			fmt.Println(err)
		},
		nil, // latest message replace snapshot
		time.Second,
	)
	go func() {
		for {
			m.SendAll(time.Now())
			time.Sleep(time.Millisecond * 100)
		}
	}()
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		conn, err := websocket.Upgrader.Upgrade(w, r, nil)
		if err != nil {
			fmt.Println(err)
			return
		}
		m.Add(conn)
	})
	if err := http.ListenAndServe(":80", nil); err != nil {
		fmt.Println(err)
	}
}
```
