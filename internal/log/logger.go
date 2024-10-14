package log

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/cooldogedev/spectral/internal/protocol"
)

type Logger interface {
	SetConnectionID(connectionID protocol.ConnectionID)
	Log(event string, params ...any)
	Close()
}

type FileLogger struct {
	perspective Perspective
	file        *os.File
	writer      *bufio.Writer
	buffered    []string
	mu          sync.Mutex
}

func (f *FileLogger) SetConnectionID(connectionID protocol.ConnectionID) {
	f.mu.Lock()
	b := make([]byte, 20)
	_, _ = rand.Read(b)
	file, err := os.OpenFile(path.Join(os.Getenv("SLOG_DIR"), fmt.Sprintf("%s-%d.%s.log", hex.EncodeToString(b), connectionID, perspectiveToString(f.perspective))), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err == nil {
		f.file = file
		f.writer = bufio.NewWriter(file)
		for _, buffered := range f.buffered {
			_, _ = f.writer.WriteString(buffered)
		}
		f.buffered = f.buffered[:0]
		f.buffered = nil
	}
	f.mu.Unlock()
}

func (f *FileLogger) Log(event string, params ...any) {
	var pairs []string
	for i := 0; i < len(params); i += 2 {
		pairs = append(pairs, fmt.Sprintf("%v=%v", params[i], params[i+1]))
	}

	log := fmt.Sprintf("timestamp=%s event=%s %s\n", time.Now().Format(time.RFC3339), event, strings.Join(pairs, " "))
	f.mu.Lock()
	if f.writer != nil {
		_, _ = f.writer.WriteString(log)
	} else {
		f.buffered = append(f.buffered, log)
	}
	f.mu.Unlock()
}

func (f *FileLogger) Close() {
	f.mu.Lock()
	if f.writer != nil {
		_ = f.writer.Flush()
		_ = f.file.Close()
	}
	f.mu.Unlock()
}

func NewLogger(perspective Perspective) Logger {
	dir := os.Getenv("SLOG_DIR")
	if dir == "" {
		return NopLogger{}
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return NopLogger{}
	}
	return &FileLogger{perspective: perspective}
}

type NopLogger struct{}

var _ Logger = NopLogger{}

func (NopLogger) SetConnectionID(_ protocol.ConnectionID) {}
func (NopLogger) Log(_ string, _ ...any)                  {}
func (NopLogger) Close()                                  {}
