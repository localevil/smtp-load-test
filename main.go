package main

import (
	"time"
)

var mtl *metrics
var conf configs

func main() {
	conf = *createConfigs()

	newLogger(conf.logFile)

	mtl = createMetrics(conf.metricDelay)

	s := createSender(createUsersHolder(conf.usersFile))
	s.startThreads(conf.mesPerSecond)
	mtl.startMetricsTreads(conf.mesPerSecond)
	s.Start()
	<-time.After(time.Duration(conf.deleyAfterStop) * time.Second)
	s.Close()

	mtl.Close()
}
