RM=rm -f
RMFORCE=rm -rf
DESTDIR=$(SR_CODE_BASE)/snaproute/src/bin
GENERATED_IPC=$(SR_CODE_BASE)/generated/src
IPC_GEN_CMD=thrift 
SRCS=ribhandler.go  ribmain.go
IPC_SRCS=ribd.thrift
COMP_NAME=ribd
all:ipc exe
ipc:
	 $(IPC_GEN_CMD) --gen go -out $(GENERATED_IPC) $(IPC_SRCS)

exe: $(SRCS)
	 go build -o $(DESTDIR)/$(COMP_NAME) $(SRCS)

guard:
ifndef SR_CODE_BASE
	 $(error SR_CODE_BASE is not set)
endif

clean:guard
	 $(RM) $(COMP_NAME) 
	 $(RMFORCE) $(GENERATED_IPC)/$(COMP_NAME)
