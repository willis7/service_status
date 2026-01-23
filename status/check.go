package status

import (
	"context"
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

// Sentinel errors for service status checks.
//
// ErrServiceDegraded indicates a service is operational but performing below
// normal parameters. Custom Pinger implementations can return this error to
// signal degraded performance (e.g., slow response times, partial functionality).
// Use IsDegraded(err) to check for this condition.
var (
	ErrServiceUnavailable = errors.New("commands: service unavailable")
	ErrServiceDegraded    = errors.New("commands: service degraded")
	ErrRegexNotFound      = errors.New("commands: regex not found")
	ErrInvalidCreate      = errors.New("commands: invalid type for create")
	ErrHostRequired       = errors.New("commands: host is required for icmp check")
	ErrInvalidHostname    = errors.New("commands: invalid hostname for icmp check")
	ErrCommandRequired    = errors.New("commands: command is required for script check")
)

// IsDegraded returns true if the error represents a degraded service state.
func IsDegraded(err error) bool {
	return errors.Is(err, ErrServiceDegraded)
}

// IsOperational returns true if the error indicates the service is operational
// (either fully up or degraded but still functioning).
func IsOperational(err error) bool {
	return err == nil || IsDegraded(err)
}

// icmpPingTimeoutSeconds is the timeout in seconds for ICMP ping attempts (string for CLI argument).
const icmpPingTimeoutSeconds = "5"

// tcpDialTimeout is the timeout duration for TCP connection attempts.
const tcpDialTimeout = 10 * time.Second

// Service represents a single endpoint to be tested.
type Service struct {
	Type    string `json:"type"`
	URL     string `json:"url"`
	Port    string `json:"port,omitempty"`
	Regex   string `json:"regex,omitempty"`
	Name    string `json:"name,omitempty"`
	Command string `json:"command,omitempty"`
}

// DisplayName returns the Name if set, otherwise falls back to the URL.
// For script services without a URL, it returns the command executable name.
func (s *Service) DisplayName() string {
	if s.Name != "" {
		return s.Name
	}
	if s.URL != "" {
		return s.URL
	}
	if s.Command != "" {
		args := strings.Fields(s.Command)
		if len(args) > 0 {
			return args[0]
		}
	}
	return "unknown"
}

// StatusResult contains the result of a status check, including timing information.
type StatusResult struct {
	// Err is the error from the status check (nil if successful)
	Err error
	// ResponseTime is the duration of the check
	ResponseTime time.Duration
}

// Pinger is an interface which describes how
// to test a service status.
type Pinger interface {
	GetService() *Service
	Status() error
	// StatusWithTiming performs a status check and returns the result with timing information.
	// ResponseTime measures the total wall-clock duration of the Status() call,
	// which may include network latency, processing time, and timeouts.
	StatusWithTiming() StatusResult
}

// statusWithTiming wraps a status check function and returns the result with timing.
func statusWithTiming(statusFn func() error) StatusResult {
	start := time.Now()
	err := statusFn()
	return StatusResult{
		Err:          err,
		ResponseTime: time.Since(start),
	}
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

// StatusWithTiming performs a status check and returns the result with timing information.
func (p *Ping) StatusWithTiming() StatusResult {
	return statusWithTiming(p.Status)
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

// StatusWithTiming performs a status check and returns the result with timing information.
func (g *Grep) StatusWithTiming() StatusResult {
	return statusWithTiming(g.Status)
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

// StatusWithTiming performs a status check and returns the result with timing information.
func (t *TCP) StatusWithTiming() StatusResult {
	return statusWithTiming(t.Status)
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

// StatusWithTiming performs a status check and returns the result with timing information.
func (i *ICMP) StatusWithTiming() StatusResult {
	return statusWithTiming(i.Status)
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

// isValidHostname checks if a string is a valid hostname or IP address.
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

// scriptExitCodeDegraded is the exit code that indicates a degraded service status.
const scriptExitCodeDegraded = 80

// scriptTimeout is the timeout duration for script execution.
const scriptTimeout = 30 * time.Second

// Script executes an external command or script and interprets the exit code.
// Exit code 0 = OK, 80 = Degraded, any other = Failure.
//
// Security note: Commands are read from the config file, which should be
// protected with appropriate file permissions. No additional command
// validation is performed beyond basic parsing.
type Script struct {
	Service
}

// GetService returns the Service pointer.
func (s *Script) GetService() *Service {
	return &s.Service
}

// Status executes the configured command and interprets the exit code.
// Returns nil for exit code 0, ErrServiceDegraded for exit code 80,
// and ErrServiceUnavailable for any other exit code.
func (s *Script) Status() error {
	args := parseCommand(s.Command)
	if len(args) == 0 {
		return ErrCommandRequired
	}

	ctx, cancel := context.WithTimeout(context.Background(), scriptTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return ErrServiceUnavailable
		}
		// Check if it's an exit error to get the exit code
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == scriptExitCodeDegraded {
				return ErrServiceDegraded
			}
		}
		return ErrServiceUnavailable
	}
	return nil
}

// StatusWithTiming performs a status check and returns the result with timing information.
func (s *Script) StatusWithTiming() StatusResult {
	return statusWithTiming(s.Status)
}

// parseCommand splits a command string into executable and arguments.
// It handles quoted strings to allow spaces in arguments.
func parseCommand(cmd string) []string {
	var args []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for _, c := range cmd {
		switch {
		case c == '"' || c == '\'':
			if !inQuote {
				inQuote = true
				quoteChar = c
			} else if c == quoteChar {
				inQuote = false
				quoteChar = 0
			} else {
				current.WriteRune(c)
			}
		case c == ' ' && !inQuote:
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(c)
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}
	return args
}

// ScriptFactory implements the PingerFactory interface.
type ScriptFactory struct{}

// Create returns a pointer to a Pinger.
func (f *ScriptFactory) Create(s Service) (Pinger, error) {
	if s.Type != "script" {
		return nil, ErrInvalidCreate
	}
	if s.Command == "" {
		return nil, ErrCommandRequired
	}
	return &Script{
		Service: Service{Command: s.Command, Name: s.Name},
	}, nil
}
