package main
import (
	"flag"
	"bytes"
)
type Cmdline struct {
	ConfigFN string
	Key      string
	Help     bool
	VerRq    bool
	VerJ     bool
	Enc      bool
	GenKeys  bool
	CalFetch bool
	DebLev   int
	Dacl     string
}

func parseCMDline() Cmdline {
	configFNPtr := flag.String("config", "config.json", "Config file name")
	keyPtr := flag.String("key", "", "Key file name. if not given, the config isassumed plaintext")
	enccfgPtr := flag.Bool("enc", false, "interpret \"config\" as input file and \"key\" as private key file and outputs in config.rsa")
	genkeysPtr := flag.Bool("genkeys", false, "generates two new keypairs")
	calFetc1stPtr := flag.Bool("calfetch", false, "fetch an element from calendar. Useful for 1st oauth authorization")
	VerRq := flag.Bool("version", false, "Returns the version string")
	VerJ := flag.Bool("verj", false, "Returns the version json")
	Dl := flag.Int("debug", 2, "stes the message level: 0: panics, 1: Errors, 2: Warnings - default, 3: Notices, 4:Infos, 5: Debugs")
	Dacl := flag.String("dacl", "All", "Specify the list of functions to watch, filtering out all the rest. Defaults to all")

	helpPtr := flag.Bool("help", false, "Show help")

	flag.Parse()

	config := Cmdline{
		ConfigFN: *configFNPtr,
		Key:      *keyPtr,
		Help:     *helpPtr,
		VerRq:    *VerRq,
		VerJ:     *VerJ,
		Enc:      *enccfgPtr,
		GenKeys:  *genkeysPtr,
		CalFetch: *calFetc1stPtr,
		DebLev:   *Dl,
		Dacl:     *Dacl,
	}
	return config
}
func helpText() string {
	var buf bytes.Buffer
	flag.CommandLine.SetOutput(&buf)
	flag.Usage()
	return buf.String()
}
