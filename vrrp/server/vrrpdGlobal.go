package vrrp

import (
	"log/syslog"
)

type VrrpServiceHandler struct {
}

var (
	logger *syslog.Writer
)
