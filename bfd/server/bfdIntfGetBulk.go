package server

import (
//"fmt"
)

func (server *BFDServer) GetBulkBfdIntfStates(idx int, cnt int) (int, int, []IntfState) {
	var nextIdx int
	var count int
	length := len(server.bfdGlobal.InterfacesIdSlice)
	if idx+cnt >= length {
		count = length - idx
		nextIdx = 0
	} else {
		nextIdx = idx + count + 1
	}
	result := make([]IntfState, count)
	for i := idx; i < count; i++ {
		intfId := server.bfdGlobal.InterfacesIdSlice[i]
		result[i].InterfaceId = server.bfdGlobal.Interfaces[intfId].conf.InterfaceId
		result[i].Enabled = server.bfdGlobal.Interfaces[intfId].Enabled
		result[i].NumSessions = server.bfdGlobal.Interfaces[intfId].NumSessions
		result[i].LocalMultiplier = server.bfdGlobal.Interfaces[intfId].conf.LocalMultiplier
		result[i].DesiredMinTxInterval = server.bfdGlobal.Interfaces[intfId].conf.DesiredMinTxInterval
		result[i].RequiredMinRxInterval = server.bfdGlobal.Interfaces[intfId].conf.RequiredMinRxInterval
		result[i].RequiredMinEchoRxInterval = server.bfdGlobal.Interfaces[intfId].conf.RequiredMinEchoRxInterval
		result[i].DemandEnabled = server.bfdGlobal.Interfaces[intfId].conf.DemandEnabled
		result[i].AuthenticationEnabled = server.bfdGlobal.Interfaces[intfId].conf.AuthenticationEnabled
		result[i].AuthenticationType = server.bfdGlobal.Interfaces[intfId].conf.AuthenticationType
		result[i].AuthenticationKeyId = server.bfdGlobal.Interfaces[intfId].conf.AuthenticationKeyId
		result[i].SequenceNumber = server.bfdGlobal.Interfaces[intfId].conf.SequenceNumber
		result[i].AuthenticationData = server.bfdGlobal.Interfaces[intfId].conf.AuthenticationData
	}
	return nextIdx, count, result
}
