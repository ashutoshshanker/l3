namespace go dhcprelayd

typedef i32 int

enum RelayAgentSubOptType {
    ClientIdSubOpt = 0,
    RemoteIdSubOpt
}

struct RelayAgentInfoField {
    1: RelayAgentSubOptType agentType,
    2: i32 len,
    3: i32 value, // this should be in octets
}

struct RelayAgentInfo {
    1: i32 code,
    2: i32 len,
    3: RelayAgentInfoField raif,
}

/*
 * This DS will be used while adding/deleting Relay Agent.
 * It will take Ip Subnet, If_Index and ... as it fields
 */
struct DhcpRelayConf {
    1: string IpSubnet,
    2: string IfIndex, // @TODO: Need to check if_index type 
}

service DhcpRelayServer {
    void AddRelayAgent(1: DhcpRelayConf dhcprelayConf);
    void DelRelayAgent();
    void UpdRelayAgent();
}
