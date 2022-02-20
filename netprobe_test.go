package netprobe

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"
)

// TODO

func TestNetprobe(t *testing.T) {
	addrs := []string{
		"127.0.0.1:1235",
		"127.0.0.1:1236",
		"127.0.0.1:1237",
		"127.0.0.1:1238",
	}
	c, err := Dial(context.Background(), "tcp", addrs, 5*time.Second)
	if err != nil {
		log.Printf("failed: %v", err)
		return
	}
	fmt.Printf("DEBUG conntected %v\n", c)
	c.Close()
}
