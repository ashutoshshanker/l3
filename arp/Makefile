RM=rm -f
RMFORCE=rm -rf
DESTDIR=$(SR_CODE_BASE)/snaproute/src/bin
GENERATED_IPC=$(SR_CODE_BASE)/generated/src
IPC_GEN_CMD=thrift 
SRCS=arphandler.go  arpmain.go arpapis.go
IPC_SRCS=arpd.thrift
COMP_NAME=arpd
all:ipc exe
ipc:
	 $(IPC_GEN_CMD) --gen go -out $(GENERATED_IPC) $(IPC_SRCS)

exe: $(SRCS)
	 go build -o $(DESTDIR)/$(COMP_NAME) $(SRCS)

guard:
ifndef SR_CODE_BASE
	 $(error SR_CODE_BASE is not set)
endif

install:
	@echo "ARP has no files to install"
clean:guard
	 $(RM) $(DESTDIR)/$(COMP_NAME) 
	 $(RMFORCE) $(GENERATED_IPC)/$(COMP_NAME)
