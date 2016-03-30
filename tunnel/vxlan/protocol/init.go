// init.go
package vxlan

import ()

func init() {
	// initialize the various db maps
	vtepDB = make(map[vtepDbKey]*vtepDbEntry, 0)
	vxlanDB = make(map[uint32]*vxlanDbEntry, 0)
}
