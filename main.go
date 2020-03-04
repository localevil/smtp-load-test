package main

import (
	"fmt"
	"runtime"
	"time"
)

var mtl *metrics
var conf configs

func main() {
	fmt.Println("Version 1.3")
	runtime.GOMAXPROCS(runtime.NumCPU())
	conf = *createConfigs()
	newLogger(conf.logFile)
	mtl = createMetrics(conf.metricDelay)
	s := createSender(createUsersHolder(conf.usersFile))
	s.startThreads(conf.mesPerSecond)
	mtl.startMetricsTreads(conf.mesPerSecond)
	s.Start()
	globalStart := time.Now()
	fmt.Printf("---- Start testing: %s ----\n\n", globalStart.Format("Mon Jan _2 15:04:05 2006"))
	<-time.After(time.Duration(conf.deleyAfterStop) * time.Second)
	s.Close()

	mtl.Close()
	fmt.Printf("\n---- Stop testing: %s ----\n---- Test total time: %s ----\n", time.Now().Format("Mon Jan _2 15:04:05 2006"), time.Since(globalStart))
}

// Y2FiLXNhLXNibWwwMDAx
// cFUzRmc4N0YhMQ==
