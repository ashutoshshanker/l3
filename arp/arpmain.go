package main 
import ("os"
		  "log"
        "git.apache.org/thrift.git/lib/go/thrift"
        "arpd"
        )

var logger *log.Logger

func main () {
    var transport thrift.TServerTransport
    var err error
	 var addr = "localhost:6000"

    logger = log.New(os.Stdout, "ARPD :", log.Ldate|log.Ltime|log.Lshortfile)                                

    transport, err = thrift.NewTServerSocket(addr)
	 if err != nil {
		  logger.Println("Failed to create Socket with:", addr)
	 }
    handler := NewARPServiceHandler()
    processor := arpd.NewARPServiceProcessor(handler)
    transportFactory := thrift.NewTBufferedTransportFactory(8192) 
    protocolFactory := thrift.NewTBinaryProtocolFactoryDefault()
    server := thrift.NewTSimpleServer4(processor, transport, transportFactory, protocolFactory)
    logger.Println("Starting ARP daemon")
    server.Serve()
}
