#!/usr/bin/env python
import sys
from scapy.all import *

#pkt=Ether(src="00:24:e8:7a:2a:7a",dst="e0:ac:cb:82:ff:7c")/IP(ttl=255, src="10.1.10.252",dst="224.0.0.18", proto=112)/VRRP(priority=123, version=3,ipcount=1,addrlist=["90.0.1.1"])
pkt=IP(ttl=255, src="10.1.10.252",dst="224.0.0.18", proto=112)/VRRP(priority=123, version=3,ipcount=1,addrlist=["90.0.1.1"])
print pkt.show()
send(pkt)
