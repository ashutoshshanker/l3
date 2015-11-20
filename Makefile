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

install:
	@echo "All files that need to be copied would go here"

