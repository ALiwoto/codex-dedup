package localProxyTests

import (
	"bufio"
	"bytes"
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
	localURL, proxyServer := startLocalProxy(t, providerURL)
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

	snapshot := proxyServer.DedupMeasurementSnapshot()
	if snapshot.MeasuredRequestCount != 1 {
		t.Fatalf("unexpected measured request count: %d", snapshot.MeasuredRequestCount)
	}
	if snapshot.BodyBytes != uint64(len(requestBody)) {
		t.Fatalf("unexpected measured body bytes: %d", snapshot.BodyBytes)
	}
	if snapshot.NewChunkBytes+snapshot.ReusableChunkBytes != snapshot.BodyBytes {
		t.Fatal("measured chunks do not account for the complete request body")
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
	localURL, _ := startLocalProxy(t, providerURL)

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

func TestDedupMeasurementReusesAnIdenticalBody(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer provider.Close()

	providerURL, err := url.Parse(provider.URL)
	if err != nil {
		t.Fatalf("parse provider URL: %v", err)
	}
	localURL, proxyServer := startLocalProxy(t, providerURL)
	body := deterministicRequestBody(2 * 1024 * 1024)

	sendRequestBody(t, localURL, body)
	first := proxyServer.DedupMeasurementSnapshot()
	sendRequestBody(t, localURL, body)
	second := proxyServer.DedupMeasurementSnapshot()

	if second.MeasuredRequestCount != 2 {
		t.Fatalf("unexpected measured request count: %d", second.MeasuredRequestCount)
	}
	if second.NewChunkBytes != first.NewChunkBytes {
		t.Fatalf("identical request added %d new bytes", second.NewChunkBytes-first.NewChunkBytes)
	}
	if second.ReusableChunkBytes-first.ReusableChunkBytes != uint64(len(body)) {
		t.Fatalf(
			"identical request reused %d of %d bytes",
			second.ReusableChunkBytes-first.ReusableChunkBytes,
			len(body),
		)
	}
}

func TestConcurrentDedupMeasurementsUseOneSimulatedCache(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer provider.Close()

	providerURL, err := url.Parse(provider.URL)
	if err != nil {
		t.Fatalf("parse provider URL: %v", err)
	}
	localURL, proxyServer := startLocalProxy(t, providerURL)
	body := deterministicRequestBody(1024 * 1024)
	const requestCount = 6

	var requests sync.WaitGroup
	requests.Add(requestCount)
	for range requestCount {
		go func() {
			defer requests.Done()
			sendRequestBody(t, localURL, body)
		}()
	}
	requests.Wait()

	snapshot := proxyServer.DedupMeasurementSnapshot()
	expectedBodyBytes := uint64(requestCount * len(body))
	if snapshot.MeasuredRequestCount != requestCount {
		t.Fatalf("unexpected measured request count: %d", snapshot.MeasuredRequestCount)
	}
	if snapshot.BodyBytes != expectedBodyBytes {
		t.Fatalf("unexpected measured body bytes: %d", snapshot.BodyBytes)
	}
	if snapshot.NewChunkBytes+snapshot.ReusableChunkBytes != expectedBodyBytes {
		t.Fatal("concurrent measurements did not account for every request byte")
	}
	if snapshot.NewChunkBytes > uint64(len(body)) {
		t.Fatalf("identical concurrent requests added too many new bytes: %d", snapshot.NewChunkBytes)
	}
	if snapshot.SimulatedCacheBytes != snapshot.NewChunkBytes {
		t.Fatal("simulated cache size differs from unique uploaded bytes")
	}
}

func startLocalProxy(t *testing.T, providerURL *url.URL) (string, *localProxy.LocalProxy) {
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

	return "http://" + listener.Addr().String(), proxyServer
}

func sendRequestBody(t *testing.T, localURL string, body []byte) {
	t.Helper()

	request, err := http.NewRequest(http.MethodPost, localURL+"/opaque", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("send request: %v", err)
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, response.Body)
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("unexpected response status: %d", response.StatusCode)
	}
}

func deterministicRequestBody(size int) []byte {
	body := make([]byte, size)
	state := uint64(0x13198a2e03707344)
	for index := range body {
		state ^= state << 13
		state ^= state >> 7
		state ^= state << 17
		body[index] = byte(state >> 56)
	}
	return body
}
