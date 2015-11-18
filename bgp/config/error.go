// conn.go
package config

import (
	"fmt"
)

type AddressError struct {
	Message string
}

func (a AddressError) Error() string {
	return fmt.Sprintf("%s", a.Message)
}
