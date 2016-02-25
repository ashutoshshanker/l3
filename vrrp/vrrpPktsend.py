#!/usr/bin/env python
import sys
from scapy.all import *

pkt=IP()
pkt.dst="224.0.0.18"
pkt.proto=112 #vrrp protocol
print pkt.show()
send(pkt/VRRP())
