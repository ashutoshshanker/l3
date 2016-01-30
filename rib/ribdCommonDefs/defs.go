package ribdCommonDefs
import "ribd"

const (
      CONNECTED  = 0
      STATIC     = 1
      OSPF       = 89
      BGP        = 8
	  PolicyConditionTypePrefixMatch = 0
	  PolicyConditionTypeProtocolMatch = 1
	  PolicyActionTypeRouteDisposition = 0
	  PolicyActionTypeRouteRedistribute = 1
	  PUB_SOCKET_ADDR = "ipc:///tmp/ribd.ipc"	
	  PUB_SOCKET_BGPD_ADDR = "ipc:///tmp/ribd_bgpd.ipc"
	  NOTIFY_ROUTE_CREATED = 1
	  NOTIFY_ROUTE_DELETED = 2
	  NOTIFY_ROUTE_INVALIDATED = 3
	  DEFAULT_NOTIFICATION_SIZE = 128
	  PolicyPath_Import = 1
	  PolicyPath_Export = 2
	  RoutePolicyStateChangetoValid=1
	  RoutePolicyStateChangetoInValid = 2
	  RoutePolicyStateChangeNoChange=3
)
//enumerations
var (
	AttributeComparisonPtAttributeGe = 0
	AttributeComparisonPtAttributeLe = 1
	AttributeComparisonAttributeLe = 2
	AttributeComparisonAttributeGe = 3
	AttributeComparisonPtypesAttributeEq = 4
	AttributeComparisonPtypesAttributeGe = 5
	AttributeComparisonPtAttributeEq = 6
	AttributeComparisonAttributeEq = 7
	AttributeComparisonPtypesAttributeLe = 8
	InetIpVersionUnknown = 0
	InetIpVersionIpv4 = 1
	InetIpVersionIpv6 = 2
	PtMatchSetOptionsTypeINVERT = 0
	PtMatchSetOptionsTypeALL = 1
	PtMatchSetOptionsTypeANY = 2
	PtypesMatchSetOptionsTypeINVERT = 0
	PtypesMatchSetOptionsTypeALL = 1
	PtypesMatchSetOptionsTypeANY = 2
	RpolDefaultPolicyTypeACCEPTROUTE = 0
	RpolDefaultPolicyTypeREJECTROUTE = 1
	PtypesAttributeComparisonAttributeLe = 0
	PtypesAttributeComparisonAttributeGe = 1
	PtypesAttributeComparisonPtypesAttributeEq = 2
	PtypesAttributeComparisonPtypesAttributeGe = 3
	PtypesAttributeComparisonAttributeEq = 4
	PtypesAttributeComparisonPtypesAttributeLe = 5
	MatchSetOptionsTypeINVERT = 0
	MatchSetOptionsTypeALL = 1
	MatchSetOptionsTypeANY = 2
	PtypesInstallProtocolTypePtypesSTATIC = 0
	PtypesInstallProtocolTypePtypesBGP = 1
	PtypesInstallProtocolTypeISIS = 2
	PtypesInstallProtocolTypeBGP = 3
	PtypesInstallProtocolTypePtypesOSPF3 = 4
	PtypesInstallProtocolTypePtypesOSPF = 5
	PtypesInstallProtocolTypePtypesISIS = 6
	PtypesInstallProtocolTypePtypesLOCALAGGREGATE = 7
	PtypesInstallProtocolTypeDIRECTLYCONNECTED = 8
	PtypesInstallProtocolTypeSTATIC = 9
	PtypesInstallProtocolTypePtypesDIRECTLYCONNECTED = 10
	PtypesInstallProtocolTypeLOCALAGGREGATE = 11
	PtypesInstallProtocolTypeOSPF = 12
	PtypesInstallProtocolTypeOSPF3 = 13
	MatchSetOptionsRestrictedTypeINVERT = 0
	MatchSetOptionsRestrictedTypeANY = 1
	PtAttributeComparisonPtAttributeGe = 0
	PtAttributeComparisonPtAttributeLe = 1
	PtAttributeComparisonAttributeLe = 2
	PtAttributeComparisonAttributeGe = 3
	PtAttributeComparisonPtAttributeEq = 4
	PtAttributeComparisonAttributeEq = 5
	IpVersionUnknown = 0
	IpVersionIpv4 = 1
	IpVersionIpv6 = 2
	DefaultPolicyTypeACCEPTROUTE = 0
	DefaultPolicyTypeREJECTROUTE = 1
	InstallProtocolTypePtypesSTATIC = 0
	InstallProtocolTypeBGP = 1
	InstallProtocolTypePtBGP = 2
	InstallProtocolTypeISIS = 3
	InstallProtocolTypePtOSPF = 4
	InstallProtocolTypePtDIRECTLYCONNECTED = 5
	InstallProtocolTypePtISIS = 6
	InstallProtocolTypePtypesOSPF3 = 7
	InstallProtocolTypePtypesOSPF = 8
	InstallProtocolTypePtypesBGP = 9
	InstallProtocolTypePtypesISIS = 10
	InstallProtocolTypePtypesLOCALAGGREGATE = 11
	InstallProtocolTypeDIRECTLYCONNECTED = 12
	InstallProtocolTypeSTATIC = 13
	InstallProtocolTypePtypesDIRECTLYCONNECTED = 14
	InstallProtocolTypePtSTATIC = 15
	InstallProtocolTypeLOCALAGGREGATE = 16
	InstallProtocolTypeOSPF = 17
	InstallProtocolTypePtOSPF3 = 18
	InstallProtocolTypeOSPF3 = 19
	InstallProtocolTypePtLOCALAGGREGATE = 20
	PtMatchSetOptionsRestrictedTypeINVERT = 0
	PtMatchSetOptionsRestrictedTypeANY = 1
	PtypesMatchSetOptionsRestrictedTypeINVERT = 0
	PtypesMatchSetOptionsRestrictedTypeANY = 1
	PtInstallProtocolTypePtBGP = 0
	PtInstallProtocolTypeISIS = 1
	PtInstallProtocolTypePtOSPF = 2
	PtInstallProtocolTypePtDIRECTLYCONNECTED = 3
	PtInstallProtocolTypePtISIS = 4
	PtInstallProtocolTypeLOCALAGGREGATE = 5
	PtInstallProtocolTypeBGP = 6
	PtInstallProtocolTypeDIRECTLYCONNECTED = 7
	PtInstallProtocolTypeSTATIC = 8
	PtInstallProtocolTypePtSTATIC = 9
	PtInstallProtocolTypeOSPF = 10
	PtInstallProtocolTypePtOSPF3 = 11
	PtInstallProtocolTypeOSPF3 = 12
	PtInstallProtocolTypePtLOCALAGGREGATE = 13
)

type RibdNotifyMsg struct {
    MsgType uint16
    MsgBuf []byte
}

type RoutelistInfo struct {
    RouteInfo ribd.Routes
}
