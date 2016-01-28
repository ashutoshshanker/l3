package main

import (
	"flag"
	"git.apache.org/thrift.git/lib/go/thrift"
	"log"
	"log/syslog"
	"os"
	"ribd"
)

var logger *log.Logger
var routeServiceHandler *RouteServiceHandler

func main() {
	var transport thrift.TServerTransport
	var err error
	var addr = "localhost:5000"

	logger = log.New(os.Stdout, "RIBD :", log.Ldate|log.Ltime|log.Lshortfile)

	syslogger, err := syslog.New(syslog.LOG_NOTICE|syslog.LOG_INFO|syslog.LOG_DAEMON, "RIBD")
	if err == nil {
		syslogger.Info("### RIB Daemon started")
		logger.SetOutput(syslogger)
	}

	paramsDir := flag.String("params", "", "Directory Location for config files")
	logger.Println("### Params Dir ", paramsDir)
	flag.Parse()

	transport, err = thrift.NewTServerSocket(addr)
	if err != nil {
		logger.Println("Failed to create Socket with:", addr)
	}
	handler := NewRouteServiceHandler(*paramsDir)
	if handler == nil {
		logger.Println("handler nill")
		return
	} 
	routeServiceHandler = handler
	processor := ribd.NewRouteServiceProcessor(handler)
	transportFactory := thrift.NewTBufferedTransportFactory(8192)
	protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
	server := thrift.NewTSimpleServer4(processor, transport, transportFactory, protocolFactory)
	logger.Println("Starting RIB daemon")
	server.Serve()
}
