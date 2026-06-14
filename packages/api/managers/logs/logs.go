package logs

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

var (
	dir     string
	logFile string
	mu      sync.Mutex
)

func Init(s string) error {
	dir = s
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	logFile = filepath.Join(dir, "latest.log")
	os.Remove(logFile)
	return nil
}

func Log(s string, a ...any) error {
	mu.Lock()
	defer mu.Unlock()
	msg := s
	if len(a) > 0 {
		msg = fmt.Sprintf(s, a...)
	}
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(msg + "\n")
	return err
}
