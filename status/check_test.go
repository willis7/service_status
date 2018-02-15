package status

import (
	"io"
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
	if tc.Status() != nil {
		t.Fail()
	}
}

func TestPingFail(t *testing.T) {
	tc := Ping{Service: Service{URL: "garbage"}}
	if tc.Status() == nil {
		t.Fail()
	}
}

func TestPingStatusCodeFail(t *testing.T) {
	tc := Ping{Service: Service{URL: "http://google.com/xyzabc"}}
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
		t.Errorf("expected %v got %v", ap.Pointer(), ep.Pointer())
	}
}

func TestPingFactoryCreateErr(t *testing.T) {
	s := Service{Type: "grep", URL: "test"}
	p := PingFactory{}
	_, err := p.Create(s)
	if err != ErrInvalidCreate {
		t.Fatalf("failed create with error: %v", err)
	}
}

func TestGrepSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "<html><body>Hello World!</body></html>")
	}))
	defer ts.Close()

	tc := Grep{Service: Service{URL: ts.URL, Regex: "Hello World!"}}
	if tc.Status() != nil {
		t.Fail()
	}
}

func TestGrepFail(t *testing.T) {
	tc := Grep{Service: Service{URL: "garbage", Regex: "Hello World!"}}
	if tc.Status() == nil {
		t.Fail()
	}
}

func TestGrepStatusCodeFail(t *testing.T) {
	tc := Grep{Service: Service{URL: "http://google.com/xyzabc", Regex: "Hello World!"}}
	actual := tc.Status()
	expected := ErrServiceUnavailable
	if actual != expected {
		t.Errorf("expected %v got %v", expected, actual)
	}
}

func TestGrepRegexFail(t *testing.T) {
	tc := Grep{Service: Service{URL: "http://google.com", Regex: "Hello World!"}}
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
		t.Fail()
	}
}

func TestGrepFactoryCreateErr(t *testing.T) {
	s := Service{Type: "ping", URL: "test", Regex: "hello"}
	p := GrepFactory{}
	_, err := p.Create(s)
	if err != ErrInvalidCreate {
		t.Fail()
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
			if validStatus(tc.code) != tc.output {
				t.Fail()
			}
		})
	}
}
