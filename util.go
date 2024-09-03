package fdxproxy

import (
    "os"
    "log"
    "sync"
)

var logger = log.New(os.Stderr, "[PXY]", log.LstdFlags|log.Lmicroseconds|log.Lshortfile)

func SetLogger(path string) (*os.File, error) {
    logFile, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
    if err != nil {
        log.Fatalf("error opening file: %v", err)
        return nil, err
    }
    // defer logFile.Close()
    logger = log.New(logFile, "[PXY]", log.Default().Flags()|log.Lmicroseconds|log.Lshortfile)
    return logFile, nil
}

type CurrentMap struct {
    mu     sync.RWMutex
    data   map[string]interface{}
}

func NewCurrentMap() *CurrentMap {
    return &CurrentMap{
        data: make(map[string]interface{}),
    }
}

func (m *CurrentMap) Set(key string, value interface{}) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.data[key] = value
}

func (m *CurrentMap) Get(key string) (interface{}, bool) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    value, ok := m.data[key]
    return value, ok
}

func (m *CurrentMap) Delete(key string) {
    m.mu.Lock()
    defer m.mu.Unlock()
    delete(m.data, key)
}


type RequestMessage struct {
    ReqID   string            `json:"reqid"`
    Method  string            `json:"method"`
    URL     string            `json:"url"`
    // Headers have no concurrent scenario, use orignal map
    Headers map[string]string `json:"headers"`
    Body    string            `json:"body"`
}

type ResponseMessage struct {
    ReqID   string            `json:"reqid"`
    Status  int               `json:"status"`
    Headers map[string]string `json:"headers"`
    Body    string            `json:"body"`
}


const (
    ConnReqQueueSize = 10
    ConnResQueueSize = 10
    ResponseTimeout  = 10
)
