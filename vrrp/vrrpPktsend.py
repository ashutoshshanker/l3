#!/usr/bin/env python
import sys
from scapy.all import *

pkt=IP()
pkt.dst="224.0.0.18"
#pkt.src="10.1.10.244"
print pkt.show()
send(pkt/VRRP())
