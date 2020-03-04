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
	mps                                   int
	printMetric                           chan int
	stopMetric                            chan struct{}
	sumMutex                              sync.Mutex
	mpsMutex                              sync.Mutex
	wg                                    sync.WaitGroup
}

func createMetrics(delay int) *metrics {
	return &metrics{delay: delay,
		addChan:     make(chan *metric),
		printMetric: make(chan int),
		stopMetric:  make(chan struct{}),
	}
}

func (ml *metrics) startMetricsTreads(n int) {
	for i := 0; i < int(float32(n)*0.10); i++ {
		go ml.showMetrics()
	}
	go ml.tick()
}

func (ml *metrics) showMetrics() {
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
		ml.sumMutex.Unlock()

		ml.mpsMutex.Lock()
		ml.mps++
		ml.mpsMutex.Unlock()
		ml.wg.Done()
	}
}

func (ml *metrics) print() {
	ml.sumMutex.Lock()
	sumSession := ml.sumSession
	// sumExc := ml.sumExc
	sumAuth := ml.sumAuth
	sumTotal := ml.sumTotal
	sent := ml.sent
	// received := ml.received
	ml.sumMutex.Unlock()

	ml.mpsMutex.Lock()
	mps := ml.mps
	ml.mpsMutex.Unlock()
	if ml.count <= 0 {
		return
	}
	avSession := time.Duration(sumSession.Nanoseconds() / ml.count)
	// avExc := time.Duration(sumExc.Nanoseconds() / ml.count)
	avAuth := time.Duration(sumAuth.Nanoseconds() / ml.count)
	avTotal := time.Duration(sumTotal.Nanoseconds() / ml.count)

	fmt.Printf("SMTP Session %s | SMTP AUTH: %s | Message Sending Time: %s | MPS: %d | Sent: %d \n", avSession, avAuth, avTotal, mps, sent)
	ml.mpsMutex.Lock()
	ml.mps = 0
	ml.mpsMutex.Unlock()
}

func (ml *metrics) tick() {
	for {
		select {
		case <-ml.stopMetric:
			return
		case <-time.After(time.Second * time.Duration(ml.delay)):
			go ml.print()
		}
	}
}

func (ml *metrics) add(m *metric) {
	if ml.addChan != nil {
		ml.wg.Add(1)
		ml.addChan <- m
	}
	ml.count++
}

func (ml *metrics) Close() {
	ml.wg.Wait()
	ml.print()
	close(ml.stopMetric)

	close(ml.addChan)
}
