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
)

// Pinger is an interface for all modes of testing
// a services availability
type Pinger interface {
	Status() error
}

// Ping performs a ping-like test of a
// services availability
type Ping struct {
	URL string
}

// Status sends a HEAD http request and checks for a valid
// http responce code
func (p *Ping) Status() error {
	resp, err := http.Head(p.URL)
	if err != nil {
		return err
	}

	if !validStatus(resp.StatusCode) {
		return ErrServiceUnavailable
	}

	return nil
}

// Grep checks a response body for a value
type Grep struct {
	URL   string
	Regex string
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
	re := regexp.MustCompile(p.Regex)
	if !re.Match(bodyBytes) {
		return ErrRegexNotFound
	}

	return nil
}

// validStatus checks the input against a list of known-good
// http status codes and returns a bool
func validStatus(code int) bool {
	if code != http.StatusOK {
		return false
	}
	return true
}
