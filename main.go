package main

import (
	"flag"
	"log"
	"github.com/garv-goyal/ghostcache/coordinator"
	"github.com/garv-goyal/ghostcache/node"
)

func main() {
	mode := flag.String("mode", "node", "run mode: node or coordinator")
	port := flag.String("port", "8080", "port to run on")
	flag.Parse()

	if *mode == "coordinator" {
		coord := coordinator.NewCoordinator()
		coord.Start(*port)
	} else {
		srv := node.NewServer()
		srv.Start(*port)
	}
	log.Println("Service started.")
}
