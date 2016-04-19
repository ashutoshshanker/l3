package ovsdbHandler

import (
	ovsdb "github.com/socketplane/libovsdb"

	"fmt"
	"reflect"
)

const (
	// OVSDB Server Location
	OVSDB_HANDLER_HOST_IP   = "10.1.10.229"
	OVSDB_HANDLER_HOST_PORT = 6640

	// OVSDB Table
	OVSDB_HANDLER_DB_TABLE = "OpenSwitch" //"todo"

	// OVSDB macro defines
	OVSDB_HANDLER_OPERATIONS_SIZE = 1024
)

type BGPOvsdbNotifier struct {
	updateCh chan *ovsdb.TableUpdates
}

type BGPOvsOperations struct {
	operations []ovsdb.Operation
}

type BGPOvsdbHandler struct {
	bgpovs      *ovsdb.OvsdbClient
	ovsUpdateCh chan *ovsdb.TableUpdates
	cache       map[string]map[string]ovsdb.Row
	operCh      chan *BGPOvsOperations
}

func NewBGPOvsdbNotifier(ch chan *ovsdb.TableUpdates) *BGPOvsdbNotifier {
	return &BGPOvsdbNotifier{
		updateCh: ch,
	}
}

func NewBGPOvsdbHandler() (*BGPOvsdbHandler, error) {
	ovs, err := ovsdb.Connect(OVSDB_HANDLER_HOST_IP, OVSDB_HANDLER_HOST_PORT)
	if err != nil {
		return nil, err
	}
	ovsUpdateCh := make(chan *ovsdb.TableUpdates)
	n := NewBGPOvsdbNotifier(ovsUpdateCh)
	ovs.Register(n)

	return &BGPOvsdbHandler{
		bgpovs:      ovs,
		ovsUpdateCh: ovsUpdateCh,
		operCh:      make(chan *BGPOvsOperations, OVSDB_HANDLER_OPERATIONS_SIZE),
		cache:       make(map[string]map[string]ovsdb.Row),
	}, nil
}

/*  BGP OVS DB populate cache with the latest update information from the
 *  notification channel
 */
func (svr *BGPOvsdbHandler) BGPPopulateOvsdbCache(updates ovsdb.TableUpdates) {
	for table, tableUpdate := range updates.Updates {
		if _, ok := svr.cache[table]; !ok {
			svr.cache[table] = make(map[string]ovsdb.Row)
		}

		for uuid, row := range tableUpdate.Rows {
			empty := ovsdb.Row{}
			if !reflect.DeepEqual(row.New, empty) {
				svr.cache[table][uuid] = row.New
			} else {
				delete(svr.cache[table], uuid)
			}
		}
	}
}

/* Stub interfaces for ovsdb library notifier
 */
func (svr BGPOvsdbNotifier) Update(context interface{}, tableUpdates ovsdb.TableUpdates) {
	svr.updateCh <- &tableUpdates
}

/* Stub interfaces for ovsdb library notifier
 */
func (svr BGPOvsdbNotifier) Locked([]interface{}) {
}

/* Stub interfaces for ovsdb library notifier
 */
func (svr BGPOvsdbNotifier) Stolen([]interface{}) {
}

/* Stub interfaces for ovsdb library notifier
 */
func (svr BGPOvsdbNotifier) Echo([]interface{}) {
}

/* Stub interfaces for ovsdb library notifier
 */
func (svr BGPOvsdbNotifier) Disconnected(client *ovsdb.OvsdbClient) {
}

/*  BGP OVS DB transaction api handler
 */
func (svr *BGPOvsdbHandler) BGPOvsdbTransact(operations []ovsdb.Operation) error {
	return nil
}

/*  BGP OVS DB handle update information
 */
func (svr *BGPOvsdbHandler) BGPOvsdbUpdateInfo(updates ovsdb.TableUpdates) {
	table, ok := updates.Updates["BGP_Router"]
	if ok {
		fmt.Println(table)
	}
	table, ok = updates.Updates["BGP_Neighbor"]
	if ok {
		fmt.Println(table)
	}
	table, ok = updates.Updates["BGP_Route"]
	if ok {
		fmt.Println(table)
	}
}

/*
 *  BGP OVS DB server.
 *	This API will handle reading operations from table... It can also do
 *	transactions.... In short its read/write bgp ovsdb handler
 */
func (svr *BGPOvsdbHandler) BGPOvsdbServe() error {
	initial, err := svr.bgpovs.MonitorAll(OVSDB_HANDLER_DB_TABLE, "")
	if err != nil {
		return err
	}

	go func() {
		svr.ovsUpdateCh <- initial
	}()

	for {
		select {
		case updates := <-svr.ovsUpdateCh:
			svr.BGPPopulateOvsdbCache(*updates)
			svr.BGPOvsdbUpdateInfo(*updates)
		case oper := <-svr.operCh:
			if err := svr.BGPOvsdbTransact(oper.operations); err != nil {
				//@FIXME: add some error message if needed
			}
		}
	}
	return nil
}
