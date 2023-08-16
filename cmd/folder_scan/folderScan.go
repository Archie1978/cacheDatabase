package main

import (
	"cacheDatabase"
	"flag"
	"runtime"
	"time"
)

func main() {

	MaxParallelism()

	var scanPath = flag.String("scanPath", "", "Scan the directory path")
	var databasePath = flag.String("database", "database.sqlite", "Use database")

	flag.Parse()

	// Open database
	cacheDatabase.DEBUG = true
	openDatabase(*databasePath)

	FileScan(*scanPath)
	<-time.After(10 * time.Second)
}

func MaxParallelism() int {
	maxProcs := runtime.GOMAXPROCS(0)
	numCPU := runtime.NumCPU()
	if maxProcs < numCPU {
		return maxProcs
	}
	return numCPU
}
