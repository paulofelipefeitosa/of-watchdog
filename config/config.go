package config

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// WatchdogConfig configuration for a watchdog.
type WatchdogConfig struct {
	TCPPort          int
	HTTPReadTimeout  time.Duration
	HTTPWriteTimeout time.Duration
	ExecTimeout      time.Duration

	FunctionProcess  string
	ContentType      string
	InjectCGIHeaders bool
	OperationalMode  int
	SuppressLock     bool
	UpstreamURL      string
	StaticPath       string

	// BufferHTTPBody buffers the HTTP body in memory
	// to prevent transfer type of chunked encoding
	// which some servers do not support.
	BufferHTTPBody bool

	// MetricsPort TCP port on which to serve HTTP Prometheus metrics
	MetricsPort int

	// MaxInflight limits the number of simultaneous
	// requests that the watchdog allows concurrently.
	// Any request which exceeds this limit will
	// have an immediate response of 429.
	MaxInflight int

	CRIUExec       bool
	RestoreLogPath string
}

// Process returns a string for the process and a slice for the arguments from the FunctionProcess.
func (w WatchdogConfig) Process() (string, []string) {
	parts := strings.Split(w.FunctionProcess, " ")

	if len(parts) > 1 {
		return parts[0], parts[1:]
	}

	return parts[0], []string{}
}

// New create config based upon environmental variables.
func New(env []string) WatchdogConfig {

	envMap := mapEnv(env)

	var (
		functionProcess string
		upstreamURL     string
	)

	if val, exists := envMap["fprocess"]; exists {
		functionProcess = val
	}

	if val, exists := envMap["function_process"]; exists {
		functionProcess = val
	}

	if val, exists := envMap["upstream_url"]; exists {
		upstreamURL = val
	}

	if val, exists := envMap["http_upstream_url"]; exists {
		upstreamURL = val
	}

	contentType := "application/octet-stream"
	if val, exists := envMap["content_type"]; exists {
		contentType = val
	}

	staticPath := "/home/app/public"
	if val, exists := envMap["static_path"]; exists {
		staticPath = val
	}

	restoreLogPath := "restore.log"
	if val, exists := envMap["restore_log_path"]; exists {
		restoreLogPath = val
	}

	config := WatchdogConfig{
		TCPPort:          getInt(envMap, "port", 8080),
		HTTPReadTimeout:  getDuration(envMap, "read_timeout", time.Second*10),
		HTTPWriteTimeout: getDuration(envMap, "write_timeout", time.Second*10),
		FunctionProcess:  functionProcess,
		StaticPath:       staticPath,
		InjectCGIHeaders: true,
		ExecTimeout:      getDuration(envMap, "exec_timeout", time.Second*10),
		OperationalMode:  ModeStreaming,
		ContentType:      contentType,
		SuppressLock:     getBool(envMap, "suppress_lock"),
		UpstreamURL:      upstreamURL,
		BufferHTTPBody:   getBools(envMap, "buffer_http", "http_buffer_req_body"),
		MetricsPort:      8081,
		MaxInflight:      getInt(envMap, "max_inflight", 0),
		CRIUExec:         getBool(envMap, "criu_exec"),
		RestoreLogPath:   restoreLogPath,
	}

	if val := envMap["mode"]; len(val) > 0 {
		config.OperationalMode = WatchdogModeConst(val)
	}

	return config
}

func mapEnv(env []string) map[string]string {
	mapped := map[string]string{}

	for _, val := range env {
		sep := strings.Index(val, "=")

		if sep > 0 {
			key := val[0:sep]
			value := val[sep+1:]
			mapped[key] = value
		} else {
			fmt.Println("Bad environment: " + val)
		}
	}

	return mapped
}

func getDuration(env map[string]string, key string, defaultValue time.Duration) time.Duration {
	result := defaultValue
	if val, exists := env[key]; exists {
		parsed, _ := time.ParseDuration(val)
		result = parsed

	}

	return result
}

func getInt(env map[string]string, key string, defaultValue int) int {
	result := defaultValue
	if val, exists := env[key]; exists {
		parsed, _ := strconv.Atoi(val)
		result = parsed

	}

	return result
}

func getBool(env map[string]string, key string) bool {
	if env[key] == "true" {
		return true
	}

	return false
}

func getBools(env map[string]string, key ...string) bool {
	v := false
	for _, k := range key {
		if getBool(env, k) == true {
			v = true
			break
		}
	}
	return v
}
