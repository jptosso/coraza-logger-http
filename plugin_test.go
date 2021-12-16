package main

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/jptosso/coraza-waf/v2"
	"github.com/jptosso/coraza-waf/v2/loggers"
	"github.com/jptosso/coraza-waf/v2/seclang"
	"github.com/jptosso/coraza-waf/v2/types"
)

func TestPlugin(t *testing.T) {
	al := make(chan *loggers.AuditLog)
	listener(al, t)
	waf := coraza.NewWaf()
	parser, _ := seclang.NewParser(waf)
	if err := parser.FromString(`
		SecAction "id:1,phase:1,auditlog,log"
	`); err != nil {
		t.Fatal(err)
	}
	waf.AuditEngine = types.AuditEngineOn
	waf.AuditLog = "http://127.0.0.1:9200/coraza/audit/_create"
	waf.AuditLogType = "http"
	waf.AuditLogFormat = "json"
	if err := waf.UpdateAuditLogger(); err != nil {
		t.Error(err)
	}
	tx := waf.NewTransaction()
	req, _ := http.NewRequest("GET", "http://example.com/?id=123", nil)
	if _, err := tx.ProcessRequest(req); err != nil {
		t.Error(err)
	}
	tx.ProcessLogging()
	audit := <-al
	if audit.Transaction.ID != tx.ID {
		t.Error("transaction id mismatch")
	}
}

// listen on port 9200 and check that the auditlog is sent
func listener(al chan *loggers.AuditLog, t *testing.T) {
	go func() {
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// read json payload and create auditlog
			a := &loggers.AuditLog{}
			if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
				t.Error(err)
			}
			al <- a
		})
		if err := http.ListenAndServe(":9200", nil); err != nil {
			t.Error(err)
		}
	}()
}
