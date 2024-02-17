package main

import (
	"log"
	"sync"
	"golang.org/x/crypto/ssh"
	"io"
	"os"
	"net"
	"fmt"
)

func SSHHandler(sshPort string, desc string, sshIn chan<- []byte, sshOut <-chan []byte) {
	authorizedKeysBytes, err := os.ReadFile("authorized_keys")
	if err != nil {
		log.Fatalf("Failed to load authorized_keys, err: %v", err)
	}

	authorizedKeysMap := map[string]bool{}
	for len(authorizedKeysBytes) > 0 {
		pubKey, _, _, rest, err := ssh.ParseAuthorizedKey(authorizedKeysBytes)
		if err != nil {
			log.Println("here")
			log.Fatal(err)
		}

		authorizedKeysMap[string(pubKey.Marshal())] = true
		authorizedKeysBytes = rest
	}
	config := &ssh.ServerConfig{
		PublicKeyCallback: func(c ssh.ConnMetadata, pubKey ssh.PublicKey) (*ssh.Permissions, error) {
			if authorizedKeysMap[string(pubKey.Marshal())] {
				return &ssh.Permissions{
					Extensions: map[string]string{
						"pubkey-fp": ssh.FingerprintSHA256(pubKey),
					},
				}, nil
			}
			return nil, fmt.Errorf("unknown public key for %q", c.User())
		},
	}

	privateBytes, err := os.ReadFile("id_rsa")
	if err != nil {
		log.Fatal("Failed to load private key: ", err)
	}
	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		log.Fatal("Failed to parse private key: ", err)
	}
	config.AddHostKey(private)


	listener, err := net.Listen("tcp", ":"+sshPort)
	if err != nil {
		log.Fatal("failed to listen for ssh:", err)
	}
	defer listener.Close()

	log.Printf("Starting %s SSH server on port %s\n", desc, sshPort)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal("failed to accept incoming connection:", err)
		}

		go handleSSHConnection(conn, config, sshIn, sshOut)
	}
}

func handleSSHConnection(conn net.Conn, config *ssh.ServerConfig, sshIn chan<- []byte, sshOut <-chan []byte) {
	defer conn.Close()

	sshConn, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		log.Println("failed to establish SSH connection:", err)
		return
	}
	defer sshConn.Close()

	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		go handleSSHChannel(newChannel, sshIn, sshOut)
	}
}

func handleSSHChannel(newChannel ssh.NewChannel, sshIn chan<- []byte, sshOut <-chan []byte) {
	if newChannel.ChannelType() != "session" {
		newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
		return
	}

	channel, _, err := newChannel.Accept()
	if err != nil {
		log.Println("failed to accept channel:", err)
		return
	}
	defer channel.Close()

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			data := <-sshOut
			_, err := channel.Write(data)
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
			sshIn <- buf[:n]
		}
	}()

	wg.Wait()
}
