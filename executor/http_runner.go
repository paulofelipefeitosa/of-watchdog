package executor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"strconv"
	"syscall"
	"time"
)

// HTTPFunctionRunner creates and maintains one process responsible for handling all calls
type HTTPFunctionRunner struct {
	ExecTimeout    time.Duration // ExecTimeout the maximum duration or an upstream function call
	ReadTimeout    time.Duration // ReadTimeout for HTTP server
	WriteTimeout   time.Duration // WriteTimeout for HTTP Server
	Process        string        // Process to run as fprocess
	ProcessArgs    []string      // ProcessArgs to pass to command
	Command        *exec.Cmd
	StdinPipe      io.WriteCloser
	StdoutPipe     io.ReadCloser
	Client         *http.Client
	UpstreamURL    *url.URL
	BufferHTTPBody bool
	StartupTime    int64
	CRIUExec       bool
	RestoreLogPath string
}

// Start forks the process used for processing incoming requests
func (f *HTTPFunctionRunner) Start() error {
	cmd := exec.Command(f.Process, f.ProcessArgs...)

	var stdinErr error
	var stdoutErr error

	f.Command = cmd
	f.StdinPipe, stdinErr = cmd.StdinPipe()
	if stdinErr != nil {
		return stdinErr
	}

	f.StdoutPipe, stdoutErr = cmd.StdoutPipe()
	if stdoutErr != nil {
		return stdoutErr
	}

	errPipe, _ := cmd.StderrPipe()

	// Logs lines from stderr and stdout to the stderr and stdout of this process
	bindLoggingPipe("stderr", errPipe, os.Stderr)
	bindLoggingPipe("stdout", f.StdoutPipe, os.Stdout)

	f.Client = makeProxyClient(f.ExecTimeout)

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGTERM)

		<-sig
		cmd.Process.Signal(syscall.SIGTERM)

	}()

	err := cmd.Start()
	go func() {
		err := cmd.Wait()
		if err != nil {
			log.Fatalf("Forked function has terminated: %s", err.Error())
		}
	}()

	return err
}

// Run a function with a long-running process with a HTTP protocol for communication
func (f *HTTPFunctionRunner) Run(req FunctionRequest, contentLength int64, r *http.Request, w http.ResponseWriter) error {
	startedTime := time.Now()

	upstreamURL := f.UpstreamURL.String()

	if len(r.RequestURI) > 0 {
		upstreamURL += r.RequestURI
	}

	var body io.Reader
	if f.BufferHTTPBody {
		reqBody, _ := ioutil.ReadAll(r.Body)
		body = bytes.NewReader(reqBody)
	} else {
		body = r.Body
	}

	request, _ := http.NewRequest(r.Method, upstreamURL, body)
	for h := range r.Header {
		request.Header.Set(h, r.Header.Get(h))
	}

	request.Host = r.Host
	copyHeaders(request.Header, &r.Header)

	var reqCtx context.Context
	var cancel context.CancelFunc

	if f.ExecTimeout.Nanoseconds() > 0 {
		reqCtx, cancel = context.WithTimeout(context.Background(), f.ExecTimeout)
	} else {
		reqCtx = context.Background()
		cancel = func() {

		}
	}

	defer cancel()

	res, err := f.Client.Do(request.WithContext(reqCtx))

	if err != nil {
		log.Printf("Upstream HTTP request error: %s\n", err.Error())

		// Error unrelated to context / deadline
		if reqCtx.Err() == nil {
			w.Header().Set("X-Duration-Seconds", fmt.Sprintf("%f", time.Since(startedTime).Seconds()))

			w.WriteHeader(http.StatusInternalServerError)

			return nil
		}

		select {
		case <-reqCtx.Done():
			{
				if reqCtx.Err() != nil {
					// Error due to timeout / deadline
					log.Printf("Upstream HTTP killed due to exec_timeout: %s\n", f.ExecTimeout)
					w.Header().Set("X-Duration-Seconds", fmt.Sprintf("%f", time.Since(startedTime).Seconds()))

					w.WriteHeader(http.StatusGatewayTimeout)
					return nil
				}

			}
		}

		w.Header().Set("X-Duration-Seconds", fmt.Sprintf("%f", time.Since(startedTime).Seconds()))
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}

	copyHeaders(w.Header(), &res.Header)

	w.Header().Set("X-Duration-Seconds", fmt.Sprintf("%f", time.Since(startedTime).Seconds()))
	if f.StartupTime == -1 {
		f.StartupTime = getStartupTime(f.CRIUExec, f.RestoreLogPath, res.Header.Get("X-App-Startup-Time"))
	}
	w.Header().Set("X-App-Startup-Time", fmt.Sprintf("%d", f.StartupTime))

	w.WriteHeader(res.StatusCode)
	if res.Body != nil {
		defer res.Body.Close()

		bodyBytes, bodyErr := ioutil.ReadAll(res.Body)
		if bodyErr != nil {
			log.Println("read body err", bodyErr)
		}
		w.Write(bodyBytes)
	}

	log.Printf("%s %s - %s - ContentLength: %d", r.Method, r.RequestURI, res.Status, res.ContentLength)

	return nil
}

func getStartupTime(CRIUExec bool, restoreLogPath string, strAppStartupTS string) int64 {
	if CRIUExec {
		lastLine := string(tail(restoreLogPath))
		re := regexp.MustCompile(`\((\d+\.\d+)\) Writing stats`)
		log.Printf("Last Line: %v", lastLine)
		result := re.FindAllStringSubmatch(lastLine, -1)[0][1]
		s, err := strconv.ParseFloat(result, 64)
		if err != nil {
			log.Printf("Found %s, but cannot convert %s to float64\n", lastLine, result)
			return 0
		}
		log.Printf("Found %s and converted %s to %f\n", lastLine, result, s)
		return int64(s * 1e9)
	} else {
		strContainerStartupTS := os.Getenv("CONTAINER_STARTUP_TS")
		containerStartupTS, err := strconv.ParseInt(strContainerStartupTS, 10, 64)
		if err != nil {
			log.Printf("Cannot convert container startup timestamp value (%s) to int64, due to %v\n", strContainerStartupTS, err.Error())
			return 0
		}
		appStartupTS, err := strconv.ParseInt(strAppStartupTS, 10, 64)
		if err != nil {
			log.Printf("Cannot convert app startup timestamp value (%s) to int64, due to %v\n", strAppStartupTS, err.Error())
			return 0
		}
		return appStartupTS - containerStartupTS
	}
}

func copyHeaders(destination http.Header, source *http.Header) {
	for k, v := range *source {
		vClone := make([]string, len(v))
		copy(vClone, v)
		(destination)[k] = vClone
	}
}

func makeProxyClient(dialTimeout time.Duration) *http.Client {
	proxyClient := http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   dialTimeout,
				KeepAlive: 10 * time.Second,
			}).DialContext,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   100,
			DisableKeepAlives:     false,
			IdleConnTimeout:       500 * time.Millisecond,
			ExpectContinueTimeout: 1500 * time.Millisecond,
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return &proxyClient
}
