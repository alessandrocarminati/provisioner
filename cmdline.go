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
}

func parseCMDline() Cmdline {
	configFNPtr := flag.String("config", "config.json", "Config file name")
	keyPtr := flag.String("key", "", "Key file name. if not given, the config isassumed plaintext")
	enccfgPtr := flag.Bool("enc", false, "interpret \"config\" as input file and \"key\" as private key file and outputs in config.rsa")
	genkeysPtr := flag.Bool("genkeys", false, "generates two new keypairs")
	calFetc1stPtr := flag.Bool("calfetch", false, "fetch an element from calendar. Useful for 1st oauth authorization")
	VerRq := flag.Bool("version", false, "Returns the version string")
	VerJ := flag.Bool("verj", false, "Returns the version json")

	helpPtr := flag.Bool("help", false, "Show help")

	flag.Parse()

	config := Cmdline{
		ConfigFN: *configFNPtr,
		Key:      *keyPtr,
		Help:     *helpPtr,
		VerRq:    *VerRq,
		VerJ:    *VerJ,
		Enc:      *enccfgPtr,
		GenKeys:  *genkeysPtr,
		CalFetch: *calFetc1stPtr,
	}
	return config
}
func helpText() string {
	var buf bytes.Buffer
	flag.CommandLine.SetOutput(&buf)
	flag.Usage()
	return buf.String()
}
