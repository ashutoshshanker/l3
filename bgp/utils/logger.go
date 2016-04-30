// conn.go
package utils

import (
	"utils/logging"
)

//var Logger *logging.Writer
var Logger *logging.LogFile

//func SetLogger(logger *logging.Writer) {
func SetLogger(logger *logging.LogFile) {
	Logger = logger
}
