package main

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

var SerialBaudRate int

const (
	xmodemSOH   = 0x01
	xmodemEOT   = 0x04
	xmodemACK   = 0x06
	xmodemNAK   = 0x15
	xmodemCRC   = 0x43
	xmodemBlock = 128
	xmodemPad   = 0x1A
)

type SerialIO struct {
	In  <-chan byte
	Out chan<- byte
}

func (s *SerialIO) writeByte(b byte) {
	s.Out <- b
}

func (s *SerialIO) writeBytes(data []byte) {
	for _, b := range data {
		s.Out <- b
	}
}

func (s *SerialIO) pace(n int, bytesPerSec float64, lastSend time.Time) time.Time {
	if bytesPerSec <= 0 || n <= 0 {
		return lastSend
	}
	elapsed := time.Since(lastSend)
	need := time.Duration(float64(n)/bytesPerSec) * time.Second
	if need > elapsed {
		time.Sleep(need - elapsed)
	}
	return time.Now()
}

func (s *SerialIO) writeString(str string) {
	s.writeBytes([]byte(str))
}

func (s *SerialIO) readByteWithTimeout(timeout time.Duration) (byte, error) {
	select {
	case b, ok := <-s.In:
		if !ok {
			return 0, errors.New("channel closed")
		}
		return b, nil
	case <-time.After(timeout):
		return 0, errors.New("timeout")
	}
}

func (s *SerialIO) drainIn(max int) {
	for i := 0; i < max; i++ {
		select {
		case <-s.In:
		default:
			return
		}
	}
}

func (s *SerialIO) sendCommand(cmd string) {
	s.writeString(strings.TrimSpace(cmd) + "\r\n")
}

const (
	defaultDestPlain  = "/tmp/recv_pl.bin"
	defaultDestGzip   = "/tmp/recv_gz.bin"
	defaultDestXmodem = "/tmp/recv_xm.bin"
)

const (
	recvTempFile      = "/tmp/recv_tmp.b64"
	chunkSize         = 128
	targetBytesPerSec = 3500
	depsCheckTimeout  = 5 * time.Second
	depsMarkerOK      = "SEND_DEPS_OK"
	depsMarkerMissing = "SEND_DEPS_MISSING"
)

func remoteDepsCheckCmd() string {
	return "cmd_check(){ command -v $1 >/dev/null 2>&1 || (command -v busybox >/dev/null 2>&1 && busybox $1 --help >/dev/null 2>&1); }; " +
		"cmd_check stty && cmd_check dd && cmd_check base64 && cmd_check gzip && cmd_check rm && echo " + depsMarkerOK + " || echo " + depsMarkerMissing
}

func chunkDelay() time.Duration {
	if targetBytesPerSec <= 0 {
		return 37 * time.Millisecond
	}
	secPerChunk := float64(chunkSize) / float64(targetBytesPerSec)
	return time.Duration(int64(secPerChunk * 1e9))
}

func remoteRecvStart() string {
	return fmt.Sprintf("stty -icanon -echo; dd of=%s bs=1", recvTempFile)
}

func remoteRecvFinishPlain(dest string) string {
	return fmt.Sprintf("base64 -d %s > %s && rm -f %s", recvTempFile, dest, recvTempFile)
}

func remoteRecvFinishGzip(dest string) string {
	return fmt.Sprintf("base64 -d %s | gunzip > %s && rm -f %s", recvTempFile, dest, recvTempFile)
}

func CheckSerialDeps(io *SerialIO) (result string, err error) {
	io.drainIn(4096)
	io.sendCommand(remoteDepsCheckCmd())
	var buf []byte
	deadline := time.Now().Add(depsCheckTimeout)
	for time.Now().Before(deadline) {
		b, err := io.readByteWithTimeout(200 * time.Millisecond)
		if err != nil {
			if len(buf) > 0 && (bytes.Contains(buf, []byte(depsMarkerOK)) || bytes.Contains(buf, []byte(depsMarkerMissing))) {
				break
			}
			continue
		}
		buf = append(buf, b)
		if len(buf) > 1024 {
			buf = buf[len(buf)-512:]
		}
		if bytes.Contains(buf, []byte(depsMarkerOK)) {
			return "ok", nil
		}
		if bytes.Contains(buf, []byte(depsMarkerMissing)) {
			return "missing", nil
		}
	}
	if bytes.Contains(buf, []byte(depsMarkerOK)) {
		return "ok", nil
	}
	if bytes.Contains(buf, []byte(depsMarkerMissing)) {
		return "missing", nil
	}
	return "timeout", nil
}

func (s *SerialIO) sendChunkedBase64(encoded string) {
	delay := chunkDelay()
	for i := 0; i < len(encoded); i += chunkSize {
		end := i + chunkSize
		if end > len(encoded) {
			end = len(encoded)
		}
		s.writeString(encoded[i:end])
		time.Sleep(delay)
	}
	s.writeString("\n")
	s.writeByte(0x03)
}

func SendFilePlain(io *SerialIO, localPath, destPath string) error {
	debugPrint(log.Printf, levelDebug, "Enter\n")
	if destPath == "" {
		destPath = defaultDestPlain
	}
	debugPrint(log.Printf, levelDebug, "sending '%s'\n", remoteRecvStart())
	io.sendCommand(remoteRecvStart())
	time.Sleep(500 * time.Millisecond)

	data, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}
	encoded := base64.StdEncoding.EncodeToString(data)
	debugPrint(log.Printf, levelDebug, "Start transmissions\n")
	io.sendChunkedBase64(encoded)
	debugPrint(log.Printf, levelDebug, "End trasmissions\n")
	time.Sleep(2 * time.Second)
	io.sendCommand("stty sane")
	time.Sleep(2 * time.Second)
	io.sendCommand(remoteRecvFinishPlain(destPath))
	debugPrint(log.Printf, levelDebug, "last sent '%s'\n", remoteRecvFinishPlain(destPath))
	return nil
}

func SendFileGzip(io *SerialIO, localPath, destPath string) error {
	debugPrint(log.Printf, levelDebug, "Enter\n")
	if destPath == "" {
		destPath = defaultDestGzip
	}
	debugPrint(log.Printf, levelDebug, "sending '%s'\n", remoteRecvStart())
	io.sendCommand(remoteRecvStart())
	time.Sleep(500 * time.Millisecond)

	data, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("gzip: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("gzip close: %w", err)
	}
	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
	debugPrint(log.Printf, levelDebug, "Start transmissions\n")
	io.sendChunkedBase64(encoded)
	debugPrint(log.Printf, levelDebug, "End trasmissions\n")
	time.Sleep(2 * time.Second)
	io.sendCommand("stty sane")
	time.Sleep(3 * time.Second)
	io.sendCommand(remoteRecvFinishGzip(destPath))
	debugPrint(log.Printf, levelDebug, "last sent '%s'\n", remoteRecvFinishGzip(destPath))
	return nil
}

func crc16CCITT(data []byte) uint16 {
	var crc uint16
	for _, b := range data {
		crc ^= uint16(b) << 8
		for i := 0; i < 8; i++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}

func SendFileXmodemUnix(io *SerialIO, localPath, destPath string) error {
	if destPath == "" {
		destPath = defaultDestXmodem
	}
	io.drainIn(4096)
	io.sendCommand("rx " + destPath)
	return xmodemSend(io, localPath, 10*time.Second)
}

func SendFileXmodemUboot(io *SerialIO, localPath string) error {
	io.drainIn(4096)
	io.sendCommand("loadx")
	return xmodemSend(io, localPath, 15*time.Second)
}

func xmodemSend(io *SerialIO, localPath string, startTimeout time.Duration) error {
	data, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	var useCRC bool
	deadline := time.Now().Add(startTimeout)
	for time.Now().Before(deadline) {
		b, err := io.readByteWithTimeout(500 * time.Millisecond)
		if err != nil {
			continue
		}
		if b == xmodemNAK {
			useCRC = false
			break
		}
		if b == xmodemCRC {
			useCRC = true
			break
		}
	}
	if time.Now().After(deadline) {
		return errors.New("xmodem: no NAK/CRC from receiver within timeout (is rx/loadx running on the remote?)")
	}

	blockNum := byte(1)
	const blockSize = xmodemBlock
	numBlocks := (len(data) + blockSize - 1) / blockSize

	for i := 0; i < numBlocks; i++ {
		start := i * blockSize
		end := start + blockSize
		if end > len(data) {
			end = len(data)
		}
		chunk := data[start:end]

		block := make([]byte, blockSize)
		copy(block, chunk)
		for j := len(chunk); j < blockSize; j++ {
			block[j] = xmodemPad
		}

		var ack bool
		for retry := 0; retry < 10; retry++ {
			io.writeByte(xmodemSOH)
			io.writeByte(blockNum)
			io.writeByte(0xFF - blockNum)
			io.writeBytes(block)
			if useCRC {
				crc := crc16CCITT(block)
				io.writeByte(byte(crc >> 8))
				io.writeByte(byte(crc))
			} else {
				var sum byte
				for _, b := range block {
					sum += b
				}
				io.writeByte(sum)
			}

			b, err := io.readByteWithTimeout(5 * time.Second)
			if err != nil {
				continue
			}
			if b == xmodemACK {
				ack = true
				break
			}
			if b == xmodemNAK {
				continue
			}
		}
		if !ack {
			return fmt.Errorf("xmodem: no ACK after block %d", blockNum)
		}
		blockNum++
	}

	for retry := 0; retry < 10; retry++ {
		io.writeByte(xmodemEOT)
		b, err := io.readByteWithTimeout(3 * time.Second)
		if err != nil {
			continue
		}
		if b == xmodemACK {
			return nil
		}
	}
	return errors.New("xmodem: no ACK for EOT")
}

func SendFile(io *SerialIO, localPath, mode, destPath string) error {
	debugPrint(log.Printf, levelDebug, "SendFile %s %s %s\n", localPath, mode, destPath)
	switch mode {
	case "plain":
		return SendFilePlain(io, localPath, destPath)
	case "gzip":
		return SendFileGzip(io, localPath, destPath)
	case "xmodem_unix":
		return SendFileXmodemUnix(io, localPath, destPath)
	case "xmodem_uboot":
		return SendFileXmodemUboot(io, localPath)
	default:
		return fmt.Errorf("unknown mode %q (use: plain, gzip, xmodem_unix, xmodem_uboot)", mode)
	}
}
