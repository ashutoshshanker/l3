.. FlexSwitchL3 documentation master file, created by
   sphinx-quickstart on Mon May 16 11:13:19 2016.
   You can adapt this file completely to your liking, but it should at least
   contain the root `toctree` directive.

OSPF Design and documentation!
========================================

Introduction
========================================
This module implements Open Shortest Path First.v2
RFC 2328 

Architecture
========================================
.. image:: imagesospf_architecture.png

Modules
========================================
OSPF has below components
1) Interface FSM - 
Handles per interfce OSPF config events,send hello packets, DR/BDR election .
 It signals Neighbor FSM whenenever new neighbor is detected. 

2) Neighbor FSM -
This component implements 
RX packets such as DB description , LSA Update/Request/Ack.
Takes care of flooding. 
Inform LSDB for different events such as neighbor full, install LSA.

3) LSDB -
LSA database. Stores 5 types of LSAs.
Trigger SPF when new LSA is installed . 
Generate router/networks LSAs for neighbor full event.
Generate summary LSA if ABR.
Generate AS external if ASBR.
Implements LSA ageing.
Inform Neighbor FSM for flooding  when new LSA is installed.

4)SPF - 
Takes care of shortest path calculation and install routes.
Signal LSDB to generate summary LSA when ABR.

5) RIBd listener -
Listens to RIBd updates when OSPF policy is configured. 
When router is acting as ASBR - RIbd listener will receive route updates as per the 
policy statement.
It signals LSDB for AS external generation when router is configured as ASBR.

Configuration
========================================
Current OSPF configuration is as per OSPF-MIB.yang file 


This is a implementation of Routing Information Base (RIB) in Go.
Summary of functionality implemented by this module is as follows:

1. Handle all network route based configuration (route create, route delete, route update) from either users or other applications (e.g., BGP, OSPF)

2. Handle all routing policy based configuration :
   a. policy conditions create/delete/updates
   b. policy statements create/delete/updates
   c. policy definitions create/delete/updates

3. Implement policy engine 
   a. Based on the policy objects configured and applied on the device, the policy engine filter will match on the conditions provisioned and implement actions based on the application location. For instance, the policy engine filter may result in redistributing certain (route type based/ network prefix based) routes into other applications (BGP,OSPF, etc.,)
4. Responsible for calling ASICd thrift APIs to program the routes in the FIB.

