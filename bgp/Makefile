RM=rm -f
RMFORCE=rm -rf
DESTDIR=$(SR_CODE_BASE)/src/bin
GENERATED_IPC=$(SR_CODE_BASE)/generated/src
IPC_GEN_CMD=thrift 
IPC_SRCS=rpc/bgpd.thrift
SRCS=main.go
OBJS=$(subst .cpp,.o,$(SRCS))
COMP_NAME=bgpd
all:ipc $(COMP_NAME)
ipc:
	 $(IPC_GEN_CMD) -r --gen go -out $(GENERATED_IPC) $(IPC_SRCS)

bgpd: $(OBJS)
	 go build -o $@ $(SRCS)

guard:
ifndef SR_CODE_BASE
	 $(error SR_CODE_BASE is not set)
endif

clean:guard
	 $(RM) $(COMP_NAME) 
	 $(RMFORCE) $(GENERATED_IPC)/$(COMP_NAME)
