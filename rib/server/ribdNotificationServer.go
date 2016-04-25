// ribNotify.go
package server

import (
	"fmt"
	"github.com/op/go-nanomsg"
	"time"
)
type NotificationMsg struct {
	pub_socket *nanomsg.PubSocket
	msg    []byte
	eventInfo string
}
func (ribdServiceHandler *RIBDServer) NotificationServer() {
	logger.Info(fmt.Sprintln("Starting notification server loop"))
	for {
		notificationMsg := <-ribdServiceHandler.NotificationChannel
		logger.Info(fmt.Sprintln("Event received with eventInfo: ", notificationMsg.eventInfo))
	    routeEventInfo := RouteEventInfo{timeStamp: time.Now().String(), eventInfo: notificationMsg.eventInfo}
	    localRouteEventsDB = append(localRouteEventsDB, routeEventInfo)
		notificationMsg.pub_socket.Send(notificationMsg.msg, nanomsg.DontWait)
	}
}

