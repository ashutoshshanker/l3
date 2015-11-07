// path.go
package server

import (
	_ "fmt"
	"l3/bgp/packet"
	_ "net"
)

type Path struct {
	nlri packet.IPPrefix
	pathAttrs []packet.BGPPathAttr
	withdrawn bool
}

func NewPath(nlri packet.IPPrefix, pa []packet.BGPPathAttr, withdrawn bool) *Path {
	path := &Path{
		nlri: nlri,
		pathAttrs: pa,
		withdrawn: withdrawn,
	}

	return path
}

func (path *Path) SetWithdrawn(status bool) {
	path.withdrawn = status
}

func (path *Path) GetWithdrawn() bool {
	return path.withdrawn
}
