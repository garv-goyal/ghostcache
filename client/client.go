package main

import (
    "fmt"
    "log"
    "time"

    "github.com/garv-goyal/ghostcache/clientlib"
)

func main() {
	gc := clientlib.NewGhostCacheClient("http://localhost:9000")

	if err := gc.Set("testkey", "testvalue"); err != nil {
		log.Fatalf("Set error: %v", err)
	}
	fmt.Println("Set successful")

	val, err := gc.Get("testkey")
	if err != nil {
		log.Fatalf("Get error: %v", err)
	}
	fmt.Println("Got value:", val)

	if err := gc.UpdateTTL("testkey", 30); err != nil {
		log.Fatalf("UpdateTTL error: %v", err)
	}
	fmt.Println("TTL updated")

	stats, err := gc.Stats("testkey")
	if err != nil {
		log.Fatalf("Stats error: %v", err)
	}
	fmt.Println("Stats:", stats)

	if err := gc.Delete("testkey"); err != nil {
		log.Fatalf("Delete error: %v", err)
	}
	fmt.Println("Key deleted")

	time.Sleep(2 * time.Second)
}
