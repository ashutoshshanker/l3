package ovsdbHandler

import (
	"bgpd"
	"errors"
	"fmt"
	_ "github.com/nu7hatch/gouuid"
	ovsdb "github.com/socketplane/libovsdb"
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
func (svr *BGPOvsdbHandler) getObjUUID(val interface{}) string {
	retVal, exists := val.([]interface{})
	if !exists {
		return ""
	}
	if len(retVal) != 2 || retVal[0].(string) != "uuid" {
		return ""
	}
	return retVal[1].(string)
}

/*  Lets get asn number for the local bgp and also get the ovsdb BGP_Router uuid
 */
func (svr *BGPOvsdbHandler) GetBGPRouterUUID() (uint32, string, error) {
	var asn uint32
	var id string

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

/*  BGP neighbor update in ovsdb... we will update our backend object
 */
func (svr *BGPOvsdbHandler) HandleBGPNeighborUpd(table ovsdb.TableUpdate) error {
	asn, bgpRouterUUID, err := svr.GetBGPRouterUUID()
	if err != nil {
		return err
	}
	fmt.Println("asn:", asn, "BGP_Router UUID:", bgpRouterUUID)
	return nil
}

func (svr *BGPOvsdbHandler) HandleBGPRouteUpd() error {
	return nil
}
