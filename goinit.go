package main

import (
	"fmt"
	"golang.org/x/sys/unix"
	"os/signal"
	"bufio"
        "syscall"
	"os"
	"strings"
	"io/ioutil"
	"net"
	"pippo.com/goinit/dhclient"
	logbuf "pippo.com/goinit/logbuf"
	"github.com/google/gopacket/layers"
	"github.com/vishvananda/netlink"
	syslog "log/syslog"
)
var startmsg chan bool

func dhcpFetch(ifname string, terminate chan int, msgs chan string) {

	ifacen, err := netlink.LinkByName(ifname)
	if err != nil {
		msgs <- logbuf.LogSprintf(logbuf.LevelWarning, "Failed to retrieve interface %s: %s", ifname, err.Error())
	}
	attrs := ifacen.Attrs()
	if attrs.Flags&net.FlagUp == 0 {
		msgs <- logbuf.LogSprintf(logbuf.LevelWarning, "bring %s up", ifname)
		err = netlink.LinkSetUp(ifacen)
		if err != nil {
			msgs <- logbuf.LogSprintf(logbuf.LevelWarning, "Failed to retrieve interface %s: %s", ifname, err.Error())
		}
	}

	iface, err := net.InterfaceByName(ifname)
	if err != nil {
		msgs <- logbuf.LogSprintf(logbuf.LevelWarning, "unable to find interface %s: %s", ifname, err.Error())
		os.Exit(1)
	}

	client := dhclient.Client{
		Iface: iface, OnBound: func(lease *dhclient.Lease) {
			msgs <- logbuf.LogSprintf(logbuf.LevelWarning, "Assigned address: %s", lease.FixedAddress)
//			fmt.Println(lease)
////////////////////////////////////////////////////////////////////////////////////////////////////////////
//			fmt.Println("seting ip address")
			ip := net.IPNet{IP: lease.FixedAddress, Mask: lease.Netmask}
			gateway := lease.Router[0]
			link, err := netlink.LinkByName(ifname)
			if err != nil {
				fmt.Println("Failed to retrieve interface:", err)
			}
			addr := &netlink.Addr{IPNet: &ip}
			if err := netlink.AddrAdd(link, addr); err != nil {
				fmt.Println("Failed to set IP address:", err)
			}
//			fmt.Println("seting default route:", lease.Router[0])
			defaultRoute := netlink.Route{
				LinkIndex: link.Attrs().Index,
				Gw:        gateway,
			}
			if err := netlink.RouteAdd(&defaultRoute); err != nil {
				fmt.Println("Failed to add default route:", err)
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

func listdev(msgs chan string) {
    netDir := "/sys/class/net"
    adapters, err := ioutil.ReadDir(netDir)
    if err != nil {
        msgs <- logbuf.LogSprintf(logbuf.LevelWarning, "Error reading network interfaces:", err)
        os.Exit(1)
    }

    msgs <- logbuf.LogSprintf(logbuf.LevelWarning, "Network Adapters:")
    for _, adapter := range adapters {
            msgs <- logbuf.LogSprintf(logbuf.LevelWarning, adapter.Name())
    }
}

func isSymbolicLink(path string, msgs chan string) bool {
    fileInfo, err := os.Lstat(path)
    if err != nil {
        return false
    }
    return fileInfo.Mode()&os.ModeSymlink == os.ModeSymlink
}

func mount(device, target string, msgs chan string){
	msgs <- logbuf.LogSprintf(logbuf.LevelWarning, "Mount %s", device)

	if err := os.Mkdir(target, os.ModePerm); err != nil {
		msgs <- logbuf.LogSprintf(logbuf.LevelWarning, "Error creating procfs: %s", err.Error())
		os.Exit(0xfff2) 
		}
	if err := unix.Mount(device, target, device, 0, ""); err != nil {
		msgs <- logbuf.LogSprintf(logbuf.LevelWarning, "Error mounting %s: %s", device, err.Error())
		os.Exit(0xfff3) 
	}
	msgs <- logbuf.LogSprintf(logbuf.LevelWarning, "%s mounted successfully at %s", device, target)
}


func syslogSender(msgs chan string, config map[string] string){
	var s string
	var err error
	var syslogWriter *syslog.Writer
	<- startmsg

	syslogIP, ok := config["pr.syslogIP"]
	usesyslog:=false
	if ok {
		usesyslog=true
		syslogWriter, err = syslog.Dial("udp", syslogIP + ":514", syslog.LOG_INFO, "provisioner")
		if err != nil {
			fmt.Printf("Failed to connect to syslog server: %v", err)
			usesyslog=false
		}
		defer syslogWriter.Close()
	}
	for {
		s = <- msgs
		if s != "" {
			fmt.Println(s)
			if usesyslog {
				syslogWriter.Info(s)
			}
		}
	}
}

func fetchConfig(s string) map[string] string{

	res := make(map[string] string, 50)
	tmp := strings.Split(s, " ")
	for _, item := range tmp {
		if strings.HasPrefix(item, "pr.") {
			tmp2 := strings.Split(item, "=")
			res[tmp2[0]]=tmp2[1]
		}
	}
	return res
}

func main() {
	var config map[string] string

	msgs := make(chan string, 300)
	startmsg = make(chan bool ,1)



	msgs <- logbuf.LogSprintf(logbuf.LevelWarning, "Starting Init")
	msgs <- logbuf.LogSprintf(logbuf.LevelWarning, "Checking pid")
        if os.Getpid() != 1 {
                msgs <- logbuf.LogSprintf(logbuf.LevelWarning, "This is not pid 1")
        }
	msgs <- logbuf.LogSprintf(logbuf.LevelWarning, "Mounting file systems")
	mount("proc", "/proc", msgs)
	mount("sysfs", "/sys", msgs)

	file, err := os.Open("/proc/cmdline")
	if err != nil {
		os.Exit(0xfff2)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		s:=scanner.Text()
		msgs <- logbuf.LogSprintf(logbuf.LevelWarning, s)
		config = fetchConfig(s)
	}

	for key, value := range config {
		msgs <- logbuf.LogSprintf(logbuf.LevelWarning, "Key: %s Value: %s", key, value)
	}
	if err := scanner.Err(); err != nil {
		os.Exit(0xfff1)
	}

	c:= make(chan  int)
	ifName, ok := config["pr.ifname"]
	if ok {
		go dhcpFetch(ifName, c, msgs)
	}
	syslogSender(msgs, config)

	listdev(msgs)

	sigs := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(sigs, syscall.SIGINT)

	go func() {
		sig := <-sigs
		fmt.Println()
		msgs <- logbuf.LogSprintf(logbuf.LevelWarning, "Received %s, exiting...", sig)
		done <- true
	}()


	fmt.Println("Press Ctrl-C to exit...")
	<-done
	fmt.Println("Done")
	os.Exit(0xfff0)
}
