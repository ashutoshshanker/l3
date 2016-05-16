// conn.go
package config

import ()

type ConnDir int

const (
	ConnDirOut ConnDir = iota
	ConnDirIn
	ConnDirMax
	ConnDirInvalid = ConnDirMax
)
