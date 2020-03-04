package main

import (
	"encoding/base64"
	"fmt"
	"log"
	"strconv"
)

type configHolder map[string]string

func (ch *configHolder) add(key string, value string) {
	(*ch)[key] = value
}

type configs struct {
	smtpServer     string
	smtpPort       int
	imapServer     string
	imapPort       int
	rcptQuantity   int
	metricDelay    int
	deleyAfterStop int
	mesPerSecond   int
	file           string
	logFile        string
	usersFile      string
}

func (ch configHolder) parseIntConf(key string, confField *int, errChan chan<- error) {
	fieldStr, ok := ch[key]
	if !ok {
		errChan <- fmt.Errorf("Configuration %s not found", key)
		return
	}
	field, err := strconv.Atoi(fieldStr)
	if err != nil {
		errChan <- fmt.Errorf("Invalid xvalue %s configuration: %v", key, err)
		return
	}
	*confField = field
}

func (ch configHolder) parseStringConf(key string, confField *string, errChan chan<- error) {
	field, ok := ch[key]
	if !ok {
		errChan <- fmt.Errorf("Configuration %s not found", key)
		return
	}
	*confField = field
}

func createConfigs() *configs {
	ch := make(configHolder)
	readConfFile("config", &ch)
	c := configs{}

	errChan := make(chan error)
	haveErrors := false
	go func() {
		for {
			if err := <-errChan; err != nil {
				log.Println(err)
				haveErrors = true
			}
		}
	}()
	var fileSize int
	ch.parseStringConf("smtp_server", &c.smtpServer, errChan)
	ch.parseIntConf("smtp_port", &c.smtpPort, errChan)
	ch.parseStringConf("imap_server", &c.imapServer, errChan)
	ch.parseIntConf("smtp_port", &c.imapPort, errChan)
	ch.parseIntConf("rcpt_quantity", &c.rcptQuantity, errChan)
	ch.parseIntConf("metric_delay", &c.metricDelay, errChan)
	ch.parseIntConf("deley_after_stop", &c.deleyAfterStop, errChan)
	ch.parseIntConf("mes_per_second", &c.mesPerSecond, errChan)
	ch.parseIntConf("file_size", &fileSize, errChan)
	ch.parseStringConf("log_file", &c.logFile, errChan)
	ch.parseStringConf("users_file", &c.usersFile, errChan)

	if fileSize > 0 {
		c.file = base64.StdEncoding.EncodeToString(make([]byte, fileSize*1000))
	}
	if haveErrors {
		return nil
	}

	return &c
}

func createEmptyFile(size int) {

}
