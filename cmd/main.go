package main

import (
	"context"
	"fmt"
	"ip-counter/pkg/IPCounter"
	"log"
	"time"
)

func main() {
	filePath := "U:/ip_addresses"
	start := time.Now()
	ip := IPCounter.NewIPCounter(1, '\n')
	v, err := ip.UniqueIP4(context.Background(), filePath) //"U:/ip_addresses"
	fmt.Printf("Time taken: %s\n", time.Since(start))
	if err != nil {
		log.Fatalf("Failed to count unique IPs in file %s: %v", filePath, err)
	}
	fmt.Printf("The number of unique IPv4 addresses found is: %d", v)
}
