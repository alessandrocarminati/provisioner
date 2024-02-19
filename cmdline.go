package main
import (
	"flag"
	"bytes"
)
type Cmdline struct {
	ConfigFN string
	Key      string
	Help     bool
	Enc      bool
	GenKeys  bool
}

func parseCMDline() Cmdline {
	configFNPtr := flag.String("config", "config.json", "Config file name")
	keyPtr := flag.String("key", "", "Key file name. if not given, the config isassumed plaintext")
	enccfgPtr := flag.Bool("enc", false, "interpret \"config\" as input file and \"key\" as private key file and outputs in config.rsa")
	genkeysPtr := flag.Bool("genkeys", false, "generates two new keypairs")

	helpPtr := flag.Bool("help", false, "Show help")

	flag.Parse()

	config := Cmdline{
		ConfigFN: *configFNPtr,
		Key:      *keyPtr,
		Help:     *helpPtr,
		Enc:      *enccfgPtr,
		GenKeys:  *genkeysPtr,
	}
	return config
}
func helpText() string {
	var buf bytes.Buffer
	flag.CommandLine.SetOutput(&buf)
	flag.Usage()
	return buf.String()
}
