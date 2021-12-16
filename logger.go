package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/jptosso/coraza-waf/v2/loggers"
)

const queueSize = 1000
const workers = 10
const writeTimeout = time.Second * 2

type httpLogger struct {
	queue      chan loggers.AuditLog
	stop       chan bool
	systemStop chan os.Signal
	options    loggers.LoggerOptions
}

func (l *httpLogger) Init(logger loggers.LoggerOptions) error {
	l.queue = make(chan loggers.AuditLog, queueSize)
	l.stop = make(chan bool)
	l.systemStop = make(chan os.Signal, 1)
	l.options = logger
	l.start()
	signal.Notify(l.systemStop, os.Interrupt)

	return nil
}

func (l *httpLogger) Write(al loggers.AuditLog) error {
	select {
	case l.queue <- al:
		// it works
	default:
		// channel is full, we will discard it but
		// future versions will cache it to a file
	}

	return nil
}

func (l *httpLogger) Close() error {
	//close(l.queue)
	l.stop <- true
	return nil
}

func (l *httpLogger) start() {
	// TODO we may add another routine to handle overload
	// and create cache files
	// TODO if you stop the WAF this will just die
	for i := 0; i < workers; i++ {
		go func() {
			for {
				select {
				case al := <-l.queue:
					if err := l.writeHttp(al); err != nil {
						// back to the queue by now
						l.queue <- al
					}
				case <-l.stop:
					return
				case <-l.systemStop:
					// this works if the server was stopped
					return
				}
			}
		}()
	}
}

func (l *httpLogger) writeHttp(al loggers.AuditLog) error {
	// we send the auditlog as a json payload
	client := http.Client{
		Timeout: writeTimeout,
	}
	bts, err := json.Marshal(al)
	if err != nil {
		return err
	}
	_, err = client.Post(l.options.File, "application/json", bytes.NewReader(bts))
	return err
}

var _ loggers.LogWriter = &httpLogger{}
