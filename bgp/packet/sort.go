// bgp.go
package packet

type PathAttrs []BGPPathAttr

func (pa PathAttrs) Len() int {
	return len(pa)
}

func (pa PathAttrs) Swap(i, j int) {
	pa[i], pa[j] = pa[j], pa[i]
}

func (pa PathAttrs) Less(i, j int) bool {
	return pa[i].GetCode() < pa[j].GetCode()
}
