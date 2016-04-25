package server

import ()

func (server *BFDServer) processSessionParamConfig(paramConfig SessionParamConfig) error {
	sessionParam, exist := server.bfdGlobal.SessionParams[paramConfig.Name]
	if !exist {
		sessionParam.state.Name = paramConfig.Name
		sessionParam.state.LocalMultiplier = paramConfig.LocalMultiplier
		sessionParam.state.DesiredMinTxInterval = paramConfig.DesiredMinTxInterval * 1000
		sessionParam.state.RequiredMinRxInterval = paramConfig.RequiredMinRxInterval * 1000
		sessionParam.state.RequiredMinEchoRxInterval = paramConfig.RequiredMinEchoRxInterval * 1000
		sessionParam.state.DemandEnabled = paramConfig.DemandEnabled
		sessionParam.state.AuthenticationEnabled = paramConfig.AuthenticationEnabled
		sessionParam.state.AuthenticationType = paramConfig.AuthenticationType
		sessionParam.state.AuthenticationKeyId = paramConfig.AuthenticationKeyId
		sessionParam.state.AuthenticationData = paramConfig.AuthenticationData
		server.UpdateBfdSessionsUsingParam(sessionParam.state.Name)
	}
	return nil
}

func (server *BFDServer) processSessionParamDelete(paramName string) error {
	_, exist := server.bfdGlobal.SessionParams[paramName]
	if exist {
		delete(server.bfdGlobal.SessionParams, paramName)
		server.UpdateBfdSessionsUsingParam(paramName)
	}
	return nil
}
