package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	pinger "github.com/gvialetto/go-pinger"
)

func main() {
	flag.Parse()
	args := flag.Args()
	pingServ, err := pinger.New()
	if err != nil {
		log.Fatal("Cannot initialize pinger: ", err)
	}
	pingServ.AddHandler(func(host string, latency time.Duration, err error) {
		fmt.Printf("got response from %s: time=%s\n", host, latency)
	})
	for _, host := range args {
		pingServ.AddHost(host, 1*time.Second, 1*time.Minute)
	}
	pingServ.Run(context.Background())
	pingServ.Wait()
}
