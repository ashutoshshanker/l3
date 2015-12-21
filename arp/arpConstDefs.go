package main

const (
        ARP_ERR_NOT_FOUND = iota
        ARP_PARSE_ADDR_FAIL
        ARP_ERR_REQ_FAIL
        ARP_ERR_RESP_FAIL
        ARP_ERR_ADD_FAIL
        ARP_REQ_SUCCESS
        ARP_ERR_LAST
)

const (
        ARP_ADD_ENTRY = iota
        ARP_DEL_ENTRY
        ARP_UPDATE_ENTRY
)

