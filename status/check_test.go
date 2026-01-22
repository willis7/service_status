package status

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestPingSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "<html><body>Hello World!</body></html>")
	}))
	defer ts.Close()

	tc := Ping{Service: Service{URL: ts.URL}}
	if err := tc.Status(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestPingFail(t *testing.T) {
	tc := Ping{Service: Service{URL: "garbage"}}
	if err := tc.Status(); err == nil {
		t.Errorf("expected error for invalid URL, got nil")
	}
}

func TestPingStatusCodeFail(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	tc := Ping{Service: Service{URL: ts.URL}}
	actual := tc.Status()
	expected := ErrServiceUnavailable
	if actual != expected {
		t.Errorf("expected %v got %v", expected, actual)
	}
}

func TestPingFactoryCreate(t *testing.T) {
	s := Service{Type: "ping", URL: "test"}
	p := PingFactory{}
	actual, err := p.Create(s)
	if err != nil {
		t.Fatalf("failed create with error: %v", err)
	}

	expected := &Ping{Service: Service{URL: "test"}}
	ap := reflect.ValueOf(actual)
	ep := reflect.ValueOf(expected)
	if ap.Pointer() == ep.Pointer() {
		t.Errorf("expected different pointers, got same: %v", ap.Pointer())
	}
}

func TestPingFactoryCreateErr(t *testing.T) {
	s := Service{Type: "grep", URL: "test"}
	p := PingFactory{}
	_, err := p.Create(s)
	if err != ErrInvalidCreate {
		t.Errorf("expected ErrInvalidCreate, got %v", err)
	}
}

func TestGrepSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "<html><body>Hello World!</body></html>")
	}))
	defer ts.Close()

	tc := Grep{Service: Service{URL: ts.URL, Regex: "Hello World!"}}
	if err := tc.Status(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestGrepFail(t *testing.T) {
	tc := Grep{Service: Service{URL: "garbage", Regex: "Hello World!"}}
	if err := tc.Status(); err == nil {
		t.Errorf("expected error for invalid URL, got nil")
	}
}

func TestGrepStatusCodeFail(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	tc := Grep{Service: Service{URL: ts.URL, Regex: "Hello World!"}}
	actual := tc.Status()
	expected := ErrServiceUnavailable
	if actual != expected {
		t.Errorf("expected %v got %v", expected, actual)
	}
}

func TestGrepRegexFail(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "<html><body>Different content</body></html>")
	}))
	defer ts.Close()

	tc := Grep{Service: Service{URL: ts.URL, Regex: "Hello World!"}}
	actual := tc.Status()
	expected := ErrRegexNotFound
	if actual != expected {
		t.Errorf("expected %v got %v", expected, actual)
	}
}

func TestGrepFactoryCreate(t *testing.T) {
	s := Service{Type: "grep", URL: "test", Regex: "hello"}
	p := GrepFactory{}
	actual, err := p.Create(s)
	if err != nil {
		t.Fatalf("failed create with error: %v", err)
	}

	expected := &Grep{Service: Service{URL: "test", Regex: "hello"}}
	ap := reflect.ValueOf(actual)
	ep := reflect.ValueOf(expected)
	if ap.Pointer() == ep.Pointer() {
		t.Errorf("expected different pointers, got same: %v", ap.Pointer())
	}
}

func TestGrepFactoryCreateErr(t *testing.T) {
	s := Service{Type: "ping", URL: "test", Regex: "hello"}
	p := GrepFactory{}
	_, err := p.Create(s)
	if err != ErrInvalidCreate {
		t.Errorf("expected ErrInvalidCreate, got %v", err)
	}
}

func TestValidStatus(t *testing.T) {
	tt := []struct {
		name   string
		code   int
		output bool
	}{
		{name: "status ok", code: http.StatusOK, output: true},
		{name: "bad gateway", code: http.StatusBadGateway, output: false},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			if got := validStatus(tc.code); got != tc.output {
				t.Errorf("validStatus(%d) = %v, want %v", tc.code, got, tc.output)
			}
		})
	}
}

func TestTCPSuccess(t *testing.T) {
	// Create a TCP listener for testing
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to create test listener: %v", err)
	}
	defer listener.Close()

	// Get the port that was assigned
	_, port, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatalf("failed to get port: %v", err)
	}

	tc := TCP{Service: Service{URL: "127.0.0.1", Port: port}}
	if err := tc.Status(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestTCPFail(t *testing.T) {
	// Use a port that's unlikely to have a listener
	tc := TCP{Service: Service{URL: "127.0.0.1", Port: "59999"}}
	if err := tc.Status(); err == nil {
		t.Errorf("expected error for closed port, got nil")
	}
}

func TestTCPInvalidHost(t *testing.T) {
	tc := TCP{Service: Service{URL: "invalid.host.that.does.not.exist", Port: "80"}}
	actual := tc.Status()
	expected := ErrServiceUnavailable
	if actual != expected {
		t.Errorf("expected %v got %v", expected, actual)
	}
}

func TestTCPFactoryCreate(t *testing.T) {
	s := Service{Type: "tcp", URL: "localhost", Port: "8080"}
	f := TCPFactory{}
	actual, err := f.Create(s)
	if err != nil {
		t.Fatalf("failed create with error: %v", err)
	}

	expected := &TCP{Service: Service{URL: "localhost", Port: "8080"}}
	ap := reflect.ValueOf(actual)
	ep := reflect.ValueOf(expected)
	if ap.Pointer() == ep.Pointer() {
		t.Errorf("expected different pointers, got same: %v", ap.Pointer())
	}
}

func TestTCPFactoryCreateErr(t *testing.T) {
	s := Service{Type: "ping", URL: "localhost", Port: "8080"}
	f := TCPFactory{}
	_, err := f.Create(s)
	if err != ErrInvalidCreate {
		t.Errorf("expected ErrInvalidCreate, got %v", err)
	}
}

func TestTCPFactoryCreateEmptyPort(t *testing.T) {
	s := Service{Type: "tcp", URL: "localhost", Port: ""}
	f := TCPFactory{}
	_, err := f.Create(s)
	if err == nil {
		t.Errorf("expected error for empty port, got nil")
	}
}

func TestTCPGetService(t *testing.T) {
	s := Service{URL: "localhost", Port: "8080"}
	tc := TCP{Service: s}
	got := tc.GetService()
	if got.URL != s.URL {
		t.Errorf("expected URL %v, got %v", s.URL, got.URL)
	}
	if got.Port != s.Port {
		t.Errorf("expected Port %v, got %v", s.Port, got.Port)
	}
}
