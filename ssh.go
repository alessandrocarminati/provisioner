package main

import (
	"log"
	"sync"
	"golang.org/x/crypto/ssh"
	"io"
	"os"
	"net"
	"fmt"
//	"crypto/md5"
	"encoding/hex"
)
type DefAuth struct {
	service		string
	name		string
	token		string
	state		bool
}


var GenAuth []DefAuth

var sshChannels = make(map[string] *ssh.Channel)

var sshConns map[string]int

func ssh_init() {
	sshConns = make(map[string]int)
	sshConns["montor"]=0
	sshConns["tunnel"]=0
}


func checkPerm(token string, service string) bool {
	for _, i := range GenAuth {
		if (token == i.token) && (service == i.service) {
			return i.state
		}
	}
	return false
}

func SSHHandler(sshcfg SSHCFG, desc string, sshIn chan<- byte, sshOut <-chan byte, def_aut bool) {
	debugPrint(log.Printf, levelDebug, "request descr=%s connected=%d", desc, sshConns[desc])
	authorizedKeysBytes, err := os.ReadFile(sshcfg.Authorized_keys)
	if err != nil {
		debugPrint(log.Printf, levelPanic, "Failed to load authorized_keys, err: %s", err.Error())
	}

	authorizedKeysMap := map[string]bool{}
	for len(authorizedKeysBytes) > 0 {
		pubKey, comment, _, rest, err := ssh.ParseAuthorizedKey(authorizedKeysBytes)
		if err != nil {
			debugPrint(log.Printf, levelPanic, "Only one line per user, no extra lines: %s", err.Error())
		}
		debugPrint(log.Printf, levelDebug, "add key for %s -> %s", comment, hex.EncodeToString(pubKey.Marshal()))
		GenAuth = append(GenAuth, DefAuth{
			service: desc,
			name: comment,
			token: hex.EncodeToString(pubKey.Marshal()),
			state: def_aut,
		})
		authorizedKeysMap[string(pubKey.Marshal())] = true
		authorizedKeysBytes = rest
	}
	config := &ssh.ServerConfig{
		PublicKeyCallback: func (c ssh.ConnMetadata, pubKey ssh.PublicKey) (*ssh.Permissions, error) {
			if sshConns[desc]<1 {
				sshConns[desc]++
				if authorizedKeysMap[string(pubKey.Marshal())] {
					debugPrint(log.Printf, levelDebug, "Authorized user attempt check permissions")
					if checkPerm(hex.EncodeToString(pubKey.Marshal()), desc) {
						return &ssh.Permissions{
							Extensions: map[string]string{
								"pubkey-fp": ssh.FingerprintSHA256(pubKey),
							},
						}, nil
					} else {
					return nil, fmt.Errorf("unauthorized user")
					}
				}
				return nil, fmt.Errorf("unknown public key for %q", c.User())
			} else {
				return nil, fmt.Errorf("Too many users")
			}
		},
	}

	privateBytes, err := os.ReadFile(sshcfg.IdentitFn)
	if err != nil {
		debugPrint(log.Printf, levelPanic, "Failed to load private key: %s", err.Error())
	}
	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		debugPrint(log.Printf, levelPanic, "Failed to parse private key: %s", err.Error())
	}
	config.AddHostKey(private)


	listener, err := net.Listen("tcp", ":"+sshcfg.Port)
	if err != nil {
		debugPrint(log.Printf, levelPanic, "failed to listen for ssh: %s", err.Error())
	}
	defer listener.Close()

	debugPrint(log.Printf, levelWarning, "Starting %s SSH server on port %s\n", desc, sshcfg.Port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			debugPrint(log.Printf, levelPanic, "failed to accept incoming connection: %s", err.Error())
		}

		go handleSSHConnection(conn, config, sshIn, sshOut, desc)
	}
}

func checkBrokenConnections(conn ssh.Conn, desc string){
	conn.Wait()
	sshConns[desc]--
	debugPrint(log.Printf, levelInfo, "Closing broken connection")
	conn.Close()
}

func handleSSHConnection(conn net.Conn, config *ssh.ServerConfig, sshIn chan<- byte, sshOut <-chan byte, desc string) {
	defer conn.Close()

	debugPrint(log.Printf, levelDebug, "request descr=%s, connected=%d", desc, sshConns[desc])
	sshConn, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		debugPrint(log.Printf, levelError, "failed to establish SSH connection: %s", err.Error())
		return
	}
	defer sshConn.Close()

	go checkBrokenConnections(sshConn, desc)
	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		go handleSSHChannel(newChannel, sshIn, sshOut, desc)
	}
}

func handleSSHChannel(newChannel ssh.NewChannel, sshIn chan<- byte, sshOut <-chan byte, desc string) {
	if newChannel.ChannelType() != "session" {
		newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
		return
	}

	channel, _, err := newChannel.Accept()
	if err != nil {
		debugPrint(log.Printf, levelError, "failed to accept channel: %s", err.Error())
		return
	}
	sshChannels[desc]=&channel
	defer channel.Close()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			data := <-sshOut
			_, err := channel.Write([]byte{data})
			if err != nil {
				if err != io.EOF {
					debugPrint(log.Printf, levelError, "Error writing to SSH channel: %s", err.Error())
				}
				return
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			n, err := channel.Read(buf)
			if err != nil {
				if err != io.EOF {
					debugPrint(log.Printf, levelError, "Error reading from SSH channel: %s", err.Error())
				}
				return
			}
			if n>0 {
				debugPrint(log.Printf, levelDebug, "read %d bytes = '%s'", len(buf), string(buf[:n]))
				for i:=0;i<n;i++ {
					sshIn <- buf[i]
				}
			}
		}
	}()

	wg.Wait()
}
