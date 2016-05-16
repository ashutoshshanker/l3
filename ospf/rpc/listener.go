Copyright [2016] [SnapRoute Inc]

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

	 Unless required by applicable law or agreed to in writing, software
	 distributed under the License is distributed on an "AS IS" BASIS,
	 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
	 See the License for the specific language governing permissions and
	 limitations under the License.
package rpc

import (
	//    "ospfd"
	//    "fmt"
	//    "l3/ospf/config"
	"l3/ospf/server"
	"utils/logging"
	//    "net"
)

type OSPFHandler struct {
	server *server.OSPFServer
	logger *logging.Writer
}

func NewOSPFHandler(server *server.OSPFServer, logger *logging.Writer) *OSPFHandler {
	h := new(OSPFHandler)
	h.server = server
	h.logger = logger
	return h
}
