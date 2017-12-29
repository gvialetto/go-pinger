package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	pinger "github.com/gvialetto/go-pinger"
)

func main() {
	flag.Parse()
	args := flag.Args()
	pingServ, _ := pinger.New()
	pingServ.AddHandler(func(host string, latency time.Duration, err error) {
		fmt.Printf("got response from %s: time=%s\n", host, latency)
	})
	for _, host := range args {
		pingServ.AddHost(host, 1*time.Second, 1*time.Minute)
	}
	pingServ.Run(context.Background())
	pingServ.Wait()
}
