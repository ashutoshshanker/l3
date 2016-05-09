# Address Resolution Protocol

### Introduction
The address resolution protocol (arp) is a protocol used by the Internet Protocol (IP) [RFC826], specifically IPv4, to map IP network addresses to the hardware addresses used by a data link protocol. The protocol operates below the network layer as a part of the interface between the OSI network and OSI link layer.


### Architecture
![alt text](https://github.com/SnapRoute/l3/blob/master/arp/docs/ARP.png "Architecture")

### Description

ARP module listens to ASICD notification for L3 interface creation/deletion. It starts Rx/Tx go routines on all L3 interface.

- When it receives any ARP request, ARP cache is updated with IP Address (source IP) to Mac Address (source MAC) in ARP request packet. Linux ARP stack replies to the ARP Request.

- When it receives any ARP reply, ARP cache is updated with IP Address (destination IP) to Mac Address (destination MAC) in the ARP reply packet.

- When it receives any IPv4 packet, ARP cache is updated with IP Address (source IP) to Mac Address (source MAC) in the IPv4 packet if source IP is in local subnet of the switch's L3 interface. And ARP module sends an ARP request packet for the destination IP address.

- When RIB module receives a route, RIB daemon sends ARP daemon a message to resolve IP Address to Mac Address mapping for the nexthop IP Address.

### Interfaces
Configutation Object Name: **ArpConfig**

> - Create ARP Config:

		bool CreateArpConfig(1: ArpConfig config);


>  - Update ARP Config:
	
		bool UpdateArpConfig(1: ArpConfig origconfig, 2: ArpConfig newconfig, 3: list<bool> attrset);


>  - Delete ARP Config: 

		bool DeleteArpConfig(1: ArpConfig config);

State Object Name: **ArpEntryState**

>  - Get the list of ARP Entries (Object Name: ArpEntryState):

		ArpEntryStateGetInfo **GetBulkArpEntryState**(1: int fromIndex, 2: int count);


>  - Get the list of ARP Entry corresponding to given IP Address:

		ArpEntryState **GetArpEntryState**(1: string IpAddr);

Actions:

> - Delete all the ARP entries learnt on given interface name

		bool ExecuteActionArpDeleteByIfName(1: ArpDeleteByIfName config);


> - Delete all the ARP entries learnt on given interface name

		bool ExecuteActionArpDeleteByIPv4Addr(1: ArpDeleteByIPv4Addr config);
