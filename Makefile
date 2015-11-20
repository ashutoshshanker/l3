COMPS=arp\
		bgp\
		rib

IPCS=arp\
	  bgp\
	  rib
all: ipc exe 

exe: $(COMPS)
	 $(foreach f,$^, make -C $(f) exe;)

ipc: $(IPCS)
	 $(foreach f,$^, make -C $(f) ipc;)
