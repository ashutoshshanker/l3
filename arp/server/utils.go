package server

import (
	"strconv"
	"strings"
)

//Note: Caller validates that portStr is a valid port range string
func parsePortRange(portStr string) (int, int, error) {
	portNums := strings.Split(portStr, "-")
	startPort, err := strconv.Atoi(portNums[0])
	if err != nil {
		return 0, 0, err
	}
	endPort, err := strconv.Atoi(portNums[1])
	if err != nil {
		return 0, 0, err
	}
	return startPort, endPort, nil
}

/*
 * Utility function to parse from a user specified port string to a list of ports.
 * Supported formats for port string shown below:
 * - 1,2,3,10 (comma separate list of ports)
 * - 1-10,15-18 (hyphen separated port ranges)
 * - 1,2,6-9 (combination of comma and hyphen separated strings)
 */
func ParseUsrPortStrToPortList(usrPortStr string) ([]int32, error) {
	var portList []int32 = make([]int32, 0)
	if len(usrPortStr) == 0 {
		return nil, nil
	}
	//Handle ',' separated strings
	if strings.Contains(usrPortStr, ",") {
		commaSepList := strings.Split(usrPortStr, ",")
		for _, subStr := range commaSepList {
			//Substr contains '-' separated range
			if strings.Contains(subStr, "-") {
				startPort, endPort, err := parsePortRange(subStr)
				if err != nil {
					return nil, err
				}
				for port := startPort; port <= endPort; port++ {
					portList = append(portList, int32(port))
				}
			} else {
				//Substr is a port number
				port, err := strconv.Atoi(subStr)
				if err != nil {
					return nil, err
				}
				portList = append(portList, int32(port))
			}
		}
	} else if strings.Contains(usrPortStr, "-") {
		//Handle '-' separated range
		startPort, endPort, err := parsePortRange(usrPortStr)
		if err != nil {
			return nil, err
		}
		for port := startPort; port <= endPort; port++ {
			portList = append(portList, int32(port))
		}
	} else {
		//Handle single port number
		port, err := strconv.Atoi(usrPortStr)
		if err != nil {
			return nil, err
		}
		portList = append(portList, int32(port))
	}
	return portList, nil
}

func ConvertPortListToPortStr(portList []int32) string {
	var portStr string = ""
	for _, port := range portList {
		portStr += strconv.Itoa(int(port)) + ","
	}
	return (strings.TrimRight(portStr, ","))
}
