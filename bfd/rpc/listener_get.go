//
//Copyright [2016] [SnapRoute Inc]
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//	 Unless required by applicable law or agreed to in writing, software
//	 distributed under the License is distributed on an "AS IS" BASIS,
//	 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//	 See the License for the specific language governing permissions and
//	 limitations under the License.
//
// _______  __       __________   ___      _______.____    __    ____  __  .___________.  ______  __    __  
// |   ____||  |     |   ____\  \ /  /     /       |\   \  /  \  /   / |  | |           | /      ||  |  |  | 
// |  |__   |  |     |  |__   \  V  /     |   (----` \   \/    \/   /  |  | `---|  |----`|  ,----'|  |__|  | 
// |   __|  |  |     |   __|   >   <       \   \      \            /   |  |     |  |     |  |     |   __   | 
// |  |     |  `----.|  |____ /  .  \  .----)   |      \    /\    /    |  |     |  |     |  `----.|  |  |  | 
// |__|     |_______||_______/__/ \__\ |_______/        \__/  \__/     |__|     |__|      \______||__|  |__| 
//                                                                                                           

package rpc

import (
	"bfdd"
	"fmt"
)

func (h *BFDHandler) GetBfdGlobalState(bfd string) (*bfdd.BfdGlobalState, error) {
	h.logger.Info(fmt.Sprintln("Get Global attrs"))
	bfdGlobalStateResponse := bfdd.NewBfdGlobalState()
	gState := h.server.GetBfdGlobalState()
	bfdGlobalState := h.convertGlobalStateToThrift(*gState)
	bfdGlobalStateResponse = bfdGlobalState
	return bfdGlobalStateResponse, nil
}

func (h *BFDHandler) GetBfdSessionState(ipAddr string) (*bfdd.BfdSessionState, error) {
	h.logger.Info(fmt.Sprintln("Get Session attrs for neighbor ", ipAddr))
	bfdSessionStateResponse := bfdd.NewBfdSessionState()
	sessionState := h.server.GetBfdSessionState(ipAddr)
	bfdSessionState := h.convertSessionStateToThrift(*sessionState)
	bfdSessionStateResponse = bfdSessionState
	return bfdSessionStateResponse, nil
}

func (h *BFDHandler) GetBfdSessionParamState(paramName string) (*bfdd.BfdSessionParamState, error) {
	h.logger.Info(fmt.Sprintln("Get Session Params attrs for ", paramName))
	bfdSessionParamStateResponse := bfdd.NewBfdSessionParamState()
	sessionParamState := h.server.GetBfdSessionParamState(paramName)
	bfdSessionParamState := h.convertSessionParamStateToThrift(*sessionParamState)
	bfdSessionParamStateResponse = bfdSessionParamState
	return bfdSessionParamStateResponse, nil
}
