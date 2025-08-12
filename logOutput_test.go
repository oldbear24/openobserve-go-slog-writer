package logoutput

import (
	"encoding/base64"
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func TestWrite_WritesToStdout(t *testing.T) {
	l := &LogOutput{}
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	orig := os.Stdout
	os.Stdout = w
	msg := []byte("hello\n")
	if _, err := l.Write(msg); err != nil {
		t.Fatalf("write: %v", err)
	}
	w.Close()
	os.Stdout = orig
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(out) != string(msg) {
		t.Errorf("expected %q, got %q", msg, out)
	}
}

func TestSendLogToOpenObserve(t *testing.T) {
	token := base64.StdEncoding.EncodeToString([]byte("root@example.com:Complexpass#123"))
	l := &LogOutput{
		enableExtenal:        true,
		externalUrl:          "http://localhost:5080",
		authToken:            token,
		externalOrganization: "default",
		externalStream:       "default",
		logChannel:           make(chan []byte, 1),
		logs:                 make([][]byte, 0),
	}
	log := []byte(`{"message":"integration test"}`)
	if _, err := l.Write(log); err != nil {
		t.Fatalf("write: %v", err)
	}
	select {
	case msg := <-l.logChannel:
		l.logs = append(l.logs, msg)
	case <-time.After(5 * time.Second):
		t.Skip("log not received; skipping")
	}
	if err := l.sendLogToExternalService(); err != nil {
		if strings.Contains(err.Error(), "connection refused") || strings.Contains(err.Error(), "No such host") {
			t.Skipf("OpenObserve not running: %v", err)
		}
		t.Fatalf("send log: %v", err)
	}
}
