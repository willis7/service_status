package commands

import "os/exec"

var timeout = "4"

// Commander is a single method interface for running
// cmd line tasks
type Commander interface {
	Command() *exec.Cmd
}

// Ping implements the Commander interface by
// calling the ping command line tool
type Ping struct {
	URL string
}

// Command calls the ping command line arg
func (c *Ping) Command() *exec.Cmd {
	args := []string{"-t", timeout, "-c", "2", c.URL}
	return exec.Command("ping", args...)
}

// NC implements the Commander interface by
// calling the nc command line tool
type NC struct {
	URL  string
	Port string
}

// Command calls the nc command line arg
func (c *NC) Command() *exec.Cmd {
	args := []string{"-z", "-w", timeout, c.URL, c.Port}
	return exec.Command("nc", args...)
}

// Curl implements the Commander interface by
// calling the curl command line tool
type Curl struct {
	URL string
}

// Command calls the curl command line arg
func (c *Curl) Command() *exec.Cmd {
	args := []string{"If", "--max-time", timeout, c.URL}
	return exec.Command("curl", args...)
}
