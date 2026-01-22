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

func TestServiceDisplayName(t *testing.T) {
	tt := []struct {
		name     string
		service  Service
		expected string
	}{
		{
			name:     "returns name when set",
			service:  Service{URL: "http://example.com", Name: "Example Service"},
			expected: "Example Service",
		},
		{
			name:     "falls back to URL when name is empty",
			service:  Service{URL: "http://example.com", Name: ""},
			expected: "http://example.com",
		},
		{
			name:     "falls back to URL when name is not set",
			service:  Service{URL: "http://example.com"},
			expected: "http://example.com",
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.service.DisplayName(); got != tc.expected {
				t.Errorf("DisplayName() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestPingFactoryPreservesName(t *testing.T) {
	s := Service{Type: "ping", URL: "http://example.com", Name: "My Service"}
	f := PingFactory{}
	p, err := f.Create(s)
	if err != nil {
		t.Fatalf("failed to create ping: %v", err)
	}
	if got := p.GetService().Name; got != "My Service" {
		t.Errorf("expected Name 'My Service', got %v", got)
	}
}

func TestGrepFactoryPreservesName(t *testing.T) {
	s := Service{Type: "grep", URL: "http://example.com", Regex: "test", Name: "My Grep Service"}
	f := GrepFactory{}
	g, err := f.Create(s)
	if err != nil {
		t.Fatalf("failed to create grep: %v", err)
	}
	if got := g.GetService().Name; got != "My Grep Service" {
		t.Errorf("expected Name 'My Grep Service', got %v", got)
	}
}

func TestTCPFactoryPreservesName(t *testing.T) {
	s := Service{Type: "tcp", URL: "localhost", Port: "8080", Name: "My TCP Service"}
	f := TCPFactory{}
	tc, err := f.Create(s)
	if err != nil {
		t.Fatalf("failed to create tcp: %v", err)
	}
	if got := tc.GetService().Name; got != "My TCP Service" {
		t.Errorf("expected Name 'My TCP Service', got %v", got)
	}
}

func TestICMPSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping ICMP test in short mode")
	}
	// Ping localhost which should always be reachable
	ic := ICMP{Service: Service{URL: "127.0.0.1"}}
	if err := ic.Status(); err != nil {
		t.Errorf("expected no error pinging localhost, got %v", err)
	}
}

func TestICMPFail(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping ICMP test in short mode")
	}
	// Use an invalid/unreachable IP address
	// 192.0.2.1 is a TEST-NET address that should be unreachable
	ic := ICMP{Service: Service{URL: "192.0.2.1"}}
	if err := ic.Status(); err == nil {
		t.Errorf("expected error for unreachable host, got nil")
	}
}

func TestICMPInvalidHost(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping ICMP test in short mode")
	}
	ic := ICMP{Service: Service{URL: "invalid.host.that.does.not.exist.local"}}
	actual := ic.Status()
	expected := ErrServiceUnavailable
	if actual != expected {
		t.Errorf("expected %v got %v", expected, actual)
	}
}

func TestICMPFactoryCreate(t *testing.T) {
	s := Service{Type: "icmp", URL: "8.8.8.8"}
	f := ICMPFactory{}
	actual, err := f.Create(s)
	if err != nil {
		t.Fatalf("failed create with error: %v", err)
	}

	expected := &ICMP{Service: Service{URL: "8.8.8.8"}}
	ap := reflect.ValueOf(actual)
	ep := reflect.ValueOf(expected)
	if ap.Pointer() == ep.Pointer() {
		t.Errorf("expected different pointers, got same: %v", ap.Pointer())
	}
}

func TestICMPFactoryCreateErr(t *testing.T) {
	s := Service{Type: "ping", URL: "8.8.8.8"}
	f := ICMPFactory{}
	_, err := f.Create(s)
	if err != ErrInvalidCreate {
		t.Errorf("expected ErrInvalidCreate, got %v", err)
	}
}

func TestICMPFactoryCreateEmptyHost(t *testing.T) {
	s := Service{Type: "icmp", URL: ""}
	f := ICMPFactory{}
	_, err := f.Create(s)
	if err != ErrHostRequired {
		t.Errorf("expected ErrHostRequired, got %v", err)
	}
}

func TestICMPGetService(t *testing.T) {
	s := Service{URL: "8.8.8.8", Name: "Google DNS"}
	ic := ICMP{Service: s}
	got := ic.GetService()
	if got.URL != s.URL {
		t.Errorf("expected URL %v, got %v", s.URL, got.URL)
	}
	if got.Name != s.Name {
		t.Errorf("expected Name %v, got %v", s.Name, got.Name)
	}
}

func TestICMPFactoryPreservesName(t *testing.T) {
	s := Service{Type: "icmp", URL: "8.8.8.8", Name: "Google DNS"}
	f := ICMPFactory{}
	ic, err := f.Create(s)
	if err != nil {
		t.Fatalf("failed to create icmp: %v", err)
	}
	if got := ic.GetService().Name; got != "Google DNS" {
		t.Errorf("expected Name 'Google DNS', got %v", got)
	}
}

func TestICMPFactoryRejectsInvalidHostname(t *testing.T) {
	tt := []struct {
		name string
		url  string
	}{
		{name: "semicolon injection", url: "example.com; rm -rf /"},
		{name: "pipe injection", url: "example.com | cat /etc/passwd"},
		{name: "backtick injection", url: "`whoami`.example.com"},
		{name: "dollar injection", url: "$(whoami).example.com"},
		{name: "ampersand injection", url: "example.com && ls"},
		{name: "newline injection", url: "example.com\nls"},
		{name: "space in hostname", url: "example .com"},
		{name: "starts with hyphen", url: "-example.com"},
		{name: "label ends with hyphen", url: "example-.com"},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			s := Service{Type: "icmp", URL: tc.url}
			f := ICMPFactory{}
			_, err := f.Create(s)
			if err != ErrInvalidHostname {
				t.Errorf("expected ErrInvalidHostname for URL %q, got %v", tc.url, err)
			}
		})
	}
}

func TestIsValidHostname(t *testing.T) {
	tt := []struct {
		name     string
		host     string
		expected bool
	}{
		{name: "valid IPv4", host: "192.168.1.1", expected: true},
		{name: "valid IPv6", host: "::1", expected: true},
		{name: "valid hostname", host: "example.com", expected: true},
		{name: "valid subdomain", host: "sub.example.com", expected: true},
		{name: "valid with hyphen", host: "my-host.example.com", expected: true},
		{name: "valid numeric label", host: "123.example.com", expected: true},
		{name: "localhost", host: "localhost", expected: true},
		{name: "empty string", host: "", expected: false},
		{name: "starts with hyphen", host: "-example.com", expected: false},
		{name: "ends with hyphen", host: "example-.com", expected: false},
		{name: "contains space", host: "example .com", expected: false},
		{name: "contains semicolon", host: "example;.com", expected: false},
		{name: "contains pipe", host: "example|.com", expected: false},
		{name: "contains backtick", host: "`whoami`", expected: false},
		{name: "empty label", host: "example..com", expected: false},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			if got := isValidHostname(tc.host); got != tc.expected {
				t.Errorf("isValidHostname(%q) = %v, want %v", tc.host, got, tc.expected)
			}
		})
	}
}
