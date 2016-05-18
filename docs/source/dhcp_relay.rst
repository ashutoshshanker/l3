.. FlexSwitchL3 documentation master file, created by
   sphinx-quickstart on Mon May 16 11:13:19 2016.
   You can adapt this file completely to your liking, but it should at least
   contain the root `toctree` directive.

DHCP Relay Design and documentation!
========================================

Architecture
========================================



Interfaces
========================================
 Dhcp Relay has following state:
  1. Receive DISCOVER Packet
  2. Relay client Packet to all servers (configured) updating Relay Agent Information in Dhcp Options
  3. Receive OFFER Packet
  4. Send Unicast OFFER to Client (if configured) else Broadcast OFFER Packet
  5. Receive REQUEST Packet
  6. Relay REQUEST Packet to Server
  7. Receive ACK Packet
  8. Relay ACK Packet to Client

Configuration
========================================
 - Global Config to enable/disable Relay Agent across all interfaces
 - Create/Delete Relay Agent per interface
 - Configure Server's for Relay Agent

