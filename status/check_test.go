package status

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPingSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "<html><body>Hello World!</body></html>")
	}))
	defer ts.Close()

	tc := Ping{URL: ts.URL}
	if tc.Status() != nil {
		t.Fail()
	}
}

func TestPingFail(t *testing.T) {
	tc := Ping{URL: "garbage"}
	if tc.Status() == nil {
		t.Fail()
	}
}

func TestPingStatusCodeFail(t *testing.T) {
	tc := Ping{URL: "http://google.com/xyzabc"}
	actual := tc.Status()
	expected := ErrServiceUnavailable
	if actual != expected {
		t.Errorf("expected %v got %v", expected, actual)
	}
}

func TestGrepSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, "<html><body>Hello World!</body></html>")
	}))
	defer ts.Close()

	tc := Grep{URL: ts.URL, Regex: "Hello World!"}
	if tc.Status() != nil {
		t.Fail()
	}
}

func TestGrepFail(t *testing.T) {
	tc := Grep{URL: "garbage", Regex: "Hello World!"}
	if tc.Status() == nil {
		t.Fail()
	}
}

func TestGrepStatusCodeFail(t *testing.T) {
	tc := Grep{URL: "http://google.com/xyzabc", Regex: "Hello World!"}
	actual := tc.Status()
	expected := ErrServiceUnavailable
	if actual != expected {
		t.Errorf("expected %v got %v", expected, actual)
	}
}

func TestGrepRegexFail(t *testing.T) {
	tc := Grep{URL: "http://google.com", Regex: "Hello World!"}
	actual := tc.Status()
	expected := ErrRegexNotFound
	if actual != expected {
		t.Errorf("expected %v got %v", expected, actual)
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
