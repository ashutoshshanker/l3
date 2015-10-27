RM=rm -f
RMFORCE=rm -rf
GENERATED_IPC=$(SR_CODE_BASE)/generated/src
IPC_GEN_CMD=thrift 
SRCS=ribhandler.go  ribmain.go
IPC_SRCS=ribd.thrift
OBJS=$(subst .cpp,.o,$(SRCS))
COMP_NAME=ribd
all:ipc $(COMP_NAME)
ipc:
	 $(IPC_GEN_CMD) --gen go -out $(GENERATED_IPC) $(IPC_SRCS)

ribd: $(OBJS)
	 go build -o $@ $(SRCS)

guard:
ifndef SR_CODE_BASE
	 $(error SR_CODE_BASE is not set)
endif

clean:guard
	 $(RM) $(COMP_NAME) 
	 $(RMFORCE) $(GENERATED_IPC)/$(COMP_NAME)
