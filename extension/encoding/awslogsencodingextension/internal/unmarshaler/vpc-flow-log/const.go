// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package vpcflowlog // import "github.com/open-telemetry/opentelemetry-collector-contrib/extension/encoding/awslogsencodingextension/internal/unmarshaler/vpc-flow-log"

// protocolNames are needed to know the name of the protocol number given by the field
// protocol in a flow log record.
// See https://www.iana.org/assignments/protocol-numbers/protocol-numbers.xhtml.
var protocolNames = [257]string{
	"hopopt",
	"icmp",
	"igmp",
	"ggp",
	"ipv4",
	"st",
	"tcp",
	"cbt",
	"egp",
	"igp",
	"bbn-rcc-mon",
	"nvp-ii",
	"pup",
	"argus",
	"emcon",
	"xnet",
	"chaos",
	"udp",
	"mux",
	"dcn-meas",
	"hmp",
	"prm",
	"xns-idp",
	"trunk-1",
	"trunk-2",
	"leaf-1",
	"leaf-2",
	"rdp",
	"irtp",
	"iso-tp4",
	"netblt",
	"mfe-nsp",
	"merit-inp",
	"dccp",
	"3pc",
	"idpr",
	"xtp",
	"ddp",
	"idpr-cmtp",
	"tp++",
	"il",
	"ipv6",
	"sdrp",
	"ipv6-route",
	"ipv6-frag",
	"idrp",
	"rsvp",
	"gre",
	"dsr",
	"bna",
	"esp",
	"ah",
	"i-nlsp",
	"swipe",
	"narp",
	"mobile",
	"tlsp",
	"skip",
	"ipv6-icmp",
	"ipv6-nonxt",
	"ipv6-opts",
	"",
	"cftp",
	"",
	"sat-expak",
	"kryptolan",
	"rvd",
	"ippc",
	"",
	"sat-mon",
	"visa",
	"ipcv",
	"cpnx",
	"cphb",
	"wsn",
	"pvp",
	"br-sat-mon",
	"sun-nd",
	"wb-mon",
	"wb-expak",
	"iso-ip",
	"vmtp",
	"secure-vmtp",
	"vines",
	"ttp",
	"nsfnet-igp",
	"dgp",
	"tcf",
	"eigrp",
	"ospf",
	"sprite-rpc",
	"larp",
	"mtp",
	"ax.25",
	"ipip",
	"micp",
	"scc-sp",
	"etherip",
	"encap",
	"",
	"gmtp",
	"ifmp",
	"pnni",
	"pim",
	"aris",
	"scps",
	"qnx",
	"a/n",
	"ipcomp",
	"snp",
	"compaq-peer",
	"ipx-in-ip",
	"vrrp",
	"pgm",
	"",
	"l2tp",
	"ddx",
	"iatp",
	"stp",
	"srp",
	"uti",
	"smp",
	"sm",
	"ptp",
	"isis over ipv4",
	"fire",
	"crtp",
	"crudp",
	"sscopmce",
	"iplt",
	"sps",
	"pipe",
	"sctp",
	"fc",
	"rsvp-e2e-ignore",
	"mobility header",
	"udplite",
	"mpls-in-ip",
	"manet",
	"hip",
	"shim6",
	"wesp",
	"rohc",
	"ethernet",
	"aggfrag",
	"nsis",
	"nsh",
	// empty between 147-254
	"", "", "", "", "", "", "", "", "", "",
	"", "", "", "", "", "", "", "", "", "",
	"", "", "", "", "", "", "", "", "", "",
	"", "", "", "", "", "", "", "", "", "",
	"", "", "", "", "", "", "", "", "", "",
	"", "", "", "", "", "", "", "", "", "",
	"", "", "", "", "", "", "", "", "", "",
	"", "", "", "", "", "", "", "", "", "",
	"", "", "", "", "", "", "", "", "", "",
	"", "", "", "", "", "", "", "", "", "",
	"", "", "", "", "", "", "", "",
	"reserved",
}
