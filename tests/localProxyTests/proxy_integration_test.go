package localProxyTests

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ALiwoto/codex-dedup/src/localProxy"
)

type forwardedRequest struct {
	Method        string
	RequestURI    string
	Host          string
	OpaqueHeader  string
	RemovedHeader string
	Body          string
}

func TestLocalProxyForwardsOpaqueHTTPRequest(t *testing.T) {
	requests := make(chan *forwardedRequest, 1)
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read provider request body: %v", err)
			return
		}

		requests <- &forwardedRequest{
			Method:        r.Method,
			RequestURI:    r.RequestURI,
			Host:          r.Host,
			OpaqueHeader:  r.Header.Get("X-Opaque-Metadata"),
			RemovedHeader: r.Header.Get("X-Remove-Me"),
			Body:          string(body),
		}

		w.Header().Set("X-Provider-Metadata", "preserved")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("provider-response"))
	}))
	defer provider.Close()

	providerURL, err := url.Parse(provider.URL + "/provider-prefix")
	if err != nil {
		t.Fatalf("parse provider URL: %v", err)
	}
	localURL := startLocalProxy(t, providerURL)
	requestBody := strings.Repeat("x", 5*1024*1024)

	request, err := http.NewRequest(
		http.MethodPatch,
		localURL+"/any/version/resource?encoded=a%2Fb&mode=test",
		strings.NewReader(requestBody),
	)
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	request.Header.Set("Content-Type", "application/octet-stream")
	request.Header.Set("X-Opaque-Metadata", "preserved")
	request.Header.Set("Connection", "X-Remove-Me")
	request.Header.Set("X-Remove-Me", "must-not-reach-provider")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("send request through local proxy: %v", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read local proxy response: %v", err)
	}
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("unexpected response status: %d", response.StatusCode)
	}
	if string(responseBody) != "provider-response" {
		t.Fatalf("unexpected response body: %q", responseBody)
	}
	if response.Header.Get("X-Provider-Metadata") != "preserved" {
		t.Fatalf("provider response header was not preserved")
	}

	forwarded := <-requests
	if forwarded.Method != http.MethodPatch {
		t.Fatalf("unexpected method: %q", forwarded.Method)
	}
	if forwarded.RequestURI != "/provider-prefix/any/version/resource?encoded=a%2Fb&mode=test" {
		t.Fatalf("unexpected provider request target: %q", forwarded.RequestURI)
	}
	if forwarded.Host != providerURL.Host {
		t.Fatalf("unexpected provider host: %q", forwarded.Host)
	}
	if forwarded.OpaqueHeader != "preserved" {
		t.Fatalf("opaque header was not preserved")
	}
	if forwarded.RemovedHeader != "" {
		t.Fatalf("connection-scoped header reached provider")
	}
	if forwarded.Body != requestBody {
		t.Fatalf("provider received a different request body")
	}
}

func TestLocalProxyStreamsServerSentEvents(t *testing.T) {
	releaseSecondEvent := make(chan struct{})
	var releaseOnce sync.Once
	releaseProvider := func() {
		releaseOnce.Do(func() {
			close(releaseSecondEvent)
		})
	}
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)

		_, _ = w.Write([]byte("data: first\n\n"))
		flusher.Flush()
		<-releaseSecondEvent
		_, _ = w.Write([]byte("data: second\n\n"))
		flusher.Flush()
	}))
	defer provider.Close()
	defer releaseProvider()

	providerURL, err := url.Parse(provider.URL)
	if err != nil {
		t.Fatalf("parse provider URL: %v", err)
	}
	localURL := startLocalProxy(t, providerURL)

	client := &http.Client{Timeout: 2 * time.Second}
	response, err := client.Get(localURL + "/arbitrary/events")
	if err != nil {
		t.Fatalf("open SSE response: %v", err)
	}
	defer response.Body.Close()
	if response.Header.Get("Content-Type") != "text/event-stream" {
		t.Fatalf("unexpected SSE content type: %q", response.Header.Get("Content-Type"))
	}

	reader := bufio.NewReader(response.Body)
	firstEvent, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read first SSE event before provider completed: %v", err)
	}
	if firstEvent != "data: first\n" {
		t.Fatalf("unexpected first SSE event: %q", firstEvent)
	}

	releaseProvider()
	_, _ = reader.ReadString('\n')
	secondEvent, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("read second SSE event: %v", err)
	}
	if secondEvent != "data: second\n" {
		t.Fatalf("unexpected second SSE event: %q", secondEvent)
	}
}

func startLocalProxy(t *testing.T, providerURL *url.URL) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("create local proxy listener: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	finished := make(chan error, 1)
	proxyServer := localProxy.NewLocalProxy(&localProxy.LocalProxyOptions{
		BindAddress: listener.Addr().String(),
		ProviderURL: providerURL,
	})
	go func() {
		finished <- proxyServer.RunWithListener(ctx, listener)
	}()

	t.Cleanup(func() {
		cancel()
		select {
		case err := <-finished:
			if err != nil {
				t.Errorf("stop local proxy: %v", err)
			}
		case <-time.After(2 * time.Second):
			t.Error("local proxy did not stop")
		}
	})

	return "http://" + listener.Addr().String()
}
