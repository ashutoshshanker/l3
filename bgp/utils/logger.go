// conn.go
package utils

import (
	"log/syslog"
)

var Logger *syslog.Writer

func SetLogger(logger *syslog.Writer) {
	Logger = logger
}
