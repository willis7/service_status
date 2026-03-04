package main

import (
	"github.com/willis7/service_status/cmd"
	"github.com/willis7/service_status/status"
)

func init() {
	status.LoadTemplate()
}

func main() {
	cmd.Execute()
}
