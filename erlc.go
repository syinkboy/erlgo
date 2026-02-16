package erlc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type LogLevel int

const (
	Info LogLevel = iota
	Warning
	ErrorLevel
)

type ERLC struct {
	GlobalKey  string
	ServerKey  string
	LogLevel   LogLevel
	mu         sync.Mutex
	requests   map[string][]*Request
	rateLimits map[string]*RateLimit
}

type Request struct {
	Method   string
	Endpoint string
	Body     any
	Response chan responseResult
}

type responseResult struct {
	Data []byte
	Err  error
}

type RateLimit struct {
	Remaining int
	Reset     time.Time
}


func New() *ERLC {
	return &ERLC{
		requests:   make(map[string][]*Request),
		rateLimits: make(map[string]*RateLimit),
		LogLevel:   Info,
	}
}

func (e *ERLC) log(msg string, level LogLevel) {
	if level < e.LogLevel {
		return
	}

	prefix := "[ERLC]"
	switch level {
	case Info:
		fmt.Println(prefix, msg)
	case Warning:
		fmt.Println("[WARN]", prefix, msg)
	case ErrorLevel:
		fmt.Println("[ERROR]", prefix, msg)
	}
}

func (e *ERLC) SetGlobalKey(key string) {
	e.GlobalKey = key
	e.log("Global key [HIDDEN]", Info)

	
	if e.requests == nil {
		e.requests = make(map[string][]*Request)
	}
	if e.rateLimits == nil {
		e.rateLimits = make(map[string]*RateLimit)
	}

	go e.processQueue()
}

func (e *ERLC) SetServerKey(key string) {
	e.ServerKey = key
	e.log("Server key [HIDDEN]", Info)

	
	if e.requests == nil {
		e.requests = make(map[string][]*Request)
	}
	if e.rateLimits == nil {
		e.rateLimits = make(map[string]*RateLimit)
	}

	go e.processQueue()
}

func (e *ERLC) request(method, endpoint string, body any) ([]byte, error) {
	respChan := make(chan responseResult)

	req := &Request{
		Method:   method,
		Endpoint: endpoint,
		Body:     body,
		Response: respChan,
	}

	e.mu.Lock()
	e.requests[endpoint] = append(e.requests[endpoint], req)
	e.mu.Unlock()

	result := <-respChan
	return result.Data, result.Err
}

func (e *ERLC) processQueue() {
	for {
		e.mu.Lock()
		for endpoint, queue := range e.requests {
			if len(queue) == 0 {
				continue
			}

			rl, exists := e.rateLimits[endpoint]
			if exists && rl.Remaining <= 0 && time.Now().Before(rl.Reset) {
				continue
			}

			req := queue[0]
			e.requests[endpoint] = queue[1:]
			e.mu.Unlock()

			data, err := e.executeRequest(req)

			req.Response <- responseResult{
				Data: data,
				Err:  err,
			}

			e.mu.Lock()
		}
		e.mu.Unlock()
		time.Sleep(25 * time.Millisecond)
	}
}

func (e *ERLC) executeRequest(r *Request) ([]byte, error) {
	url := fmt.Sprintf("https://api.policeroleplay.community/v1/%s", r.Endpoint)

	var bodyReader io.Reader
	if r.Body != nil {
		jsonData, _ := json.Marshal(r.Body)
		bodyReader = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(r.Method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if e.ServerKey != "" {
		req.Header.Set("Server-Key", e.ServerKey)
	}
	if e.GlobalKey != "" {
		req.Header.Set("Authorization", e.GlobalKey)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	remaining := parseInt(resp.Header.Get("X-RateLimit-Remaining"))
	retryAfter := parseRetry(resp.Header.Get("Retry-After"))

	e.mu.Lock()
	e.rateLimits[r.Endpoint] = &RateLimit{
		Remaining: remaining,
		Reset:     time.Now().Add(retryAfter),
	}
	e.mu.Unlock()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBytes))
	}

	return respBytes, nil
}

func parseInt(val string) int {
	i, _ := strconv.Atoi(val)
	return i
}

func parseRetry(val string) time.Duration {
	seconds, _ := strconv.Atoi(val)
	return time.Duration(seconds) * time.Second
}

func (e *ERLC) Server() (map[string]any, error) {
	data, err := e.request("GET", "server", nil)
	if err != nil {
		return nil, err
	}

	var result map[string]any
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}