package bfddCommonDefs

import ()

const (
	PUB_SOCKET_ADDR = "ipc:///tmp/bfdd.ipc"
)

type BfdSessionOwner int32

// Owner
const (
	USER              BfdSessionOwner = 1
	BGP               BfdSessionOwner = 2
	OSPF              BfdSessionOwner = 3
	MAX_NUM_PROTOCOLS BfdSessionOwner = 4
)

type BfdSessionOperation int32

// Operation
const (
	CREATE    BfdSessionOperation = 1
	DELETE    BfdSessionOperation = 2
	ADMINUP   BfdSessionOperation = 3
	ADMINDOWN BfdSessionOperation = 4
)

type BfdSessionConfig struct {
	DestIp    string
	PerLink   bool
	Owner     string
	Operation string
}

func ConvertBfdSessionOwnerStrToVal(owner string) BfdSessionOwner {
	var ownerVal BfdSessionOwner
	switch owner {
	case "user":
		ownerVal = USER
	case "bgp":
		ownerVal = BGP
	case "ospf":
		ownerVal = OSPF
	}
	return ownerVal
}

func ConvertBfdSessionOwnerValToStr(owner BfdSessionOwner) string {
	var ownerStr string
	switch owner {
	case USER:
		ownerStr = "user"
	case BGP:
		ownerStr = "bgp"
	case OSPF:
		ownerStr = "ospf"
	}
	return ownerStr
}

func ConvertBfdSessionOperationStrToVal(oper string) BfdSessionOperation {
	var operVal BfdSessionOperation
	switch oper {
	case "create":
		operVal = CREATE
	case "delete":
		operVal = DELETE
	case "up":
		operVal = ADMINUP
	case "down":
		operVal = ADMINDOWN
	}
	return operVal
}

func ConvertBfdSessionOperationValToStr(oper BfdSessionOperation) string {
	var operStr string
	switch oper {
	case CREATE:
		operStr = "create"
	case DELETE:
		operStr = "delete"
	case ADMINUP:
		operStr = "up"
	case ADMINDOWN:
		operStr = "down"
	}
	return operStr
}
