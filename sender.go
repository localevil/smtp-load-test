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
	delimeter    = "**=myohmy689407924327"
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
	preprocWg      sync.WaitGroup
	stopThreadChan chan struct{}
	startChan      chan struct{}
	sendMsgChan    chan int
	sendInfoChan   chan sendInfo
	currentMsgID   int
	stopProcMut    sync.Mutex
	attached       string
}

func createSender(u *usersHolder) sender {
	return sender{users: u, mr: createReader(u), startChan: make(chan struct{}), sendMsgChan: make(chan int)}
}

func attachFile(buf string, filename string) string {
	return fmt.Sprintf("\r\n--%s\r\nContent-Type: text/plain; charset=\"utf-8\"\r\n"+
		"Content-Transfer-Encoding: base64\r\nContent-Disposition: attachment;filename=\"%s\"\r\n\r\n%s\r\n", delimeter, filename, buf)
}

func (s *sender) createMsg(from string, msgID int, to rcptList) string {
	sampleMsg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\n"+
		"Content-Type: multipart/mixed; boundary=\"%s\"\r\n\r\n--%s\r\n"+
		"Content-Type: text/html; charset=\"utf-8\"\r\nContent-Transfer-Encoding: 7bit\r\n\r\n%s \r\n%s \r\n", from, to, "Test Subject", delimeter, delimeter, "Test body ", s.attached)
	return sampleMsg
}

func (s *sender) sendMail() {

	<-s.startChan

	for {
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

		message := []byte(s.createMsg(si.a.username, si.mID, si.to))
		n, err := w.Write(message)
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
		m.total = time.Since(smtpStart)
		m.smtpSession = time.Since(start)
		// for _, rcpt := range si.to {
		// 	go s.mr.messageHandler(rcpt, si.mID, start, m)
		// }

		mtl.add(m)
		s.preprocWg.Done()
	}
}

func (s *sender) startThreads(c int) {
	t := time.Duration(time.Second.Nanoseconds() / int64(c))
	s.stopThreadChan = make(chan struct{})
	s.sendInfoChan = make(chan sendInfo, c)
	fmt.Printf("Start thread creating\n")
	for i := 0; i < int(float32(c)); i++ {
		go s.sendMail()
	}
	fmt.Printf("Stop thread creating\n")
	go s.sendPreprocessor()
	go s.tick(t)
}

func (s *sender) sendPreprocessor() {
	<-s.startChan
	for {
		a := s.users.GetNext()
		var to rcptList
		var mID = s.currentMsgID
		s.currentMsgID++
		for i := 0; i < conf.rcptQuantity; i++ {
			to = append(to, s.users.GetRandom().username)
		}
		if s.sendInfoChan != nil {
			<-s.sendMsgChan
			s.preprocWg.Add(1)
			s.sendInfoChan <- sendInfo{a, to, mID}
		}
	}
}

func (s *sender) tick(t time.Duration) {
	for {
		select {
		case <-s.stopThreadChan:
			return
		case <-time.After(t):
			s.sendMsgChan <- 1
		}
	}
}

func (s *sender) Start() {
	s.attached = attachFile(conf.file, "FixedSizeFile")
	if s.startChan != nil {
		close(s.startChan)
	}
}

func (s *sender) Close() {
	s.sendMsgChan <- 1
	close(s.stopThreadChan)
	s.preprocWg.Wait()
	close(s.sendInfoChan)
	s.sendInfoChan = nil
	s.mr.Close()
}
