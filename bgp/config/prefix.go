// conn.go
package config

import (
	"net"
)

type IPPrefix struct {
	Length uint8
	Prefix net.IP
}
