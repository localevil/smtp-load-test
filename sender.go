package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/smtp"
	"sync"
	"time"
)

const (
	mailboxSufix = "@sigma-exchtest.sbrf.ru"
)

//NextGetter returning point to next item interface
type NextGetter interface {
	GetNext() *account
	GetRandom() *account
}

type rcptList []string

func (s rcptList) ToString() string {
	var result string
	for _, rcpt := range s {
		result += rcpt + mailboxSufix + ", "
	}
	return result[:len(result)-2]
}

type sendInfo struct {
	a   *account
	to  rcptList
	mID int
}

type sender struct {
	mr             messageReader
	users          NextGetter
	sendWg         sync.WaitGroup
	preprocWg      sync.WaitGroup
	stopThreadChan chan struct{}
	startChan      chan struct{}
	sendInfoChan   chan sendInfo
	currentMsgID   int
	stopProcMut    sync.Mutex
	sendMessage    chan int
}

func createSender(u *usersHolder) sender {
	return sender{users: u, mr: createReader(u), startChan: make(chan struct{}), sendMessage: make(chan int)}
}

func (s *sender) createMsg(from string, msgID int, to rcptList) string {
	return fmt.Sprintf("From: %s\nTo: %s\nSubject:%d\n\n TestMesage", from, to.ToString(), msgID)
}

func (s *sender) sendMail() {
	si, ok := <-s.sendInfoChan
	if !ok {
		return
	}

	auth := smtp.PlainAuth("", si.a.username, si.a.password, conf.smtpServer)

	m := new(metric)

	start := time.Now()
	c, err := smtp.Dial(fmt.Sprintf("%s:%d", conf.smtpServer, conf.smtpPort))
	if err != nil {
		log.Printf("SMTP session not started: %v", err)
		return
	}

	c.StartTLS(&tls.Config{
		InsecureSkipVerify: true,
		ServerName:         conf.smtpServer,
	})

	startAuth := time.Now()
	if err = c.Auth(auth); err != nil {
		log.Printf("Smtp Authentication fail: %v", err)
		return
	}
	m.auth = time.Since(startAuth)

	smtpStart := time.Now()

	if err = c.Mail(si.a.username + mailboxSufix); err != nil {
		log.Printf("Sender fail: %v", err)
		return
	}

	for _, rcpt := range si.to {
		if err = c.Rcpt(rcpt + mailboxSufix); err != nil {
			log.Printf("Recipient %s fail: %v", rcpt, err)
			return
		}
	}

	w, err := c.Data()
	if err != nil {
		log.Printf("Data fail: %v", err)
	}
	message := s.createMsg(si.a.username, si.mID, si.to)
	n, err := w.Write([]byte(message))
	if err != nil || n != len(message) {
		log.Printf("Write data fail: %v", err)
		return
	}

	err = w.Close()
	if err != nil {
		log.Printf("Close session fail: %v", err)
		return
	}

	c.Quit()
	m.sent = true
	m.smtpSession = time.Since(smtpStart)
	m.total = time.Since(start)
	// for _, rcpt := range si.to {
	// 	go s.mr.messageHandler(rcpt, si.mID, start, m)
	// }

	mtl.add(m)
}

func (s *sender) startThread() {
	<-s.startChan
	s.sendWg.Add(1)
	defer s.sendWg.Done()
	for {
		select {
		case <-s.stopThreadChan:
			return
		default:
			s.sendMail()
		}
	}
}

func (s *sender) startThreads(c int) {
	t := time.Duration(time.Second.Nanoseconds() / int64(c))
	s.stopThreadChan = make(chan struct{})
	s.sendInfoChan = make(chan sendInfo, c)
	for i := 0; i < c; i++ {
		go s.startThread()
		go s.sendPerSecond()
	}
	go s.tick(t)
}

func (s *sender) sendPerSecond() {
	<-s.startChan
	for {
		select {
		case <-s.stopThreadChan:
			return
		case <-s.sendMessage:
			go s.sendPreprocessor()
		}
	}
}

func (s *sender) sendPreprocessor() {
	s.preprocWg.Add(1)
	defer s.preprocWg.Done()
	a := s.users.GetNext()
	var to rcptList
	var mID = s.currentMsgID
	s.currentMsgID++
	for i := 0; i < conf.rcptQuantity; i++ {
		to = append(to, s.users.GetRandom().username)
	}
	if s.sendInfoChan != nil {
		s.sendInfoChan <- sendInfo{a, to, mID}
	}
}

func (s *sender) tick(t time.Duration) {
	for {
		<-time.After(t)
		s.sendMessage <- 1
	}
}

func (s *sender) Start() {
	if s.startChan != nil {
		close(s.startChan)
	}
	fmt.Printf("Start testing\n")
}

func (s *sender) Close() {
	close(s.stopThreadChan)
	s.preprocWg.Wait()
	close(s.sendInfoChan)
	s.sendInfoChan = nil
	s.sendWg.Wait()
	s.mr.Close()
	fmt.Printf("Stop testing\n")
}
