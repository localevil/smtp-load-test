package main

import (
	"fmt"
	"sync"
	"time"
)

type metric struct {
	metricStatus      bool
	smtpSession       time.Duration
	excangeProcessing time.Duration
	auth              time.Duration
	total             time.Duration
	sent              bool
	received          bool
	size              int64
	rcptQuant         int
}

type metrics struct {
	sumSession, sumExc, sumAuth, sumTotal time.Duration
	sent, received                        int
	delay                                 int
	count                                 int64
	addChan                               chan *metric
	sumMutex                              sync.Mutex
	wg                                    sync.WaitGroup
	mpsMutex                              sync.Mutex
	mps                                   int
	printMetric                           chan int
	stopMetric                            chan struct{}
}

func createMetrics(delay int) *metrics {
	return &metrics{delay: delay,
		addChan:     make(chan *metric),
		printMetric: make(chan int),
		stopMetric:  make(chan struct{}),
	}
}

func (ml *metrics) startMetricsTreads(n int) {
	for i := 0; i < n; i++ {
		go ml.showMetrics()
	}
	go ml.tick()
}

func (ml *metrics) showMetrics() {
	ml.wg.Add(1)
	defer ml.wg.Done()
	for {
		m, ok := <-ml.addChan
		if !ok {
			return
		}

		ml.sumMutex.Lock()
		ml.sumSession += m.smtpSession
		// ml.sumExc += m.excangeProcessing
		ml.sumAuth += m.auth
		ml.sumTotal += m.total
		if m.sent == true {
			ml.sent++
		}
		if m.received == true {
			ml.received++
		}
		sumSession := ml.sumSession
		// sumExc := ml.sumExc
		sumAuth := ml.sumAuth
		sumTotal := ml.sumTotal
		sent := ml.sent
		// received := ml.received
		ml.sumMutex.Unlock()

		ml.mpsMutex.Lock()
		ml.mps++
		mps := ml.mps
		ml.mpsMutex.Unlock()

		if ml.count > 0 {
			avSession := time.Duration(sumSession.Nanoseconds() / ml.count)
			// avExc := time.Duration(sumExc.Nanoseconds() / ml.count)
			avAuth := time.Duration(sumAuth.Nanoseconds() / ml.count)
			avTotal := time.Duration(sumTotal.Nanoseconds() / ml.count)

			select {
			case <-ml.printMetric:
				fmt.Printf("SMTP Session %s | SMTP AUTH: %s | Total: %s | MPS: %d | Sent: %d \n", avSession, avAuth, avTotal, mps, sent)
				ml.mpsMutex.Lock()
				ml.mps = 0
				ml.mpsMutex.Unlock()
			default:
			}
		}
	}
}

func (ml *metrics) tick() {
	for {
		select {
		case <-time.After(time.Second * time.Duration(ml.delay)):
			ml.printMetric <- 1
		case <-ml.stopMetric:
			return
		}
	}
}

func (ml *metrics) add(m *metric) {
	if ml.addChan != nil {
		ml.addChan <- m
	}
	ml.count++
}

func (ml *metrics) Close() {
	close(ml.stopMetric)
	if ml.addChan != nil {
		close(ml.addChan)
	}
	ml.wg.Wait()
}
