// conn.go
package config

import (
	"fmt"
)

type IPError struct {
	Address string
}

func (i IPError) Error() string {
	return fmt.Sprintf("Can't convert %s to net.IP. Not a valid IP address.")
}

type AddressError struct {
	Message string
}

func (a AddressError) Error() string {
	return fmt.Sprintf("%s", a.Message)
}

type AddressNotResolvedError struct {
	Message string
}

func (a AddressNotResolvedError) Error() string {
	return fmt.Sprintf("%s", a.Message)
}
