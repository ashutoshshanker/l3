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

// ribdPolicyServer.go
package server

import (
	"encoding/json"
	"fmt"
	"github.com/op/go-nanomsg"
	"l3/rib/ribdCommonDefs"
	"models"
	"ribd"
)

func (ribdServiceHandler *RIBDServer) PolicyConditionNotificationSend(PUB *nanomsg.PubSocket, cfg ribd.PolicyCondition, evt int) {
	logger.Println("PolicyConditionNotificationSend")
	msgBuf := models.PolicyCondition{}
	models.ConvertThriftToribdPolicyConditionObj(&cfg, &msgBuf)
	/*	msgBuf := models.PolicyConditionConfig{
				Name : cfg.Name,
				ConditionType   :cfg.ConditionType,
				Protocol        : cfg.Protocol,
				IpPrefix        : cfg.IpPrefix,
				MasklengthRange : cfg.MasklengthRange
	}*/
	msgbufbytes, err := json.Marshal(msgBuf)
	msg := ribdCommonDefs.RibdNotifyMsg{MsgType: uint16(evt), MsgBuf: msgbufbytes}
	buf, err := json.Marshal(msg)
	if err != nil {
		logger.Println("Error in marshalling Json")
		return
	}
	var evtStr string
	if evt == ribdCommonDefs.NOTIFY_POLICY_CONDITION_CREATED {
		evtStr = " POLICY_CONDITION_CREATED "
	} else if evt == ribdCommonDefs.NOTIFY_POLICY_CONDITION_DELETED {
		evtStr = " POLICY_CONDITION_DELETED "
	}
	eventInfo := evtStr + " for condition " + cfg.Name + " " + " type " + cfg.ConditionType
	logger.Info(fmt.Sprintln("Adding ", evtStr, " to notification channel"))
	ribdServiceHandler.NotificationChannel <- NotificationMsg{PUB, buf, eventInfo}
}

func (ribdServiceHandler *RIBDServer) PolicyStmtNotificationSend(PUB *nanomsg.PubSocket, cfg ribd.PolicyStmt, evt int) {
	logger.Println("PolicyStmtNotificationSend")
	msgBuf := models.PolicyStmt{}
	models.ConvertThriftToribdPolicyStmtObj(&cfg, &msgBuf)
	/*	msgBuf := models.PolicyStmtConfig{
					Name               : cfg.Name,
					MatchConditions    : cfg.MatchConditions,
					Action             : cfg.Action
		}
		msgBuf.Conditions = make([]string,0)
		for i := 0;i<len(cfg.Conditions);i++ {
			msgBuf.Conditions = append(msgBuf.Conditions,cfg.Conditions[i])
		}*/
	msgbufbytes, err := json.Marshal(msgBuf)
	msg := ribdCommonDefs.RibdNotifyMsg{MsgType: uint16(evt), MsgBuf: msgbufbytes}
	buf, err := json.Marshal(msg)
	if err != nil {
		logger.Println("Error in marshalling Json")
		return
	}
	var evtStr string
	if evt == ribdCommonDefs.NOTIFY_POLICY_STMT_CREATED {
		evtStr = " POLICY_STMT_CREATED "
	} else if evt == ribdCommonDefs.NOTIFY_POLICY_STMT_DELETED {
		evtStr = " POLICY_STMT_DELETED "
	}
	eventInfo := evtStr + " for policy stmt " + cfg.Name
	logger.Info(fmt.Sprintln("Adding ", evtStr, " to notification channel"))
	ribdServiceHandler.NotificationChannel <- NotificationMsg{PUB, buf, eventInfo}
}

func (ribdServiceHandler *RIBDServer) PolicyDefinitionNotificationSend(PUB *nanomsg.PubSocket, cfg ribd.PolicyDefinition, evt int) {
	logger.Println("PolicyDefinitionNotificationSend")
	msgBuf := models.PolicyDefinition{}
	models.ConvertThriftToribdPolicyDefinitionObj(&cfg, &msgBuf)
	/*	msgBuf := models.PolicyDefinitionConfig{
					Name        : cfg.Name,
					Priority    : cfg.Priority,
					MatchType             : cfg.MatchType,
					PolicyType  : cfg.PolicyType
		}
		msgBuf.PolicyDefinitionStatements = make([]policy.PolicyDefinitionStmtPrecedence, 0)
		var policyDefinitionStatement policy.PolicyDefinitionStmtPrecedence
		for i := 0; i < len(cfg.StatementList); i++ {
			policyDefinitionStatement.Precedence = int(cfg.StatementList[i].Priority)
			policyDefinitionStatement.Statement = cfg.StatementList[i].Statement
			msgBuf.PolicyDefinitionStatements = append(msgBuf.PolicyDefinitionStatements, policyDefinitionStatement)
		}*/
	msgbufbytes, err := json.Marshal(msgBuf)
	msg := ribdCommonDefs.RibdNotifyMsg{MsgType: uint16(evt), MsgBuf: msgbufbytes}
	buf, err := json.Marshal(msg)
	if err != nil {
		logger.Println("Error in marshalling Json")
		return
	}
	var evtStr string
	if evt == ribdCommonDefs.NOTIFY_POLICY_DEFINITION_CREATED {
		evtStr = " POLICY_DEFINITION_CREATED "
	} else if evt == ribdCommonDefs.NOTIFY_POLICY_DEFINITION_DELETED {
		evtStr = " POLICY_DEFINITION_DELETED "
	}
	eventInfo := evtStr + " for policy " + cfg.Name
	logger.Info(fmt.Sprintln("Adding ", evtStr, " to notification channel"))
	ribdServiceHandler.NotificationChannel <- NotificationMsg{PUB, buf, eventInfo}
}

func (ribdServiceHandler *RIBDServer) StartPolicyServer() {
	logger.Info("Starting the policy server loop")
	for {
		select {
		case condCreateConf := <-ribdServiceHandler.PolicyConditionCreateConfCh:
			logger.Info("received message on PolicyConditionCreateConfCh channel")
			_, err := ribdServiceHandler.ProcessPolicyConditionConfigCreate(condCreateConf, ribdServiceHandler.GlobalPolicyEngineDB)
			if err == nil {
				ribdServiceHandler.PolicyConditionNotificationSend(RIBD_POLICY_PUB, *condCreateConf, ribdCommonDefs.NOTIFY_POLICY_CONDITION_CREATED)
				ribdServiceHandler.ProcessPolicyConditionConfigCreate(condCreateConf, ribdServiceHandler.PolicyEngineDB)
			}
		case condDeleteConf := <-ribdServiceHandler.PolicyConditionDeleteConfCh:
			logger.Info("received message on PolicyConditionDeleteConfCh channel")
			_, err := ribdServiceHandler.ProcessPolicyConditionConfigDelete(condDeleteConf, ribdServiceHandler.GlobalPolicyEngineDB)
			if err == nil {
				ribdServiceHandler.PolicyConditionNotificationSend(RIBD_POLICY_PUB, *condDeleteConf, ribdCommonDefs.NOTIFY_POLICY_CONDITION_DELETED)
				ribdServiceHandler.ProcessPolicyConditionConfigDelete(condDeleteConf, ribdServiceHandler.PolicyEngineDB)
			}
		case stmtCreateConf := <-ribdServiceHandler.PolicyStmtCreateConfCh:
			logger.Info("received message on PolicyStmtCreateConfCh channel")
			err := ribdServiceHandler.ProcessPolicyStmtConfigCreate(stmtCreateConf, GlobalPolicyEngineDB)
			if err == nil {
				ribdServiceHandler.PolicyStmtNotificationSend(RIBD_POLICY_PUB, *stmtCreateConf, ribdCommonDefs.NOTIFY_POLICY_STMT_CREATED)
				ribdServiceHandler.ProcessPolicyStmtConfigCreate(stmtCreateConf, ribdServiceHandler.PolicyEngineDB)
			}
		case stmtDeleteConf := <-ribdServiceHandler.PolicyStmtDeleteConfCh:
			logger.Info("received message on PolicyStmtDeleteConfCh channel")
			err := ribdServiceHandler.ProcessPolicyStmtConfigDelete(stmtDeleteConf, GlobalPolicyEngineDB)
			if err == nil {
				ribdServiceHandler.PolicyStmtNotificationSend(RIBD_POLICY_PUB, *stmtDeleteConf, ribdCommonDefs.NOTIFY_POLICY_STMT_DELETED)
				ribdServiceHandler.ProcessPolicyStmtConfigDelete(stmtDeleteConf, ribdServiceHandler.PolicyEngineDB)
			}
		case policyCreateConf := <-ribdServiceHandler.PolicyDefinitionCreateConfCh:
			logger.Info("received message on PolicyDefinitionCreateConfCh channel")
			err := ribdServiceHandler.ProcessPolicyDefinitionConfigCreate(policyCreateConf, GlobalPolicyEngineDB)
			if err == nil {
				ribdServiceHandler.PolicyDefinitionNotificationSend(RIBD_POLICY_PUB, *policyCreateConf, ribdCommonDefs.NOTIFY_POLICY_DEFINITION_CREATED)
				ribdServiceHandler.ProcessPolicyDefinitionConfigCreate(policyCreateConf, ribdServiceHandler.PolicyEngineDB)
			}
		case policyDeleteConf := <-ribdServiceHandler.PolicyDefinitionDeleteConfCh:
			logger.Info("received message on PolicyDefinitionDeleteConfCh channel")
			err := ribdServiceHandler.ProcessPolicyDefinitionConfigDelete(policyDeleteConf, GlobalPolicyEngineDB)
			if err == nil {
				ribdServiceHandler.PolicyDefinitionNotificationSend(RIBD_POLICY_PUB, *policyDeleteConf, ribdCommonDefs.NOTIFY_POLICY_DEFINITION_DELETED)
				ribdServiceHandler.ProcessPolicyDefinitionConfigDelete(policyDeleteConf, ribdServiceHandler.PolicyEngineDB)
			}
		case info := <-ribdServiceHandler.PolicyUpdateApplyCh:
			logger.Info("received message on PolicyUpdateApplyCh channel")
			//update the global policyEngineDB
			ribdServiceHandler.UpdateApplyPolicy(info, false, GlobalPolicyEngineDB)
		}
	}
}
