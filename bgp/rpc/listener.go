// server.go
package rpc

import (
	"bgpd"
	"errors"
	"fmt"
	"l3/bgp/config"
	bgppolicy "l3/bgp/policy"
	"l3/bgp/server"
	"models"
	"net"
	"strings"
	"utils/dbutils"
	"utils/logging"
	utilspolicy "utils/policy"
)

const DBName string = "UsrConfDb.db"

type PeerConfigCommands struct {
	IP      net.IP
	Command int
}

type BGPHandler struct {
	PeerCommandCh chan PeerConfigCommands
	server        *server.BGPServer
	bgpPE         *bgppolicy.BGPPolicyEngine
	logger        *logging.Writer
	dbUtil        *dbutils.DBUtil
}

func NewBGPHandler(server *server.BGPServer, policy *bgppolicy.BGPPolicyEngine, logger *logging.Writer,
	dbUtil *dbutils.DBUtil, filePath string) *BGPHandler {
	h := new(BGPHandler)
	h.PeerCommandCh = make(chan PeerConfigCommands)
	h.server = server
	h.bgpPE = policy
	h.logger = logger
	h.dbUtil = dbUtil
	h.readConfigFromDB(filePath)
	return h
}

func (h *BGPHandler) convertModelToBGPGlobalConfig(obj models.BGPGlobal) (config.GlobalConfig, error) {
	var err error
	gConf := config.GlobalConfig{
		AS:                  obj.ASNum,
		RouterId:            h.convertStrIPToNetIP(obj.RouterId),
		UseMultiplePaths:    obj.UseMultiplePaths,
		EBGPMaxPaths:        obj.EBGPMaxPaths,
		EBGPAllowMultipleAS: obj.EBGPAllowMultipleAS,
		IBGPMaxPaths:        obj.IBGPMaxPaths,
	}

	if gConf.RouterId == nil {
		h.logger.Err(fmt.Sprintln("convertModelToBGPGlobalConfig - IP is not valid:", obj.RouterId))
		err = config.IPError{obj.RouterId}
	}

	return gConf, err
}

func (h *BGPHandler) handleGlobalConfig() error {
	var obj models.BGPGlobal
	objList, err := h.dbUtil.GetAllObjFromDb(obj)
	if err != nil {
		h.logger.Err(fmt.Sprintf("GetAllObjFromDb failed for BGPGlobal with error %s", err))
		return err
	}

	for _, confObj := range objList {
		obj = confObj.(models.BGPGlobal)

		gConf, err := h.convertModelToBGPGlobalConfig(obj)
		if err == nil {
			h.logger.Err(fmt.Sprintln("handleGlobalConfig - Failed to convert Model object BGP Global, error:", err))
			return err
		}

		h.server.GlobalConfigCh <- gConf
	}
	return nil
}

func (h *BGPHandler) convertModelToBGPPeerGroup(obj models.BGPPeerGroup) (group config.PeerGroupConfig, err error) {
	group = config.PeerGroupConfig{
		BaseConfig: config.BaseConfig{
			PeerAS:                  uint32(obj.PeerAS),
			LocalAS:                 uint32(obj.LocalAS),
			AuthPassword:            obj.AuthPassword,
			Description:             obj.Description,
			RouteReflectorClusterId: uint32(obj.RouteReflectorClusterId),
			RouteReflectorClient:    obj.RouteReflectorClient,
			MultiHopEnable:          obj.MultiHopEnable,
			MultiHopTTL:             uint8(obj.MultiHopTTL),
			ConnectRetryTime:        uint32(obj.ConnectRetryTime),
			HoldTime:                uint32(obj.HoldTime),
			KeepaliveTime:           uint32(obj.KeepaliveTime),
			AddPathsRx:              obj.AddPathsRx,
			AddPathsMaxTx:           uint8(obj.AddPathsMaxTx),
			MaxPrefixes:             uint32(obj.MaxPrefixes),
			MaxPrefixesThresholdPct: uint8(obj.MaxPrefixesThresholdPct),
			MaxPrefixesDisconnect:   obj.MaxPrefixesDisconnect,
			MaxPrefixesRestartTimer: uint8(obj.MaxPrefixesRestartTimer),
		},
		Name: obj.Name,
	}
	return group, err
}

func (h *BGPHandler) handlePeerGroup() error {
	var obj models.BGPPeerGroup
	objList, err := h.dbUtil.GetAllObjFromDb(obj)
	if err != nil {
		h.logger.Err(fmt.Sprintf("GetAllObjFromDb for BGPPeerGroup failed with error %s", err))
		return err
	}

	for _, confObj := range objList {
		obj = confObj.(models.BGPPeerGroup)

		group, err := h.convertModelToBGPPeerGroup(obj)
		if err != nil {
			h.logger.Err(fmt.Sprintln("handlePeerGroup - Failed to convert Model object to BGP Peer group, error:",
				err))
			return err
		}

		h.server.AddPeerGroupCh <- server.PeerGroupUpdate{config.PeerGroupConfig{}, group, make([]bool, 0)}
	}

	return nil
}

func (h *BGPHandler) convertModelToBGPNeighbor(obj models.BGPNeighbor) (neighbor config.NeighborConfig, err error) {
	var ip net.IP
	var ifIndex int32
	ip, ifIndex, err = h.getIPAndIfIndexForNeighbor(obj.NeighborAddress, obj.IfIndex)
	if err != nil {
		h.logger.Info(fmt.Sprintln("convertModelToBGPNeighbor: getIPAndIfIndexForNeighbor",
			"failed for neighbor address", obj.NeighborAddress, "and ifIndex", obj.IfIndex))
		return neighbor, err
	}

	neighbor = config.NeighborConfig{
		BaseConfig: config.BaseConfig{
			PeerAS:                  uint32(obj.PeerAS),
			LocalAS:                 uint32(obj.LocalAS),
			AuthPassword:            obj.AuthPassword,
			Description:             obj.Description,
			RouteReflectorClusterId: uint32(obj.RouteReflectorClusterId),
			RouteReflectorClient:    obj.RouteReflectorClient,
			MultiHopEnable:          obj.MultiHopEnable,
			MultiHopTTL:             uint8(obj.MultiHopTTL),
			ConnectRetryTime:        uint32(obj.ConnectRetryTime),
			HoldTime:                uint32(obj.HoldTime),
			KeepaliveTime:           uint32(obj.KeepaliveTime),
			BfdEnable:               obj.BfdEnable,
			AddPathsRx:              obj.AddPathsRx,
			AddPathsMaxTx:           uint8(obj.AddPathsMaxTx),
			MaxPrefixes:             uint32(obj.MaxPrefixes),
			MaxPrefixesThresholdPct: uint8(obj.MaxPrefixesThresholdPct),
			MaxPrefixesDisconnect:   obj.MaxPrefixesDisconnect,
			MaxPrefixesRestartTimer: uint8(obj.MaxPrefixesRestartTimer),
		},
		NeighborAddress: ip,
		IfIndex:         ifIndex,
		PeerGroup:       obj.PeerGroup,
	}
	return neighbor, err
}

func (h *BGPHandler) handleNeighborConfig() error {
	var obj models.BGPNeighbor
	objList, err := h.dbUtil.GetAllObjFromDb(obj)
	if err != nil {
		h.logger.Err(fmt.Sprintf("GetAllObjFromDb for BGPNeighbor failed with error %s", err))
		return err
	}

	for _, confObj := range objList {
		obj = confObj.(models.BGPNeighbor)

		neighbor, err := h.convertModelToBGPNeighbor(obj)
		if err != nil {
			h.logger.Err(fmt.Sprintln("handleNeighborConfig - Failed to convert Model object to BGP neighbor, error:",
				err))
			return err
		}

		h.server.AddPeerCh <- server.PeerUpdate{config.NeighborConfig{}, neighbor, make([]bool, 0)}
	}

	return nil
}

func convertModelToPolicyConditionConfig(
	cfg models.BGPPolicyCondition) *utilspolicy.PolicyConditionConfig {
	destIPMatch := utilspolicy.PolicyDstIpMatchPrefixSetCondition{
		Prefix: utilspolicy.PolicyPrefix{
			IpPrefix:        cfg.IpPrefix,
			MasklengthRange: cfg.MaskLengthRange,
		},
	}
	return &utilspolicy.PolicyConditionConfig{
		Name:                          cfg.Name,
		ConditionType:                 cfg.ConditionType,
		MatchDstIpPrefixConditionInfo: destIPMatch,
	}
}

func (h *BGPHandler) handlePolicyConditions() error {
	h.logger.Info(fmt.Sprintln("handlePolicyConditions"))
	var conditionObj models.BGPPolicyCondition
	conditionList, err := h.dbUtil.GetAllObjFromDb(conditionObj)
	if err != nil {
		h.logger.Err(fmt.Sprintln("handlePolicyConditions - Failed to create policy",
			"condition config on restart with error", err))
		return err
	}

	for idx := 0; idx < len(conditionList); idx++ {
		policyCondCfg :=
			convertModelToPolicyConditionConfig(conditionList[idx].(models.BGPPolicyCondition))
		h.logger.Info(fmt.Sprintln("handlePolicyConditions - create policy condition",
			policyCondCfg.Name))
		h.bgpPE.ConditionCfgCh <- *policyCondCfg
	}
	return nil
}

func convertModelToPolicyActionConfig(cfg models.BGPPolicyAction) *utilspolicy.PolicyActionConfig {
	return &utilspolicy.PolicyActionConfig{
		Name:            cfg.Name,
		ActionType:      cfg.ActionType,
		GenerateASSet:   cfg.GenerateASSet,
		SendSummaryOnly: cfg.SendSummaryOnly,
	}
}

func (h *BGPHandler) handlePolicyActions() error {
	h.logger.Info(fmt.Sprintln("handlePolicyActions"))
	var actionObj models.BGPPolicyAction
	actionList, err := h.dbUtil.GetAllObjFromDb(actionObj)
	if err != nil {
		h.logger.Err(fmt.Sprintln("handlePolicyActions - Failed to create policy action",
			"config on restart with error", err))
		return err
	}

	for idx := 0; idx < len(actionList); idx++ {
		policyActionCfg :=
			convertModelToPolicyActionConfig(actionList[idx].(models.BGPPolicyAction))
		h.logger.Info(fmt.Sprintln("handlePolicyActions - create policy action",
			policyActionCfg.Name))
		h.bgpPE.ActionCfgCh <- *policyActionCfg
	}
	return nil
}

func convertModelToPolicyStmtConfig(cfg models.BGPPolicyStmt) *utilspolicy.PolicyStmtConfig {
	return &utilspolicy.PolicyStmtConfig{
		Name:            cfg.Name,
		MatchConditions: cfg.MatchConditions,
		Conditions:      cfg.Conditions,
		Actions:         cfg.Actions,
	}
}

func (h *BGPHandler) handlePolicyStmts() error {
	h.logger.Info(fmt.Sprintln("handlePolicyStmts"))
	var stmtObj models.BGPPolicyStmt
	stmtList, err := h.dbUtil.GetAllObjFromDb(stmtObj)
	if err != nil {
		h.logger.Err(fmt.Sprintln("handlePolicyStmts - Failed to create policy statement",
			"config on restart with error", err))
		return err
	}

	for idx := 0; idx < len(stmtList); idx++ {
		policyStmtCfg := convertModelToPolicyStmtConfig(stmtList[idx].(models.BGPPolicyStmt))
		h.logger.Info(fmt.Sprintln("handlePolicyStmts - create policy statement",
			policyStmtCfg.Name))
		h.bgpPE.StmtCfgCh <- *policyStmtCfg
	}
	return nil
}

func convertModelToPolicyDefinitionConfig(
	cfg models.BGPPolicyDefinition) *utilspolicy.PolicyDefinitionConfig {
	stmtPrecedenceList := make([]utilspolicy.PolicyDefinitionStmtPrecedence, 0)
	for i := 0; i < len(cfg.StatementList); i++ {
		stmtPrecedence := utilspolicy.PolicyDefinitionStmtPrecedence{
			Precedence: int(cfg.StatementList[i].Precedence),
			Statement:  cfg.StatementList[i].Statement,
		}
		stmtPrecedenceList = append(stmtPrecedenceList, stmtPrecedence)
	}

	return &utilspolicy.PolicyDefinitionConfig{
		Name:                       cfg.Name,
		Precedence:                 int(cfg.Precedence),
		MatchType:                  cfg.MatchType,
		PolicyDefinitionStatements: stmtPrecedenceList,
	}
}

func (h *BGPHandler) handlePolicyDefinitions() error {
	h.logger.Info(fmt.Sprintln("handlePolicyDefinitions"))
	var defObj models.BGPPolicyDefinition
	definitionList, err := h.dbUtil.GetAllObjFromDb(defObj)
	if err != nil {
		h.logger.Err(fmt.Sprintln("handlePolicyDefinitions - Failed to create policy",
			"definition config on restart with error", err))
		return err
	}

	for idx := 0; idx < len(definitionList); idx++ {
		policyDefCfg := convertModelToPolicyDefinitionConfig(
			definitionList[idx].(models.BGPPolicyDefinition))
		h.logger.Info(fmt.Sprintln("handlePolicyDefinitions - create policy definition",
			policyDefCfg.Name))
		h.bgpPE.DefinitionCfgCh <- *policyDefCfg
	}
	return nil
}

func (h *BGPHandler) readConfigFromDB(filePath string) error {
	var err error

	if err = h.handlePolicyConditions(); err != nil {
		return err
	}

	if err = h.handlePolicyActions(); err != nil {
		return err
	}

	if err = h.handlePolicyStmts(); err != nil {
		return err
	}

	if err = h.handlePolicyDefinitions(); err != nil {
		return err
	}

	if err = h.handleGlobalConfig(); err != nil {
		return err
	}

	if err = h.handlePeerGroup(); err != nil {
		return err
	}

	if err = h.handleNeighborConfig(); err != nil {
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

func (h *BGPHandler) SendBGPGlobal(bgpGlobal *bgpd.BGPGlobal) (bool, error) {
	ip := h.convertStrIPToNetIP(bgpGlobal.RouterId)
	var err error = nil
	if ip == nil {
		err = errors.New(fmt.Sprintf("BGPGlobal: IP %s is not valid", bgpGlobal.RouterId))
		h.logger.Info(fmt.Sprintln("SendBGPGlobal: IP", bgpGlobal.RouterId, "is not valid"))
		return false, err
	}

	gConf := config.GlobalConfig{
		AS:                  uint32(bgpGlobal.ASNum),
		RouterId:            ip,
		UseMultiplePaths:    bgpGlobal.UseMultiplePaths,
		EBGPMaxPaths:        uint32(bgpGlobal.EBGPMaxPaths),
		EBGPAllowMultipleAS: bgpGlobal.EBGPAllowMultipleAS,
		IBGPMaxPaths:        uint32(bgpGlobal.IBGPMaxPaths),
	}
	h.server.GlobalConfigCh <- gConf
	return true, err
}

func (h *BGPHandler) CreateBGPGlobal(bgpGlobal *bgpd.BGPGlobal) (bool, error) {
	h.logger.Info(fmt.Sprintln("Create global config attrs:", bgpGlobal))
	return h.SendBGPGlobal(bgpGlobal)
}

func (h *BGPHandler) GetBGPGlobalState(rtrId string) (*bgpd.BGPGlobalState, error) {
	bgpGlobal := h.server.GetBGPGlobalState()
	bgpGlobalResponse := bgpd.NewBGPGlobalState()
	bgpGlobalResponse.AS = int32(bgpGlobal.AS)
	bgpGlobalResponse.RouterId = bgpGlobal.RouterId.String()
	bgpGlobalResponse.UseMultiplePaths = bgpGlobal.UseMultiplePaths
	bgpGlobalResponse.EBGPMaxPaths = int32(bgpGlobal.EBGPMaxPaths)
	bgpGlobalResponse.EBGPAllowMultipleAS = bgpGlobal.EBGPAllowMultipleAS
	bgpGlobalResponse.IBGPMaxPaths = int32(bgpGlobal.IBGPMaxPaths)
	bgpGlobalResponse.TotalPaths = int32(bgpGlobal.TotalPaths)
	bgpGlobalResponse.TotalPrefixes = int32(bgpGlobal.TotalPrefixes)
	return bgpGlobalResponse, nil
}

func (h *BGPHandler) GetBulkBGPGlobalState(index bgpd.Int,
	count bgpd.Int) (*bgpd.BGPGlobalStateGetInfo, error) {
	bgpGlobalStateBulk := bgpd.NewBGPGlobalStateGetInfo()
	bgpGlobalStateBulk.EndIdx = bgpd.Int(0)
	bgpGlobalStateBulk.Count = bgpd.Int(1)
	bgpGlobalStateBulk.More = false
	bgpGlobalStateBulk.BGPGlobalStateList[0], _ = h.GetBGPGlobalState("bgp")

	return bgpGlobalStateBulk, nil
}

func (h *BGPHandler) UpdateBGPGlobal(origG *bgpd.BGPGlobal, updatedG *bgpd.BGPGlobal,
	attrSet []bool) (bool, error) {
	h.logger.Info(fmt.Sprintln("Update global config attrs:", updatedG, "old config:", origG))
	return h.SendBGPGlobal(updatedG)
}

func (h *BGPHandler) DeleteBGPGlobal(bgpGlobal *bgpd.BGPGlobal) (bool, error) {
	h.logger.Info(fmt.Sprintln("Delete global config attrs:", bgpGlobal))
	return true, nil
}

func (h *BGPHandler) getIPAndIfIndexForNeighbor(neighborIP string,
	neighborIfIndex int32) (ip net.IP, ifIndex int32,
	err error) {
	if strings.TrimSpace(neighborIP) != "" {
		ip = net.ParseIP(strings.TrimSpace(neighborIP))
		ifIndex = 0
		if ip == nil {
			err = errors.New(fmt.Sprintf("Neighbor address %s not valid", neighborIP))
		}
	} else if neighborIfIndex != 0 {
		//neighbor address is a ifIndex
		var ipv4Intf string
		// @TODO: this needs to be interface once we decide to move listener
		ipv4Intf, err = h.server.IntfMgr.GetIPv4Information(neighborIfIndex)
		if err == nil {
			h.logger.Info(fmt.Sprintln("getIPAndIfIndexForNeighbor - Call ASICd",
				"to get ip address for interface with ifIndex: ", neighborIfIndex))
			ifIP, ipMask, err := net.ParseCIDR(ipv4Intf)
			if err != nil {
				h.logger.Err(fmt.Sprintln("getIPAndIfIndexForNeighbor - IpAddr",
					ipv4Intf, "of the interface", neighborIfIndex,
					"is not valid, error:", err))
				err = errors.New(fmt.Sprintf("IpAddr %s of the interface %d is not",
					"valid, error: %s", ipv4Intf, neighborIfIndex, err))
				return ip, ifIndex, err
			}
			if ipMask.Mask[len(ipMask.Mask)-1] < 252 {
				h.logger.Err(fmt.Sprintln("getIPAndIfIndexForNeighbor - IpAddr",
					ipv4Intf, "of the interface", neighborIfIndex,
					"is not /30 or /31 address"))
				err = errors.New(fmt.Sprintln("getIPAndIfIndexForNeighbor - IpAddr %s",
					"of the interface %s is not /30 or /31 address",
					ipv4Intf, neighborIfIndex))
				return ip, ifIndex, err
			}
			h.logger.Info(fmt.Sprintln("getIPAndIfIndexForNeighbor - IpAddr", ifIP,
				"of the interface", neighborIfIndex))
			ifIP[len(ifIP)-1] = ifIP[len(ifIP)-1] ^ (^ipMask.Mask[len(ipMask.Mask)-1])
			h.logger.Info(fmt.Sprintln("getIPAndIfIndexForNeighbor - IpAddr", ifIP,
				"of the neighbor interface"))
			ip = ifIP
			ifIndex = neighborIfIndex
			h.logger.Info(fmt.Sprintln("getIPAndIfIndexForNeighbor - Neighbor IP:",
				ip.String()))
		} else {
			h.logger.Err(fmt.Sprintln("getIPAndIfIndexForNeighbor - Neighbor IP", neighborIP,
				"or interface", neighborIfIndex, "not configured "))
		}
	}
	return ip, ifIndex, err
}

// Set BGP Default values.. This needs to move to API Layer once Northbound interfaces are implemented
// for all the listeners
func (h *BGPHandler) setDefault(pconf *config.NeighborConfig) {
	if pconf.BaseConfig.HoldTime == 0 { // default hold time is 180 seconds
		pconf.BaseConfig.HoldTime = 180
	}
	if pconf.BaseConfig.KeepaliveTime == 0 { // default keep alive time is 60 seconds
		pconf.BaseConfig.KeepaliveTime = 60
	}
}

func (h *BGPHandler) ValidateBGPNeighbor(bgpNeighbor *bgpd.BGPNeighbor) (pConf config.NeighborConfig,
	err error) {
	if bgpNeighbor == nil {
		return pConf, err
	}

	var ip net.IP
	var ifIndex int32
	ip, ifIndex, err = h.getIPAndIfIndexForNeighbor(bgpNeighbor.NeighborAddress, bgpNeighbor.IfIndex)
	if err != nil {
		h.logger.Info(fmt.Sprintln("ValidateBGPNeighbor: getIPAndIfIndexForNeighbor",
			"failed for neighbor address", bgpNeighbor.NeighborAddress,
			"and ifIndex", bgpNeighbor.IfIndex))
		return pConf, err
	}

	pConf = config.NeighborConfig{
		BaseConfig: config.BaseConfig{
			PeerAS:                  uint32(bgpNeighbor.PeerAS),
			LocalAS:                 uint32(bgpNeighbor.LocalAS),
			AuthPassword:            bgpNeighbor.AuthPassword,
			Description:             bgpNeighbor.Description,
			RouteReflectorClusterId: uint32(bgpNeighbor.RouteReflectorClusterId),
			RouteReflectorClient:    bgpNeighbor.RouteReflectorClient,
			MultiHopEnable:          bgpNeighbor.MultiHopEnable,
			MultiHopTTL:             uint8(bgpNeighbor.MultiHopTTL),
			ConnectRetryTime:        uint32(bgpNeighbor.ConnectRetryTime),
			HoldTime:                uint32(bgpNeighbor.HoldTime),
			KeepaliveTime:           uint32(bgpNeighbor.KeepaliveTime),
			BfdEnable:               bgpNeighbor.BfdEnable,
			AddPathsRx:              bgpNeighbor.AddPathsRx,
			AddPathsMaxTx:           uint8(bgpNeighbor.AddPathsMaxTx),
			MaxPrefixes:             uint32(bgpNeighbor.MaxPrefixes),
			MaxPrefixesThresholdPct: uint8(bgpNeighbor.MaxPrefixesThresholdPct),
			MaxPrefixesDisconnect:   bgpNeighbor.MaxPrefixesDisconnect,
			MaxPrefixesRestartTimer: uint8(bgpNeighbor.MaxPrefixesRestartTimer),
		},
		NeighborAddress: ip,
		IfIndex:         ifIndex,
		PeerGroup:       bgpNeighbor.PeerGroup,
	}
	h.setDefault(&pConf)
	return pConf, err
}

func (h *BGPHandler) SendBGPNeighbor(oldNeighbor *bgpd.BGPNeighbor,
	newNeighbor *bgpd.BGPNeighbor, attrSet []bool) (bool, error) {
	created := h.server.VerifyBgpGlobalConfig()
	if !created {
		return created,
			errors.New("Create BGP Local AS and router id before configuring Neighbor")
	}

	oldNeighConf, err := h.ValidateBGPNeighbor(oldNeighbor)
	if err != nil {
		return false, err
	}

	newNeighConf, err := h.ValidateBGPNeighbor(newNeighbor)
	if err != nil {
		return false, err
	}

	h.server.AddPeerCh <- server.PeerUpdate{oldNeighConf, newNeighConf, attrSet}
	return true, nil
}

func (h *BGPHandler) CreateBGPNeighbor(bgpNeighbor *bgpd.BGPNeighbor) (bool, error) {
	h.logger.Info(fmt.Sprintln("Create BGP neighbor attrs:", bgpNeighbor))
	return h.SendBGPNeighbor(nil, bgpNeighbor, make([]bool, 0))
}

func (h *BGPHandler) convertToThriftNeighbor(neighborState *config.NeighborState) *bgpd.BGPNeighborState {
	bgpNeighborResponse := bgpd.NewBGPNeighborState()
	bgpNeighborResponse.PeerAS = int32(neighborState.PeerAS)
	bgpNeighborResponse.LocalAS = int32(neighborState.LocalAS)
	bgpNeighborResponse.AuthPassword = neighborState.AuthPassword
	bgpNeighborResponse.PeerType = int8(neighborState.PeerType)
	bgpNeighborResponse.Description = neighborState.Description
	bgpNeighborResponse.NeighborAddress = neighborState.NeighborAddress.String()
	bgpNeighborResponse.IfIndex = neighborState.IfIndex
	bgpNeighborResponse.SessionState = int32(neighborState.SessionState)
	bgpNeighborResponse.RouteReflectorClusterId = int32(neighborState.RouteReflectorClusterId)
	bgpNeighborResponse.RouteReflectorClient = neighborState.RouteReflectorClient
	bgpNeighborResponse.MultiHopEnable = neighborState.MultiHopEnable
	bgpNeighborResponse.MultiHopTTL = int8(neighborState.MultiHopTTL)
	bgpNeighborResponse.ConnectRetryTime = int32(neighborState.ConnectRetryTime)
	bgpNeighborResponse.HoldTime = int32(neighborState.HoldTime)
	bgpNeighborResponse.KeepaliveTime = int32(neighborState.KeepaliveTime)
	bgpNeighborResponse.BfdNeighborState = neighborState.BfdNeighborState
	bgpNeighborResponse.PeerGroup = neighborState.PeerGroup
	bgpNeighborResponse.AddPathsRx = neighborState.AddPathsRx
	bgpNeighborResponse.AddPathsMaxTx = int8(neighborState.AddPathsMaxTx)

	bgpNeighborResponse.MaxPrefixes = int32(neighborState.MaxPrefixes)
	bgpNeighborResponse.MaxPrefixesThresholdPct = int8(neighborState.MaxPrefixesThresholdPct)
	bgpNeighborResponse.MaxPrefixesDisconnect = neighborState.MaxPrefixesDisconnect
	bgpNeighborResponse.MaxPrefixesRestartTimer = int8(neighborState.MaxPrefixesRestartTimer)
	bgpNeighborResponse.TotalPrefixes = int32(neighborState.TotalPrefixes)

	received := bgpd.NewBGPCounters()
	received.Notification = int64(neighborState.Messages.Received.Notification)
	received.Update = int64(neighborState.Messages.Received.Update)
	sent := bgpd.NewBGPCounters()
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

func (h *BGPHandler) GetBGPNeighborState(neighborAddr string,
	ifIndex int32) (*bgpd.BGPNeighborState, error) {
	ip, _, err := h.getIPAndIfIndexForNeighbor(neighborAddr, ifIndex)
	if err != nil {
		h.logger.Info(fmt.Sprintln("GetBGPNeighborState: getIPAndIfIndexForNeighbor",
			"failed for neighbor address", neighborAddr, "and ifIndex", ifIndex))
		return bgpd.NewBGPNeighborState(), err
	}

	bgpNeighborState := h.server.GetBGPNeighborState(ip.String())
	bgpNeighborResponse := h.convertToThriftNeighbor(bgpNeighborState)
	return bgpNeighborResponse, nil
}

func (h *BGPHandler) GetBulkBGPNeighborState(index bgpd.Int,
	count bgpd.Int) (*bgpd.BGPNeighborStateGetInfo, error) {
	nextIdx, currCount, bgpNeighbors := h.server.BulkGetBGPNeighbors(int(index), int(count))
	bgpNeighborsResponse := make([]*bgpd.BGPNeighborState, len(bgpNeighbors))
	for idx, item := range bgpNeighbors {
		bgpNeighborsResponse[idx] = h.convertToThriftNeighbor(item)
	}

	bgpNeighborStateBulk := bgpd.NewBGPNeighborStateGetInfo()
	bgpNeighborStateBulk.EndIdx = bgpd.Int(nextIdx)
	bgpNeighborStateBulk.Count = bgpd.Int(currCount)
	bgpNeighborStateBulk.More = (nextIdx != 0)
	bgpNeighborStateBulk.BGPNeighborStateList = bgpNeighborsResponse

	return bgpNeighborStateBulk, nil
}

func (h *BGPHandler) UpdateBGPNeighbor(origN *bgpd.BGPNeighbor, updatedN *bgpd.BGPNeighbor,
	attrSet []bool) (bool, error) {
	h.logger.Info(fmt.Sprintln("Update peer attrs:", updatedN))
	return h.SendBGPNeighbor(origN, updatedN, attrSet)
}

func (h *BGPHandler) DeleteBGPNeighbor(bgpNeighbor *bgpd.BGPNeighbor) (bool, error) {
	h.logger.Info(fmt.Sprintln("Delete BGP neighbor:", bgpNeighbor.NeighborAddress))
	ip := net.ParseIP(bgpNeighbor.NeighborAddress)
	if ip == nil {
		h.logger.Info(fmt.Sprintf("Can't delete BGP neighbor - IP[%s] not valid",
			bgpNeighbor.NeighborAddress))
		return false, errors.New(fmt.Sprintf("Neighbor Address %s not valid",
			bgpNeighbor.NeighborAddress))
	}
	h.server.RemPeerCh <- bgpNeighbor.NeighborAddress
	return true, nil
}

func (h *BGPHandler) PeerCommand(in *PeerConfigCommands, out *bool) error {
	h.PeerCommandCh <- *in
	h.logger.Info(fmt.Sprintln("Good peer command:", in))
	*out = true
	return nil
}

func (h *BGPHandler) ValidateBGPPeerGroup(peerGroup *bgpd.BGPPeerGroup) (group config.PeerGroupConfig,
	err error) {
	if peerGroup == nil {
		return group, err
	}

	group = config.PeerGroupConfig{
		BaseConfig: config.BaseConfig{
			PeerAS:                  uint32(peerGroup.PeerAS),
			LocalAS:                 uint32(peerGroup.LocalAS),
			AuthPassword:            peerGroup.AuthPassword,
			Description:             peerGroup.Description,
			RouteReflectorClusterId: uint32(peerGroup.RouteReflectorClusterId),
			RouteReflectorClient:    peerGroup.RouteReflectorClient,
			MultiHopEnable:          peerGroup.MultiHopEnable,
			MultiHopTTL:             uint8(peerGroup.MultiHopTTL),
			ConnectRetryTime:        uint32(peerGroup.ConnectRetryTime),
			HoldTime:                uint32(peerGroup.HoldTime),
			KeepaliveTime:           uint32(peerGroup.KeepaliveTime),
			AddPathsRx:              peerGroup.AddPathsRx,
			AddPathsMaxTx:           uint8(peerGroup.AddPathsMaxTx),
			MaxPrefixes:             uint32(peerGroup.MaxPrefixes),
			MaxPrefixesThresholdPct: uint8(peerGroup.MaxPrefixesThresholdPct),
			MaxPrefixesDisconnect:   peerGroup.MaxPrefixesDisconnect,
			MaxPrefixesRestartTimer: uint8(peerGroup.MaxPrefixesRestartTimer),
		},
		Name: peerGroup.Name,
	}

	return group, err
}

func (h *BGPHandler) SendBGPPeerGroup(oldGroup *bgpd.BGPPeerGroup,
	newGroup *bgpd.BGPPeerGroup, attrSet []bool) (
	bool, error) {
	oldGroupConf, err := h.ValidateBGPPeerGroup(oldGroup)
	if err != nil {
		return false, err
	}

	newGroupConf, err := h.ValidateBGPPeerGroup(newGroup)
	if err != nil {
		return false, err
	}

	h.server.AddPeerGroupCh <- server.PeerGroupUpdate{oldGroupConf, newGroupConf, attrSet}
	return true, nil
}

func (h *BGPHandler) CreateBGPPeerGroup(peerGroup *bgpd.BGPPeerGroup) (bool, error) {
	h.logger.Info(fmt.Sprintln("Create BGP neighbor attrs:", peerGroup))
	return h.SendBGPPeerGroup(nil, peerGroup, make([]bool, 0))
}

func (h *BGPHandler) UpdateBGPPeerGroup(origG *bgpd.BGPPeerGroup, updatedG *bgpd.BGPPeerGroup,
	attrSet []bool) (bool, error) {
	h.logger.Info(fmt.Sprintln("Update peer attrs:", updatedG))
	return h.SendBGPPeerGroup(origG, updatedG, attrSet)
}

func (h *BGPHandler) DeleteBGPPeerGroup(peerGroup *bgpd.BGPPeerGroup) (bool, error) {
	h.logger.Info(fmt.Sprintln("Delete BGP peer group:", peerGroup.Name))
	h.server.RemPeerGroupCh <- peerGroup.Name
	return true, nil
}

func (h *BGPHandler) GetBGPRouteState(network string, cidrLen int16,
	nextHop string) (*bgpd.BGPRouteState, error) {
	bgpRoute := h.server.AdjRib.GetBGPRoute(network)
	var err error = nil
	if bgpRoute == nil {
		err = errors.New(fmt.Sprintf("Route not found for destination %s", network))
	}
	return bgpRoute, err
}

func (h *BGPHandler) GetBulkBGPRouteState(index bgpd.Int,
	count bgpd.Int) (*bgpd.BGPRouteStateGetInfo, error) {
	nextIdx, currCount, bgpRoutes := h.server.AdjRib.BulkGetBGPRoutes(int(index), int(count))

	bgpRoutesBulk := bgpd.NewBGPRouteStateGetInfo()
	bgpRoutesBulk.EndIdx = bgpd.Int(nextIdx)
	bgpRoutesBulk.Count = bgpd.Int(currCount)
	bgpRoutesBulk.More = (nextIdx != 0)
	bgpRoutesBulk.BGPRouteStateList = bgpRoutes

	return bgpRoutesBulk, nil
}

func convertThriftToPolicyConditionConfig(
	cfg *bgpd.BGPPolicyCondition) *utilspolicy.PolicyConditionConfig {
	destIPMatch := utilspolicy.PolicyDstIpMatchPrefixSetCondition{
		Prefix: utilspolicy.PolicyPrefix{
			IpPrefix:        cfg.IpPrefix,
			MasklengthRange: cfg.MaskLengthRange,
		},
	}
	return &utilspolicy.PolicyConditionConfig{
		Name:                          cfg.Name,
		ConditionType:                 cfg.ConditionType,
		MatchDstIpPrefixConditionInfo: destIPMatch,
	}
}

func (h *BGPHandler) CreateBGPPolicyCondition(cfg *bgpd.BGPPolicyCondition) (val bool, err error) {
	h.logger.Info(fmt.Sprintln("CreatePolicyConditioncfg"))
	switch cfg.ConditionType {
	case "MatchDstIpPrefix":
		policyCfg := convertThriftToPolicyConditionConfig(cfg)
		val = true
		h.bgpPE.ConditionCfgCh <- *policyCfg
		break
	default:
		h.logger.Info(fmt.Sprintln("Unknown condition type ", cfg.ConditionType))
		err = errors.New(fmt.Sprintf("Unknown condition type %s", cfg.ConditionType))
	}
	return val, err
}

func (h *BGPHandler) GetBGPPolicyConditionState(name string) (*bgpd.BGPPolicyConditionState, error) {
	//return policy.GetBulkBGPPolicyConditionState(fromIndex, rcount)
	return nil, errors.New("BGPPolicyConditionState not supported yet")
}

func (h *BGPHandler) GetBulkBGPPolicyConditionState(fromIndex bgpd.Int, rcount bgpd.Int) (
	policyConditions *bgpd.BGPPolicyConditionStateGetInfo, err error) {
	//return policy.GetBulkBGPPolicyConditionState(fromIndex, rcount)
	return nil, nil
}

func (h *BGPHandler) UpdateBGPPolicyCondition(origC *bgpd.BGPPolicyCondition,
	updatedC *bgpd.BGPPolicyCondition,
	attrSet []bool) (val bool, err error) {
	return val, err
}

func (h *BGPHandler) DeleteBGPPolicyCondition(cfg *bgpd.BGPPolicyCondition) (val bool, err error) {
	h.bgpPE.ConditionDelCh <- cfg.Name
	return val, err
}

func convertThriftToPolicyActionConfig(cfg *bgpd.BGPPolicyAction) *utilspolicy.PolicyActionConfig {
	return &utilspolicy.PolicyActionConfig{
		Name:            cfg.Name,
		ActionType:      cfg.ActionType,
		GenerateASSet:   cfg.GenerateASSet,
		SendSummaryOnly: cfg.SendSummaryOnly,
	}
}

func (h *BGPHandler) CreateBGPPolicyAction(cfg *bgpd.BGPPolicyAction) (val bool, err error) {
	h.logger.Info(fmt.Sprintln("CreatePolicyAction"))
	switch cfg.ActionType {
	case "Aggregate":
		actionCfg := convertThriftToPolicyActionConfig(cfg)
		val = true
		h.bgpPE.ActionCfgCh <- *actionCfg
		break
	default:
		h.logger.Info(fmt.Sprintln("Unknown action type ", cfg.ActionType))
		err = errors.New(fmt.Sprintf("Unknown action type %s", cfg.ActionType))
	}
	return val, err
}

func (h *BGPHandler) GetBGPPolicyActionState(name string) (*bgpd.BGPPolicyActionState, error) {
	//return policy.GetBulkBGPPolicyActionState(fromIndex, rcount)
	return nil, errors.New("BGPPolicyActionState not supported yet")
}

func (h *BGPHandler) GetBulkBGPPolicyActionState(fromIndex bgpd.Int, rcount bgpd.Int) (
	policyActions *bgpd.BGPPolicyActionStateGetInfo, err error) { //(routes []*bgpd.Routes, err error) {
	//return policy.GetBulkBGPPolicyActionState(fromIndex, rcount)
	return nil, nil
}

func (h *BGPHandler) UpdateBGPPolicyAction(origC *bgpd.BGPPolicyAction, updatedC *bgpd.BGPPolicyAction,
	attrSet []bool) (val bool, err error) {
	return val, err
}

func (h *BGPHandler) DeleteBGPPolicyAction(cfg *bgpd.BGPPolicyAction) (val bool, err error) {
	h.bgpPE.ActionDelCh <- cfg.Name
	return val, err
}

func convertThriftToPolicyStmtConfig(cfg *bgpd.BGPPolicyStmt) *utilspolicy.PolicyStmtConfig {
	return &utilspolicy.PolicyStmtConfig{
		Name:            cfg.Name,
		MatchConditions: cfg.MatchConditions,
		Conditions:      cfg.Conditions,
		Actions:         cfg.Actions,
	}
}

func (h *BGPHandler) CreateBGPPolicyStmt(cfg *bgpd.BGPPolicyStmt) (val bool, err error) {
	h.logger.Info(fmt.Sprintln("CreatePolicyStmt"))
	val = true
	stmtCfg := convertThriftToPolicyStmtConfig(cfg)
	h.bgpPE.StmtCfgCh <- *stmtCfg
	return val, err
}

func (h *BGPHandler) GetBGPPolicyStmtState(name string) (*bgpd.BGPPolicyStmtState, error) {
	//return policy.GetBulkBGPPolicyStmtState(fromIndex, rcount)
	return nil, errors.New("BGPPolicyStmtState not supported yet")
}

func (h *BGPHandler) GetBulkBGPPolicyStmtState(fromIndex bgpd.Int, rcount bgpd.Int) (
	policyStmts *bgpd.BGPPolicyStmtStateGetInfo, err error) {
	//return policy.GetBulkBGPPolicyStmtState(fromIndex, rcount)
	return nil, nil
}

func (h *BGPHandler) UpdateBGPPolicyStmt(origC *bgpd.BGPPolicyStmt,
	updatedC *bgpd.BGPPolicyStmt, attrSet []bool) (
	val bool, err error) {
	return val, err
}

func (h *BGPHandler) DeleteBGPPolicyStmt(cfg *bgpd.BGPPolicyStmt) (val bool, err error) {
	//return policy.DeleteBGPPolicyStmt(name)
	h.bgpPE.StmtDelCh <- cfg.Name
	return true, nil
}

func convertThriftToPolicyDefintionConfig(
	cfg *bgpd.BGPPolicyDefinition) *utilspolicy.PolicyDefinitionConfig {
	stmtPrecedenceList := make([]utilspolicy.PolicyDefinitionStmtPrecedence, 0)
	for i := 0; i < len(cfg.StatementList); i++ {
		stmtPrecedence := utilspolicy.PolicyDefinitionStmtPrecedence{
			Precedence: int(cfg.StatementList[i].Precedence),
			Statement:  cfg.StatementList[i].Statement,
		}
		stmtPrecedenceList = append(stmtPrecedenceList, stmtPrecedence)
	}

	return &utilspolicy.PolicyDefinitionConfig{
		Name:                       cfg.Name,
		Precedence:                 int(cfg.Precedence),
		MatchType:                  cfg.MatchType,
		PolicyDefinitionStatements: stmtPrecedenceList,
	}
}

func (h *BGPHandler) CreateBGPPolicyDefinition(cfg *bgpd.BGPPolicyDefinition) (val bool, err error) {
	h.logger.Info(fmt.Sprintln("CreatePolicyDefinition"))
	val = true
	definitionCfg := convertThriftToPolicyDefintionConfig(cfg)
	h.bgpPE.DefinitionCfgCh <- *definitionCfg
	return val, err
}

func (h *BGPHandler) GetBGPPolicyDefinitionState(name string) (*bgpd.BGPPolicyDefinitionState, error) {
	//return policy.GetBulkBGPPolicyDefinitionState(fromIndex, rcount)
	return nil, errors.New("BGPPolicyDefinitionState not supported yet")
}

func (h *BGPHandler) GetBulkBGPPolicyDefinitionState(fromIndex bgpd.Int, rcount bgpd.Int) (
	policyStmts *bgpd.BGPPolicyDefinitionStateGetInfo, err error) { //(routes []*bgpd.BGPRouteState, err error) {
	//return policy.GetBulkBGPPolicyDefinitionState(fromIndex, rcount)
	return nil, nil
}

func (h *BGPHandler) UpdateBGPPolicyDefinition(origC *bgpd.BGPPolicyDefinition,
	updatedC *bgpd.BGPPolicyDefinition,
	attrSet []bool) (val bool, err error) {
	return val, err
}

func (h *BGPHandler) DeleteBGPPolicyDefinition(cfg *bgpd.BGPPolicyDefinition) (val bool, err error) {
	h.bgpPE.DefinitionDelCh <- cfg.Name
	return val, err
}
