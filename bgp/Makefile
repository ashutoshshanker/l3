RM=rm -f
RMFORCE=rm -rf
GENERATED_IPC=$(SR_CODE_BASE)/generated/src
IPC_GEN_CMD=thrift 
IPC_SRCS=bgpd.thrift
all:ipc
ipc:
	 $(IPC_GEN_CMD) -r --gen go -out $(GENERATED_IPC) $(IPC_SRCS)
