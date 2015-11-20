package main
import ("arpd")
		  

func (m ARPServiceHandler) RestolveArpIPV4( targetIp         string, 
                                            ifName           string) (rc arpd.Int, err error) {
	 logger.Println("ARP Request received")
	 return 0, nil
}

