package main

import (
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

type searcher interface {
	findPassword(username string) (string, error)
}

type messageReader interface {
	messageHandler(username string, msgID int, startTotal time.Time, m *metric)
	Close()
}

type reader struct {
	wg       sync.WaitGroup
	s        searcher
	stopChan chan struct{}
}

func createReader(s searcher) *reader {

	return &reader{s: s, stopChan: make(chan struct{})}
}

func (r *reader) messageHandler(username string, msgID int, startTotal time.Time, m *metric) {
	r.wg.Add(1)
	defer r.wg.Done()

	c, err := client.Dial(fmt.Sprintf("%s:%d", conf.imapServer, conf.imapPort))
	if err != nil {
		log.Printf("Imap connection fail: %v", err)
		return
	}

	password, err := r.s.findPassword(username)
	if err != nil {
		log.Printf("%v", err)
		return
	}

	if err := c.Login(username, password); err != nil {
		log.Printf("Imap Login %s fail: %v", username, err)
		return
	}

	defer c.Logout()

	mbox, err := c.Select("INBOX", false)
	if err != nil {
		log.Printf("INBOX mailbox select fail: %v", err)
	}

	for {
		from := uint32(1)
		to := mbox.Messages
		if mbox.Messages > 5 {
			from = mbox.Messages - 5
		}

		seqset := new(imap.SeqSet)
		seqset.AddRange(from, to)

		messagesChan := make(chan *imap.Message, 5)
		errChan := make(chan error, 1)

		go func() {
			errChan <- c.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope}, messagesChan)
		}()

		for msg := range messagesChan {
			id, err := strconv.Atoi(msg.Envelope.Subject)
			if err != nil {
				log.Printf("Invalid subject %s: %v", msg.Envelope.Subject, err)
				continue
			}
			if id == msgID {
				m.total = time.Since(startTotal)
				item := imap.FormatFlagsOp(imap.AddFlags, true)
				flags := []interface{}{imap.DeletedFlag}
				delseq := new(imap.SeqSet)
				delseq.AddNum(msg.SeqNum)
				if err := c.Store(delseq, item, flags, nil); err != nil {
					log.Printf("Can`t mark message %s as deleted: %v", msg.Envelope.Subject, err)
				}
				if err := c.Expunge(nil); err != nil {
					log.Printf("Can`t delete %s: %v", msg.Envelope.Subject, err)
				}
				return
			}
			log.Printf("Message %s not found", msg.Envelope.Subject)
			m.received = false

		}

		if err := <-errChan; err != nil {
			log.Printf("Fetch fail: %v", err)
		}

		select {
		case <-r.stopChan:
			return
		default:
		}
	}
}

func (r *reader) Close() {
	close(r.stopChan)
	r.wg.Wait()
}
