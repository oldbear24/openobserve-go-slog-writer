# OpenObserve Go slog writer

This module provides an `io.Writer` for Go's `slog` logging framework that forwards records to an [OpenObserve](https://openobserve.ai) instance while still writing them to standard output.

## Installation

```bash
go get github.com/oldbear24/openobserve-go-slog-writer
```

## Usage

```go
package main

import (
        "log/slog"

        logoutput "github.com/oldbear24/openobserve-go-slog-writer"
)

func main() {
        // Configure the writer. When enableExtenal is false, logs are only
        // written to stdout.
        writer := logoutput.New(true, "http://localhost:5080", "authToken", "org", "stream")
        logger := slog.New(slog.NewJSONHandler(writer, nil))

        logger.Info("hello from slog")

        // Close flushes any buffered logs.
        writer.Close()
}
```

## Configuration

- `externalUrl`: Base URL of the OpenObserve instance.
- `authToken`: Base64 encoded credentials for Basic Authentication.
- `externalOrganization`: Organization to send logs to.
- `externalStream`: Stream name for logs.

Logs are batched and automatically uploaded periodically or when the buffer exceeds a threshold.

