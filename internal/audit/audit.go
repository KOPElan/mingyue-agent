package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Logger struct {
	mu       sync.Mutex
	file     *os.File
	enabled  bool
	pushURL  string
	pushChan chan *Entry
}

type Entry struct {
	Timestamp time.Time              `json:"timestamp"`
	User      string                 `json:"user"`
	Action    string                 `json:"action"`
	Resource  string                 `json:"resource"`
	Result    string                 `json:"result"`
	SourceIP  string                 `json:"source_ip"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

func New(logPath string, remotePush bool, remoteURL string, enabled bool) (*Logger, error) {
	l := &Logger{
		enabled: enabled,
		pushURL: remoteURL,
	}

	if !enabled {
		return l, nil
	}

	l.pushChan = make(chan *Entry, 1000)

	if logPath != "" {
		if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
			return nil, fmt.Errorf("create log directory: %w", err)
		}

		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("open log file: %w", err)
		}
		l.file = f
	}

	if remotePush && remoteURL != "" {
		go l.pushWorker()
	}

	return l, nil
}

func (l *Logger) Log(ctx context.Context, entry *Entry) error {
	if !l.enabled {
		return nil
	}

	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal audit entry: %w", err)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		if _, err := l.file.Write(append(data, '\n')); err != nil {
			return fmt.Errorf("write audit log: %w", err)
		}
	}

	if l.pushURL != "" {
		select {
		case l.pushChan <- entry:
		default:
		}
	}

	return nil
}

func (l *Logger) pushWorker() {
	for entry := range l.pushChan {
		_ = entry
	}
}

func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.pushChan != nil {
		close(l.pushChan)
	}

	if l.file != nil {
		return l.file.Close()
	}

	return nil
}
