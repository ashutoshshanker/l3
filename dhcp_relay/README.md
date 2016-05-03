# Dynamic Host Configuration Protocol Relay

### Introduction
This module implements Dynamic Host Configuration Protocol Relay Agent.

### Architecture


                               Client
                                  |
                                  |
                                  |
                                  V
                          +---------------+
                          |               |
             +----------->|  Relay Agent  |<----------+
             |            |               |           |
             |            +---------------+           |
             |           /                 \          |
             |          /                   \         |
             |         /                     \        |
             +----- Server                 Server-----+

### Interfaces
 Dhcp Relay has following state:
  1. Receive DISCOVER Packet
  2. Relay client Packet to all servers (configured) updating Relay Agent Information in Dhcp Options
  3. Receive OFFER Packet
  4. Send Unicast OFFER to Client (if configured) else Broadcast OFFER Packet
  5. Receive REQUEST Packet
  6. Relay REQUEST Packet to Server
  7. Receive ACK Packet
  8. Relay ACK Packet to Client

### Configuration
 - Global Config to enable/disable Relay Agent across all interfaces
 - Create/Delete Relay Agent per interface
 - Configure Server's for Relay Agent
