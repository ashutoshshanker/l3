package ovsdbHandler

import (
	"bgpd"
	"errors"
	"fmt"
	ovsdb "github.com/socketplane/libovsdb"
	"net"
)

const (
	OVSDB_DEFAULT_VRF = "vrf_default"
)

type UUID string

type BGPFlexSwitch struct {
	neighbor bgpd.BGPNeighbor
	global   bgpd.BGPGlobal
}

/*  Get object uuid from the map
 */
func (svr *BGPOvsdbHandler) getObjUUID(val interface{}) UUID {
	retVal, exists := val.([]interface{})
	if !exists {
		return ""
	}
	if len(retVal) != 2 || retVal[0].(string) != "uuid" {
		return ""
	}
	return UUID(retVal[1].(string))
}

/*  Lets get asn number for the local bgp and also get the ovsdb BGP_Router uuid
 */
func (svr *BGPOvsdbHandler) GetBGPRouterInfo() (uint32, UUID, error) {
	var asn uint32
	var id UUID

	vrfs, exists := svr.cache["VRF"]
	if !exists {
		return asn, id, errors.New("vrf table doesn't exists")
	}

	for _, vrf := range vrfs {
		// check vrf name
		if vrf.Fields["name"] == OVSDB_DEFAULT_VRF {
			// get BGP_Routers Map from the vrf fields
			bgpRouters := vrf.Fields["bgp_routers"].(ovsdb.OvsMap).GoMap
			if len(bgpRouters) < 1 {
				return asn, id, errors.New("no bgp router configured")
			} else if len(bgpRouters) > 1 {
				return asn, id,
					errors.New("Multiple bgp routers configured on vrf_default")
			}
			for key, value := range bgpRouters {
				asn = uint32(key.(float64))
				id = svr.getObjUUID(value)
				if id == "" {
					return asn, id, errors.New("invalid uuid")
				}
				return asn, id, nil
			}
		}
	}

	return asn, id, errors.New("no entry found in vrf table")
}

/*  Get bgp neighbor uuids and addrs information
 */
func (svr *BGPOvsdbHandler) GetBGPNeighborInfo(rtUuid UUID) ([]net.IP,
	[]UUID, error) {
	return nil, nil, nil
}

/*  BGP neighbor update in ovsdb... we will update our backend object
 */
func (svr *BGPOvsdbHandler) HandleBGPNeighborUpd(table ovsdb.TableUpdate) error {
	asn, bgpRouterUUID, err := svr.GetBGPRouterInfo()
	if err != nil {
		return err
	}
	fmt.Println("asn:", asn, "BGP_Router UUID:", bgpRouterUUID)
	neighborAddrs, neighborUUIDs, err := svr.GetBGPNeighborInfo(bgpRouterUUID)
	fmt.Println("neighborAddrs:", neighborAddrs, "uuid's:", neighborUUIDs)
	return nil
}

func (svr *BGPOvsdbHandler) HandleBGPRouteUpd() error {
	return nil
}
