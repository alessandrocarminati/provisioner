package main
import (
	"fmt"
	"net"
	logbuf "pippo.com/goinit/logbuf"
	"pippo.com/goinit/dhclient"
	"os"
	"github.com/google/gopacket/layers"
	"github.com/vishvananda/netlink"
	"io/ioutil"
	"regexp"
	"strconv"
	syslog "log/syslog"
)


func dhcpFetch(ifname string, terminate chan int, msgs chan string) {

	ifacen, err := netlink.LinkByName(ifname)
	if err != nil {
		msgs <- logbuf.LogSprintf(logbuf.LevelError, "Failed to retrieve interface %s: %s", ifname, err.Error())
	}
	attrs := ifacen.Attrs()
	if attrs.Flags&net.FlagUp == 0 {
		msgs <- logbuf.LogSprintf(logbuf.LevelNotice, "bring %s up", ifname)
		err = netlink.LinkSetUp(ifacen)
		if err != nil {
			msgs <- logbuf.LogSprintf(logbuf.LevelError, "Failed to retrieve interface %s: %s", ifname, err.Error())
		}
	}

	iface, err := net.InterfaceByName(ifname)
	if err != nil {
		msgs <- logbuf.LogSprintf(logbuf.LevelError, "unable to find interface %s: %s", ifname, err.Error())
		os.Exit(1)
	}

	client := dhclient.Client{
		Iface: iface, OnBound: func(lease *dhclient.Lease) {
			msgs <- logbuf.LogSprintf(logbuf.LevelNotice, "Assigned address: %s", lease.FixedAddress)
//			fmt.Println(lease)
////////////////////////////////////////////////////////////////////////////////////////////////////////////
//			fmt.Println("seting ip address")
			ip := net.IPNet{IP: lease.FixedAddress, Mask: lease.Netmask}
			gateway := lease.Router[0]
			link, err := netlink.LinkByName(ifname)
			if err != nil {
				msgs <- logbuf.LogSprintf(logbuf.LevelError, "Failed to retrieve interface:", err)
			}
			addr := &netlink.Addr{IPNet: &ip}
			if err := netlink.AddrAdd(link, addr); err != nil {
				msgs <- logbuf.LogSprintf(logbuf.LevelError, "Failed to set IP address:", err)
			}
//			fmt.Println("seting default route:", lease.Router[0])
			defaultRoute := netlink.Route{
				LinkIndex: link.Attrs().Index,
				Gw:        gateway,
			}
			if err := netlink.RouteAdd(&defaultRoute); err != nil {
				msgs <- logbuf.LogSprintf(logbuf.LevelError, "Failed to add default route:", err)
			}
///////////////////////////////////////////////////////////////////////////////////////////////////////////
			startmsg <- true
		},
	}
	for _, param := range dhclient.DefaultParamsRequestList {
//		msgs <- logbuf.LogSprintf(logbuf.LevelWarning, "Requesting default option %d", param)
		client.AddParamRequest(layers.DHCPOpt(param))
	}

	hostname, _ := os.Hostname()
	client.AddOption(layers.DHCPOptHostname, []byte(hostname))

	client.Start()
	defer client.Stop()
	<- terminate
}

func listdev(msgs chan string) []string{
	ret:=[]string{}

	netDir := "/sys/class/net"
	adapters, err := ioutil.ReadDir(netDir)
	if err != nil {
		msgs <- logbuf.LogSprintf(logbuf.LevelError, "Error reading network interfaces:", err)
		os.Exit(1)
	}

	msgs <- logbuf.LogSprintf(logbuf.LevelInfo, "Network Adapters:")
	for _, adapter := range adapters {
		msgs <- logbuf.LogSprintf(logbuf.LevelInfo, adapter.Name())
		ret = append(ret, adapter.Name())
	}
	return ret
}

func printUtil(str string, currentLevel int) {
	re := regexp.MustCompile(`^<(\d+)>`)

	match := re.FindStringSubmatch(str)
	if len(match) != 2 {
		return
	}

	msgLevel, err := strconv.Atoi(match[1])
	if err != nil {
		return
	}

	if msgLevel <= currentLevel {
		fmt.Println(str[len(match[0]):])
	}
}

func syslogSender(msgs chan string, config map[string] string, logLevel int){
	var s string
	var err error
	var syslogWriter *syslog.Writer

	usesyslog:=false
	_, ok := config["hasif"]
	if ok {
		<- startmsg

		syslogIP, ok := config["pr.syslogIP"]
		if ok {
			usesyslog=true
			syslogWriter, err = syslog.Dial("udp", syslogIP + ":514", syslog.LOG_INFO, "provisioner")
			if err != nil {
				msgs <- logbuf.LogSprintf(logbuf.LevelError, "Failed to connect to syslog server: %v", err)
				usesyslog=false
			}
			defer syslogWriter.Close()
		}
	}
	for {
		s = <- msgs
		if s != "" {
			printUtil(s, logLevel)
			if usesyslog {
				syslogWriter.Info(s)
			}
		}
	}
}

