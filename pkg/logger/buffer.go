package logger

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
)

const (
	MaxLogLines = 1000
	MaxLogSize  = 100 * 1024
)

var (
	logBuffer      *bytes.Buffer
	logBufferMu    sync.RWMutex
	lineCount      int
	originalStdout *os.File
	originalStderr *os.File
	pipeReader     *os.File
	pipeWriter     *os.File
	stopCh         chan struct{}
)

type BufferLogger struct{}

func (l BufferLogger) Debug(ctx context.Context, args ...interface{}) {
	AppendLog("[Debug] ", args)
}

func (l BufferLogger) Info(ctx context.Context, args ...interface{}) {
	AppendLog("[Info] ", args)
}

func (l BufferLogger) Warn(ctx context.Context, args ...interface{}) {
	AppendLog("[Warn] ", args)
}

func (l BufferLogger) Error(ctx context.Context, args ...interface{}) {
	AppendLog("[Error] ", args)
}

func Setup() (cleanup func(), stdoutWriter io.Writer) {
	logBufferMu.Lock()
	defer logBufferMu.Unlock()

	logBuffer = &bytes.Buffer{}
	lineCount = 0

	defaultHandler := slog.NewTextHandler(logBuffer, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})
	slog.SetDefault(slog.New(defaultHandler))

	originalStdout = os.Stdout
	originalStderr = os.Stderr

	var err error
	pipeReader, pipeWriter, err = os.Pipe()
	if err != nil {
		return func() {}, originalStdout
	}

	os.Stdout = pipeWriter
	os.Stderr = pipeWriter

	stopCh = make(chan struct{})

	go func() {
		buf := make([]byte, 4096)
		for {
			select {
			case <-stopCh:
				return
			default:
				n, err := pipeReader.Read(buf)
				if err != nil {
					return
				}
				appendLogBytes(buf[:n])
			}
		}
	}()

	cleanup = func() {
		logBufferMu.Lock()
		defer logBufferMu.Unlock()

		close(stopCh)
		pipeWriter.Close()
		pipeReader.Close()

		os.Stdout = originalStdout
		os.Stderr = originalStderr
		logBuffer = nil
	}

	return cleanup, originalStdout
}

func appendLogBytes(data []byte) {
	logBufferMu.Lock()
	defer logBufferMu.Unlock()

	if logBuffer == nil {
		return
	}

	logBuffer.Write(data)

	for _, c := range data {
		if c == '\n' {
			lineCount++
		}
	}

	if lineCount > MaxLogLines || logBuffer.Len() > MaxLogSize {
		trimLogBufferLocked()
	}
}

func trimLogBufferLocked() {
	data := logBuffer.Bytes()
	lines := bytes.Split(data, []byte("\n"))

	if len(lines) > MaxLogLines {
		keepLines := lines[len(lines)-MaxLogLines:]
		newBuffer := bytes.NewBuffer(nil)
		for i, line := range keepLines {
			if i > 0 {
				newBuffer.WriteByte('\n')
			}
			newBuffer.Write(line)
		}
		logBuffer = newBuffer
		lineCount = len(keepLines)
	}
}

func AppendLog(prefix string, args []interface{}) {
	logBufferMu.Lock()
	defer logBufferMu.Unlock()

	if logBuffer == nil {
		return
	}

	logBuffer.WriteString(prefix)
	for i, arg := range args {
		if i > 0 {
			logBuffer.WriteString(" ")
		}
		logBuffer.WriteString(fmt.Sprintf("%v", arg))
	}
	logBuffer.WriteString("\n")
	lineCount++

	if lineCount > MaxLogLines || logBuffer.Len() > MaxLogSize {
		trimLogBufferLocked()
	}
}

func GetRecentLogs(maxBytes int) string {
	logBufferMu.RLock()
	defer logBufferMu.RUnlock()

	if logBuffer == nil {
		return ""
	}

	data := logBuffer.Bytes()
	if len(data) <= maxBytes {
		return string(data)
	}
	return string(data[len(data)-maxBytes:])
}

func NewBufferLogger() BufferLogger {
	return BufferLogger{}
}
