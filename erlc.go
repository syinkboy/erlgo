package erlc

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

type LogLevel int

const (
	info LogLevel = iota
	Warning
	ErrorLevel
)

type ERLC struct {
	GlobalKey string
	ServerKey string
	LogLevel LogLevel

	mu sync.Mutex
	request map[string][]*Request
	rateLimits map[string][]*RateLimit
}

type Request struct {
	Method string 
	EndPoint string
	Body any
	Process func([]byte) any
	Response chan any
}

type RateLimit struct {
	Updated time.Time
	Retry time.Duration
	Remaining int
	Reset time.Time
}

func (e *ERLC) log(msg string, level LogLevel) {
	if level < e.LogLevel {
		return
	}

	prefix := "[ERLC]"
	swtich level {
	case info:
		fmt.println(prefix, msg)
	case Warning:
		fmt.printLn("[WARN]", prefix, msg)
	case ErrorLevel:
		fmt.printLn("[ERROR]", prefix, msg)
	}
}

func (e *ERLC) SetGlobalKey(key string) {
	e.GlobalKey = key
	e.Log("Global key [HIDDEN]", info)
}

func (e *ERLC) SetServerKey(key string) {
	e.ServerKey = key
	e.log("Server ley [HIDDEN]", info)
}

func (e *ERLC) request(method, endpoint string, body any) ([]byte, error) {
	url := fmt.Sprintf("https://api.policeroleplay.community/v1/%s", endpoint)

	var bodyReader io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		bodyReader = bytes.NewBuffer(jsonData)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Tyepe", "application/json")
	if e.ServerKey != "" {
		req.Header.Set("Server-Key", e.ServerKey)
	}
	if e.GlobalKey != "" {
		req.Header.Set("Authorization", e.GlobalKey)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil  {
		return nil, err
	}
	defer reso.Body.Close()

	respBytes, err := io:readAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d %s", resp.Statucode, string(respBytes))
	}

	return respBytes, nil
}
func (e *ERLC) server() (map[string]any, error) {
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