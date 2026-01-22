package status

import (
	"errors"
	"io/ioutil"
	"net/http"
	"regexp"
)

// ErrServiceUnavailable implements error signifying a service is unavailable
var (
	ErrServiceUnavailable = errors.New("commands: service unavailable")
	ErrRegexNotFound      = errors.New("commands: regex not found")
	ErrInvalidCreate      = errors.New("commands: invalid type for create")
)

// Service represents a single endpoint to be tested
type Service struct {
	Type  string `json:"type"`
	URL   string `json:"url"`
	Port  string `json:"port,omitempty"`
	Regex string `json:"regex,omitempty"`
}

// Pinger is an interface which describes how
// to test a service status
type Pinger interface {
	GetService() *Service
	Status() error
}

// PingerFactory is a single method interface which describes
// how to create a Pinger object.
type PingerFactory interface {
	Create(Service) (Pinger, error)
}

// Ping performs a ping-like test of a
// services availability
type Ping struct {
	Service
}

// GetService return the Service pointer
func (p *Ping) GetService() *Service {
	return &p.Service
}

// Status sends a HEAD http request and checks for a valid
// http responce code
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

// PingFactory implements the PingerFactory
// interface
type PingFactory struct{}

// Create returns a pointer to a Pinger
func (factory *PingFactory) Create(s Service) (Pinger, error) {
	if s.Type != "ping" {
		return nil, ErrInvalidCreate
	}
	return &Ping{
		Service: Service{URL: s.URL},
	}, nil
}

// Grep checks a response body for a value
type Grep struct {
	Service
}

// GetService return the Service pointer
func (p *Grep) GetService() *Service {
	return &p.Service
}

// Status requests a page given a URL and checks the response for
// a value matching the regex
func (p *Grep) Status() error {
	// hit the URL and get a response
	resp, err := http.Get(p.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if !validStatus(resp.StatusCode) {
		return ErrServiceUnavailable
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	re := regexp.MustCompile(p.Regex)
	if !re.Match(bodyBytes) {
		return ErrRegexNotFound
	}

	return nil
}

// GrepFactory implements the PingerFactory
// interface
type GrepFactory struct{}

// Create returns a pointer to a Pinger
func (factory *GrepFactory) Create(s Service) (Pinger, error) {
	if s.Type != "grep" {
		return nil, ErrInvalidCreate
	}

	return &Grep{
		Service: Service{URL: s.URL, Regex: s.Regex},
	}, nil
}

// validStatus checks the input against a list of known-good
// http status codes and returns a bool
func validStatus(code int) bool {
	if code != http.StatusOK {
		return false
	}
	return true
}
