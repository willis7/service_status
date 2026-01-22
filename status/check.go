package status

import (
	"errors"
	"io"
	"net"
	"net/http"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// ErrServiceUnavailable implements error signifying a service is unavailable.
var (
	ErrServiceUnavailable = errors.New("commands: service unavailable")
	ErrRegexNotFound      = errors.New("commands: regex not found")
	ErrInvalidCreate      = errors.New("commands: invalid type for create")
	ErrHostRequired       = errors.New("commands: host is required for icmp check")
	ErrInvalidHostname    = errors.New("commands: invalid hostname for icmp check")
)

// icmpPingTimeoutSeconds is the timeout in seconds for ICMP ping attempts (string for CLI argument).
const icmpPingTimeoutSeconds = "5"

// tcpDialTimeout is the timeout duration for TCP connection attempts.
const tcpDialTimeout = 10 * time.Second

// Service represents a single endpoint to be tested.
type Service struct {
	Type  string `json:"type"`
	URL   string `json:"url"`
	Port  string `json:"port,omitempty"`
	Regex string `json:"regex,omitempty"`
	Name  string `json:"name,omitempty"`
}

// DisplayName returns the Name if set, otherwise falls back to the URL.
func (s *Service) DisplayName() string {
	if s.Name != "" {
		return s.Name
	}
	return s.URL
}

// Pinger is an interface which describes how
// to test a service status.
type Pinger interface {
	GetService() *Service
	Status() error
}

// PingerFactory is a single method interface that describes
// how to create a Pinger object.
type PingerFactory interface {
	Create(Service) (Pinger, error)
}

// Ping performs a ping-like test of a
// service's availability.
type Ping struct {
	Service
}

// GetService returns the Service pointer.
func (p *Ping) GetService() *Service {
	return &p.Service
}

// Status sends a HEAD http request and checks for a valid
// http response code.
func (p *Ping) Status() error {
	resp, err := http.Head(p.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if !validStatus(resp.StatusCode) {
		return ErrServiceUnavailable
	}

	return nil
}

// PingFactory implements the PingerFactory interface.
type PingFactory struct{}

// Create returns a pointer to a Pinger.
func (f *PingFactory) Create(s Service) (Pinger, error) {
	if s.Type != "ping" {
		return nil, ErrInvalidCreate
	}
	return &Ping{
		Service: Service{URL: s.URL, Name: s.Name},
	}, nil
}

// Grep checks a response body for a value.
type Grep struct {
	Service
}

// GetService returns the Service pointer.
func (g *Grep) GetService() *Service {
	return &g.Service
}

// Status requests a page given a URL and checks the response for
// a value matching the regex.
func (g *Grep) Status() error {
	// hit the URL and get a response
	resp, err := http.Get(g.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if !validStatus(resp.StatusCode) {
		return ErrServiceUnavailable
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	re := regexp.MustCompile(g.Regex)
	if !re.Match(bodyBytes) {
		return ErrRegexNotFound
	}

	return nil
}

// GrepFactory implements the PingerFactory interface.
type GrepFactory struct{}

// Create returns a pointer to a Pinger.
func (f *GrepFactory) Create(s Service) (Pinger, error) {
	if s.Type != "grep" {
		return nil, ErrInvalidCreate
	}

	return &Grep{
		Service: Service{URL: s.URL, Regex: s.Regex, Name: s.Name},
	}, nil
}

// validStatus checks the input against a list of known-good
// http status codes and returns a bool.
func validStatus(code int) bool {
	return code == http.StatusOK
}

// TCP checks if a TCP port is open and accepting connections.
type TCP struct {
	Service
}

// GetService returns the Service pointer.
func (t *TCP) GetService() *Service {
	return &t.Service
}

// Status attempts to establish a TCP connection to the host:port
// and returns an error if the connection fails.
func (t *TCP) Status() error {
	address := net.JoinHostPort(t.URL, t.Port)
	conn, err := net.DialTimeout("tcp", address, tcpDialTimeout)
	if err != nil {
		return ErrServiceUnavailable
	}
	defer conn.Close()
	return nil
}

// TCPFactory implements the PingerFactory interface.
type TCPFactory struct{}

// Create returns a pointer to a Pinger.
func (f *TCPFactory) Create(s Service) (Pinger, error) {
	if s.Type != "tcp" {
		return nil, ErrInvalidCreate
	}
	if s.Port == "" {
		return nil, errors.New("commands: port is required for tcp check")
	}
	return &TCP{
		Service: Service{URL: s.URL, Port: s.Port, Name: s.Name},
	}, nil
}

// ICMP performs a true ICMP ping to check host reachability at the network level.
// It shells out to the system ping command for cross-platform compatibility
// and to avoid requiring elevated privileges.
type ICMP struct {
	Service
}

// GetService returns the Service pointer.
func (i *ICMP) GetService() *Service {
	return &i.Service
}

// Status executes an ICMP ping against the host and returns an error
// if the host is unreachable.
func (i *ICMP) Status() error {
	var cmd *exec.Cmd

	// Construct platform-specific ping command
	// macOS/BSD uses -c for count and -t for timeout
	// Linux uses -c for count and -W for timeout
	// Note: Windows is not currently supported (uses -n and -w with different semantics)
	if runtime.GOOS == "darwin" || runtime.GOOS == "freebsd" {
		cmd = exec.Command("ping", "-c", "1", "-t", icmpPingTimeoutSeconds, i.URL)
	} else {
		// Linux and other Unix-like systems
		cmd = exec.Command("ping", "-c", "1", "-W", icmpPingTimeoutSeconds, i.URL)
	}

	if err := cmd.Run(); err != nil {
		return ErrServiceUnavailable
	}
	return nil
}

// ICMPFactory implements the PingerFactory interface.
type ICMPFactory struct{}

// Create returns a pointer to a Pinger.
func (f *ICMPFactory) Create(s Service) (Pinger, error) {
	if s.Type != "icmp" {
		return nil, ErrInvalidCreate
	}
	if s.URL == "" {
		return nil, ErrHostRequired
	}
	if !isValidHostname(s.URL) {
		return nil, ErrInvalidHostname
	}
	return &ICMP{
		Service: Service{URL: s.URL, Name: s.Name},
	}, nil
}

// isValidHostname checks if a string is a valid hostname or IP address
func isValidHostname(host string) bool {
	// Check if it's a valid IP address
	if net.ParseIP(host) != nil {
		return true
	}
	// Check hostname format: alphanumeric, dots, hyphens, must not start/end with hyphen
	if len(host) == 0 || len(host) > 253 {
		return false
	}
	for _, part := range strings.Split(host, ".") {
		if len(part) == 0 || len(part) > 63 {
			return false
		}
		for i, c := range part {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || (c == '-' && i > 0 && i < len(part)-1)) {
				return false
			}
		}
	}
	return true
}
