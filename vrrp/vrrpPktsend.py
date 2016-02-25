#!/usr/bin/env python
import sys
from scapy.all import *

pkt=IP(ttl=255)/VRRP(priority=123, version=3,ipcount=1,addrlist=["90.0.1.1"])
pkt.dst="224.0.0.18"
pkt.proto=112 #vrrp protocol
send(pkt)
print pkt.show()
