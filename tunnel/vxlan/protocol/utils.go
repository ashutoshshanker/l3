// utils.go
package vxlan

import ()

func CompareVNI(vni uint32, netvni [3]byte) int {
	v := [3]byte{byte(vni >> 16 & 0xff),
		byte(vni >> 8 & 0xff),
		byte(vni >> 0 & 0xff),
	}
	if v == netvni {
		return 0
	} else if v[0] > netvni[0] ||
		v[1] > netvni[1] ||
		v[2] > netvni[1] {
		return 1
	}
	return -1
}
