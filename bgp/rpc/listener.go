// server.go
package rpc

import (
	"bgpd"
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"l3/bgp/config"
	"l3/bgp/server"
	"log/syslog"
	"net"
)

const DBName string = "UsrConfDb.db"

type PeerConfigCommands struct {
	IP      net.IP
	Command int
}

type BGPHandler struct {
	PeerCommandCh chan PeerConfigCommands
	server        *server.BGPServer
	logger        *syslog.Writer
}

func NewBGPHandler(server *server.BGPServer, logger *syslog.Writer, filePath string) *BGPHandler {
	h := new(BGPHandler)
	h.PeerCommandCh = make(chan PeerConfigCommands)
	h.server = server
	h.logger = logger
	h.readConfigFromDB(filePath)
	return h
}

func (h *BGPHandler) handleGlobalConfig(dbHdl *sql.DB) error {
	dbCmd := "select * from BGPGlobalConfig"
	rows, err := dbHdl.Query(dbCmd)
	if err != nil {
		h.logger.Err(fmt.Sprintf("DB method Query failed for %s with error %s", dbCmd, err))
		return err
	}

	defer rows.Close()

	var gConf config.GlobalConfig
	var routerIP string
	for rows.Next() {
		if err = rows.Scan(&gConf.AS, &routerIP); err != nil {
			h.logger.Err(fmt.Sprintf("DB method Scan failed when iterating over BGPGlobalConfig rows with error %s", err))
			return err
		}

		gConf.RouterId = h.convertStrIPToNetIP(routerIP)
		if gConf.RouterId == nil {
			h.logger.Err(fmt.Sprintln("handleGlobalConfig - IP is not valid:", routerIP))
			return config.IPError{routerIP}
		}

		h.server.GlobalConfigCh <- gConf
	}

	return nil
}

func (h *BGPHandler) handleNeighborConfig(dbHdl *sql.DB) error {
	dbCmd := "select * from BGPNeighborConfig"
	rows, err := dbHdl.Query(dbCmd)
	if err != nil {
		h.logger.Err(fmt.Sprintf("DB method Query failed for '%s' with error %s", dbCmd, err))
		return err
	}

	defer rows.Close()

	var nConf config.NeighborConfig
	var neighborIP string
	for rows.Next() {
		if err = rows.Scan(&nConf.PeerAS, &nConf.LocalAS, &nConf.AuthPassword, &nConf.Description, &neighborIP,
			&nConf.RouteReflectorClusterId, &nConf.RouteReflectorClient); err != nil {
			h.logger.Err(fmt.Sprintf("DB method Scan failed when iterating over BGPNeighborConfig rows with error %s", err))
			return err
		}

		nConf.NeighborAddress = net.ParseIP(neighborIP)
		if nConf.NeighborAddress == nil {
			h.logger.Info(fmt.Sprintf("Can't create BGP neighbor - IP[%s] not valid", neighborIP))
			return config.IPError{neighborIP}
		}

		h.server.AddPeerCh <- nConf
	}

	return nil
}

func (h *BGPHandler) readConfigFromDB(filePath string) error {
	var dbPath string = filePath + DBName

	dbHdl, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		h.logger.Err(fmt.Sprintf("Failed to open the DB at %s with error %s", dbPath, err))
		return err
	}

	defer dbHdl.Close()

	if err = h.handleGlobalConfig(dbHdl); err != nil {
		return err
	}

	if err = h.handleNeighborConfig(dbHdl); err != nil {
		return err
	}

	return nil
}

func (h *BGPHandler) convertStrIPToNetIP(ip string) net.IP {
	if ip == "localhost" {
		ip = "127.0.0.1"
	}

	netIP := net.ParseIP(ip)
	return netIP
}

func (h *BGPHandler) CreateBGPGlobal(bgpGlobal *bgpd.BGPGlobal) (bool, error) {
	h.logger.Info(fmt.Sprintln("Create global config attrs:", bgpGlobal))

	ip := h.convertStrIPToNetIP(bgpGlobal.RouterId)
	if ip == nil {
		h.logger.Info(fmt.Sprintln("CreateBGPGlobal - IP is not valid:", bgpGlobal.RouterId))
		return false, nil
	}

	gConf := config.GlobalConfig{
		AS:       uint32(bgpGlobal.AS),
		RouterId: ip,
	}
	h.server.GlobalConfigCh <- gConf
	return true, nil
}

func (h *BGPHandler) GetBGPGlobal() (*bgpd.BGPGlobalState, error) {
	bgpGlobal := h.server.GetBGPGlobalState()
	bgpGlobalResponse := bgpd.NewBGPGlobalState()
	bgpGlobalResponse.AS = int32(bgpGlobal.AS)
	bgpGlobalResponse.RouterId = bgpGlobal.RouterId.String()
	bgpGlobalResponse.TotalPaths = int32(bgpGlobal.TotalPaths)
	bgpGlobalResponse.TotalPrefixes = int32(bgpGlobal.TotalPrefixes)
	return bgpGlobalResponse, nil
}

func (h *BGPHandler) UpdateBGPGlobal(bgpGlobal *bgpd.BGPGlobal) (bool, error) {
	h.logger.Info(fmt.Sprintln("Update global config attrs:", bgpGlobal))
	return true, nil
}

func (h *BGPHandler) DeleteBGPGlobal(bgpGlobal *bgpd.BGPGlobal) (bool, error) {
	h.logger.Info(fmt.Sprintln("Delete global config attrs:", bgpGlobal))
	return true, nil
}

func (h *BGPHandler) CreateBGPNeighbor(bgpNeighbor *bgpd.BGPNeighbor) (bool, error) {
	h.logger.Info(fmt.Sprintln("Create BGP neighbor attrs:", bgpNeighbor))
	ip := net.ParseIP(bgpNeighbor.NeighborAddress)
	if ip == nil {
		h.logger.Info(fmt.Sprintf("Can't create BGP neighbor - IP[%s] not valid", bgpNeighbor.NeighborAddress))
	}
	pConf := config.NeighborConfig{
		PeerAS:          uint32(bgpNeighbor.PeerAS),
		LocalAS:         uint32(bgpNeighbor.LocalAS),
		AuthPassword:    bgpNeighbor.AuthPassword,
		Description:     bgpNeighbor.Description,
		NeighborAddress: ip,
		RouteReflectorClusterId: uint32(bgpNeighbor.RouteReflectorClusterId),
		RouteReflectorClient: bgpNeighbor.RouteReflectorClient,
	}
	h.server.AddPeerCh <- pConf
	return true, nil
}

func (h *BGPHandler) convertToThriftNeighbor(neighborState *config.NeighborState) *bgpd.BGPNeighborState {
	bgpNeighborResponse := bgpd.NewBGPNeighborState()
	bgpNeighborResponse.PeerAS = int32(neighborState.PeerAS)
	bgpNeighborResponse.LocalAS = int32(neighborState.LocalAS)
	bgpNeighborResponse.AuthPassword = neighborState.AuthPassword
	bgpNeighborResponse.PeerType = bgpd.PeerType(neighborState.PeerType)
	bgpNeighborResponse.Description = neighborState.Description
	bgpNeighborResponse.NeighborAddress = neighborState.NeighborAddress.String()
	bgpNeighborResponse.SessionState = int32(neighborState.SessionState)

	received := bgpd.NewBgpCounters()
	received.Notification = int64(neighborState.Messages.Received.Notification)
	received.Update = int64(neighborState.Messages.Received.Update)
	sent := bgpd.NewBgpCounters()
	sent.Notification = int64(neighborState.Messages.Sent.Notification)
	sent.Update = int64(neighborState.Messages.Sent.Update)
	messages := bgpd.NewBGPMessages()
	messages.Received = received
	messages.Sent = sent
	bgpNeighborResponse.Messages = messages

	queues := bgpd.NewBGPQueues()
	queues.Input = int32(neighborState.Queues.Input)
	queues.Output = int32(neighborState.Queues.Output)
	bgpNeighborResponse.Queues = queues

	return bgpNeighborResponse
}

func (h *BGPHandler) GetBGPNeighbor(neighborAddress string) (*bgpd.BGPNeighborState, error) {
	bgpNeighborState := h.server.GetBGPNeighborState(neighborAddress)
	bgpNeighborResponse := h.convertToThriftNeighbor(bgpNeighborState)
	return bgpNeighborResponse, nil
}

func (h *BGPHandler) BulkGetBGPNeighbors(index int64, count int64) (*bgpd.BGPNeighborStateBulk, error) {
	nextIdx, currCount, bgpNeighbors := h.server.BulkGetBGPNeighbors(int(index), int(count))
	bgpNeighborsResponse := make([]*bgpd.BGPNeighborState, len(bgpNeighbors))
	for idx, item := range bgpNeighbors {
		bgpNeighborsResponse[idx] = h.convertToThriftNeighbor(item)
	}

	bgpNeighborStateBulk := bgpd.NewBGPNeighborStateBulk()
	bgpNeighborStateBulk.NextIndex = int64(nextIdx)
	bgpNeighborStateBulk.Count = int64(currCount)
	bgpNeighborStateBulk.More = (nextIdx != 0)
	bgpNeighborStateBulk.StateList = bgpNeighborsResponse

	return bgpNeighborStateBulk, nil
}

func (h *BGPHandler) UpdateBGPNeighbor(bgpNeighbor *bgpd.BGPNeighbor) (bool, error) {
	h.logger.Info(fmt.Sprintln("Update peer attrs:", bgpNeighbor))
	return true, nil
}

func (h *BGPHandler) DeleteBGPNeighbor(neighborAddress string) (bool, error) {
	h.logger.Info(fmt.Sprintln("Delete BGP neighbor:", neighborAddress))
	ip := net.ParseIP(neighborAddress)
	if ip == nil {
		h.logger.Info(fmt.Sprintf("Can't delete BGP neighbor - IP[%s] not valid", neighborAddress))
		return false, nil
	}
	h.server.RemPeerCh <- neighborAddress
	return true, nil
}

func (h *BGPHandler) PeerCommand(in *PeerConfigCommands, out *bool) error {
	h.PeerCommandCh <- *in
	h.logger.Info(fmt.Sprintln("Good peer command:", in))
	*out = true
	return nil
}
