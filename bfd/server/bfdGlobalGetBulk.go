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
package server

import (
	"fmt"
)

func (server *BFDServer) GetBfdGlobalState() *GlobalState {
	result := new(GlobalState)
	ent := server.bfdGlobal

	result.Enable = ent.Enabled
	result.NumSessions = ent.NumSessions
	result.NumUpSessions = ent.NumUpSessions
	result.NumDownSessions = ent.NumDownSessions
	result.NumAdminDownSessions = ent.NumAdminDownSessions

	server.logger.Info(fmt.Sprintln("Global State:", result))
	return result
}
