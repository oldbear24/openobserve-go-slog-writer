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

func (l *LogOutput) Write(p []byte) (n int, err error) {
	if l.enableExtenal {
		var data []byte = make([]byte, len(p))
		copy(data, p)
		l.logChannel <- data
	}
	return os.Stdout.Write(p)
}

func (l *LogOutput) sendLogToExternalService() {
	if len(l.logs) == 0 {
		return
	}
	payload := []json.RawMessage{}
	for _, log := range l.logs {
		payload = append(payload, json.RawMessage(log))
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		panic(fmt.Errorf("failed to marshal payload: %w", err))
	}

	url, err := url.JoinPath(l.externalUrl, "api", l.externalOrganization, l.externalStream, "_json")
	if err != nil {
		panic(fmt.Errorf("failed to parse URL: %w", err))
	}
	req, err := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		panic(fmt.Errorf("failed to create request: %w", err))
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
		panic(fmt.Errorf("request failed: %w", err))
	}
	defer resp.Body.Close()
	println("Response Status:", resp.Status)
	if resp.StatusCode != http.StatusOK {
		_, err := io.ReadAll(resp.Body)
		if err != nil {
			panic(fmt.Errorf("failed to read response body: %w", err))
		}
	}
	l.logs = make([][]byte, 0)
}

func ForceLogToExternalService() {

}
func (l *LogOutput) Close() {
	if l.enableExtenal {
		l.sendLogToExternalService()
		close(l.logChannel)
	}
}
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
				l.sendLogToExternalService()
			}
		case log := <-l.logChannel:
			l.logs = append(l.logs, log)
			if l.shouldSendLog() {
				l.sendLogToExternalService()
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
