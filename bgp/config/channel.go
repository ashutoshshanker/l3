// conn.go
package config

type ReachabilityInfo struct {
	IP          string
	ReachableCh chan bool
}
