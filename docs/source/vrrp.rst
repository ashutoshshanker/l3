.. FlexSwitchL3 documentation master file, created by
   sphinx-quickstart on Mon May 16 11:13:19 2016.
   You can adapt this file completely to your liking, but it should at least
   contain the root `toctree` directive.

VRRP Design and documentation!
========================================

Introduction
========================================
This module implement Virtual Router Redundancy Protocol RFC 5798

Architecture
========================================

Interfaces
========================================
 - Create/Delete Virtual Router
 - Change timers for VRRP packet, for e.g: Advertisement Timer

Configuration
========================================
 - VRRP configuration is based of https://tools.ietf.org/html/rfc5798#section-5.2
 - Unless specified each instance of Virtual Router will use the default values specified in the RFC

