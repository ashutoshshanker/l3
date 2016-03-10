// hw.go
package vxlan

import (
	"encoding/json"
	"io/ioutil"
)

type ClientJson struct {
	Name string `json:Name`
	Port int    `json:Port`
}

// look up the various other daemons based on c string
func GetClientPort(paramsFile string, c string) int {
	var clientsList []ClientJson

	bytes, err := ioutil.ReadFile(paramsFile)
	if err != nil {
		//StpLogger("ERROR", fmt.Sprintf("Error in reading configuration file:%s err:%s\n", paramsFile, err))
		return 0
	}

	err = json.Unmarshal(bytes, &clientsList)
	if err != nil {
		//StpLogger("ERROR", "Error in Unmarshalling Json")
		return 0
	}

	for _, client := range clientsList {
		if client.Name == c {
			return client.Port
		}
	}
	return 0
}
