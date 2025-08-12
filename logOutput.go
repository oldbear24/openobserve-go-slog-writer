// Package logoutput provides an io.Writer implementation that can forward
// logs to an OpenObserve instance while still writing them to stdout.
package logoutput

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

// LogOutput implements io.Writer and buffers log entries for optional
// delivery to an OpenObserve instance.
type LogOutput struct {
	enableExtenal        bool
	externalUrl          string
	authToken            string
	externalOrganization string
	externalStream       string
	logChannel           chan []byte
	logs                 [][]byte
	lastLogUpload        time.Time
}

// Write implements io.Writer. It always writes to stdout and, when external
// logging is enabled, queues the log entry for uploading to OpenObserve.
func (l *LogOutput) Write(p []byte) (n int, err error) {
	if l.enableExtenal {
		var data []byte = make([]byte, len(p))
		copy(data, p)
		l.logChannel <- data
	}
	return os.Stdout.Write(p)
}

func (l *LogOutput) sendLogToExternalService() error {
	if len(l.logs) == 0 {
		return nil
	}
	defer func() {
		l.logs = make([][]byte, 0)
	}()
	payload := []json.RawMessage{}
	for _, log := range l.logs {
		payload = append(payload, json.RawMessage(log))
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	url, err := url.JoinPath(l.externalUrl, "api", l.externalOrganization, l.externalStream, "_json")
	if err != nil {
		return fmt.Errorf("failed to parse URL: %w", err)
	}
	req, err := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Basic "+l.authToken)

	// Tell the server we're sending JSON
	req.Header.Set("Content-Type", "application/json")

	// (Optional) add any other headers you need
	req.Header.Set("Accept", "application/json")

	// Use a client with a timeout
	client := &http.Client{Timeout: 10 * time.Second}

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status: %s body: %s", resp.Status, string(body))
	}
	return nil
}

// ForceLogToExternalService forces any pending logs to be uploaded.
// The function is currently a placeholder for future enhancements.
func ForceLogToExternalService() {

}

// Close flushes any buffered logs and releases resources associated with the
// writer.
func (l *LogOutput) Close() {
	if l.enableExtenal {
		if err := l.sendLogToExternalService(); err != nil {
			fmt.Fprintf(os.Stderr, "failed to send logs: %v\n", err)
		}
		close(l.logChannel)
	}
}

// New creates a new LogOutput. When enableExtenal is true, logs are queued and
// periodically uploaded to the provided OpenObserve endpoint.
func New(enableExtenal bool, externalUrl, authToken, externalOrganization, externalStream string) *LogOutput {

	l := &LogOutput{}
	l.enableExtenal = enableExtenal
	if l.enableExtenal {
		l.externalUrl = externalUrl
		l.authToken = authToken
		l.externalOrganization = externalOrganization
		l.externalStream = externalStream
		l.logChannel = make(chan []byte, 1024)
		l.logs = make([][]byte, 0)
		l.lastLogUpload = time.Now()
		go l.logWorker()
	}
	return l
}

func (l *LogOutput) logWorker() {
	for {
		select {
		case <-time.After(time.Second * 5):
			if l.shouldSendLog() {
				if err := l.sendLogToExternalService(); err != nil {
					fmt.Fprintf(os.Stderr, "failed to send logs: %v\n", err)
				}
			}
		case log := <-l.logChannel:
			l.logs = append(l.logs, log)
			if l.shouldSendLog() {
				if err := l.sendLogToExternalService(); err != nil {
					fmt.Fprintf(os.Stderr, "failed to send logs: %v\n", err)
				}
			}

		}
	}

}
func (l *LogOutput) shouldSendLog() bool {
	if l.lastLogUpload.Before(time.Now().Add(-time.Minute)) || len(l.logs) > 100 {
		l.lastLogUpload = time.Now()
		return true
	}
	return false
}
