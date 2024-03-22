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

func checkPerm(token string, service string) bool {
	for _, i := range GenAuth {
		if (token == i.token) && (service == i.service) {
			return i.state
		}
	}
	return false
}

func SSHHandler(sshcfg SSHCFG, desc string, sshIn chan<- byte, sshOut <-chan byte, def_aut bool) {
	authorizedKeysBytes, err := os.ReadFile(sshcfg.Authorized_keys)
	if err != nil {
		log.Fatalf("Failed to load authorized_keys, err: %v", err)
	}

	authorizedKeysMap := map[string]bool{}
	for len(authorizedKeysBytes) > 0 {
		pubKey, comment, _, rest, err := ssh.ParseAuthorizedKey(authorizedKeysBytes)
		if err != nil {
			log.Println("Only one line per user, no extra lines")
			log.Fatal(err)
		}
//		log.Printf("add key for %s -> %s", comment, hex.EncodeToString(pubKey.Marshal()))
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
		PublicKeyCallback: func(c ssh.ConnMetadata, pubKey ssh.PublicKey) (*ssh.Permissions, error) {
			if authorizedKeysMap[string(pubKey.Marshal())] {
//				log.Printf("Authorized user attempt check permissions")
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
		},
	}

	privateBytes, err := os.ReadFile(sshcfg.IdentitFn)
	if err != nil {
		log.Fatal("Failed to load private key: ", err)
	}
	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		log.Fatal("Failed to parse private key: ", err)
	}
	config.AddHostKey(private)


	listener, err := net.Listen("tcp", ":"+sshcfg.Port)
	if err != nil {
		log.Fatal("failed to listen for ssh:", err)
	}
	defer listener.Close()

	log.Printf("Starting %s SSH server on port %s\n", desc, sshcfg.Port)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal("failed to accept incoming connection:", err)
		}

		go handleSSHConnection(conn, config, sshIn, sshOut, desc)
	}
}

func handleSSHConnection(conn net.Conn, config *ssh.ServerConfig, sshIn chan<- byte, sshOut <-chan byte, desc string) {
	defer conn.Close()

	sshConn, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		log.Println("failed to establish SSH connection:", err)
		return
	}
	defer sshConn.Close()

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
		log.Println("failed to accept channel:", err)
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
					log.Println("Error writing to SSH channel:", err)
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
					log.Println("Error reading from SSH channel:", err)
				}
				return
			}
			if n>0 {
				//log.Printf("read %d bytes = '%s'", len(buf), string(buf))
				for i:=0;i<n;i++ {
					sshIn <- buf[i]
				}
			}
		}
	}()

	wg.Wait()
}
