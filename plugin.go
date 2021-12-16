package httplogger

import "github.com/jptosso/coraza-waf/v2/loggers"

func init() {
	loggers.RegisterLogWriter("http", func() loggers.LogWriter {
		return &httpLogger{}
	})
}
