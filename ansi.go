package main

import (
	"sync"
)

const (
	esc = 0x1b
	csi = '['
)

type IncomingANSI struct {
	mu  sync.Mutex
	buf []byte
}

const maxIncomingANSI = 16

func queryLengthAtEnd(buf []byte) int {
	if len(buf) < 3 {
		return 0
	}
	// ESC [ 6 n
	if len(buf) >= 4 && buf[len(buf)-4] == esc && buf[len(buf)-3] == csi &&
		buf[len(buf)-2] == '6' && buf[len(buf)-1] == 'n' {
		return 4
	}
	// ESC [ c
	if len(buf) >= 3 && buf[len(buf)-3] == esc && buf[len(buf)-2] == csi && buf[len(buf)-1] == 'c' {
		return 3
	}
	// ESC [ 0 c
	if len(buf) >= 4 && buf[len(buf)-4] == esc && buf[len(buf)-3] == csi &&
		buf[len(buf)-2] == '0' && buf[len(buf)-1] == 'c' {
		return 4
	}
	return 0
}

func syntheticReplyForQuery(query []byte) []byte {
	if len(query) >= 4 && query[len(query)-2] == '6' && query[len(query)-1] == 'n' {
		return []byte{esc, csi, '1', ';', '1', 'R'}
	}
	return []byte{esc, csi, '?', '1', ';', '0', 'c'}
}

func safePrefixLengthIncoming(buf []byte) int {
	if len(buf) == 0 {
		return 0
	}
	if buf[0] != esc {
		return 1
	}
	if len(buf) < 2 {
		return 0
	}
	if buf[1] != csi {
		return 1
	}
	if len(buf) < 3 {
		return 0
	}
	if buf[2] == '6' || buf[2] == '0' || buf[2] == 'c' {
		return 0
	}
	for k := 3; k < len(buf); k++ {
		if buf[k] == esc {
			return k
		}
	}
	return len(buf)
}

func (i *IncomingANSI) Feed(b byte) (toBroadcast []byte, injectToBoard []byte) {
	i.mu.Lock()
	defer i.mu.Unlock()
	i.buf = append(i.buf, b)
	if len(i.buf) > maxIncomingANSI {
		n := safePrefixLengthIncoming(i.buf)
		if n == 0 {
			n = 1
		}
		toBroadcast = make([]byte, n)
		copy(toBroadcast, i.buf[:n])
		i.buf = i.buf[n:]
		return toBroadcast, nil
	}
	if n := queryLengthAtEnd(i.buf); n > 0 {
		query := make([]byte, n)
		copy(query, i.buf[len(i.buf)-n:])
		i.buf = i.buf[:len(i.buf)-n]
		return query, syntheticReplyForQuery(query)
	}
	safe := safePrefixLengthIncoming(i.buf)
	if safe > 0 {
		toBroadcast = make([]byte, safe)
		copy(toBroadcast, i.buf[:safe])
		i.buf = i.buf[safe:]
	}
	return toBroadcast, nil
}

type OutgoingANSI struct {
	mu  sync.Mutex
	buf []byte
}

const maxOutgoingANSI = 32

func replyLengthAtEndScan(buf []byte) int {
	if len(buf) < 6 {
		return 0
	}
	last := len(buf) - 1
	for start := 0; start <= last-5; start++ {
		if buf[start] != esc || (start+1 <= last && buf[start+1] != csi) {
			continue
		}
		if start+2 > last {
			return 0
		}
		if buf[start+2] == '?' {
			i := start + 3
			for i < len(buf) && (buf[i] == ';' || (buf[i] >= '0' && buf[i] <= '9')) {
				i++
			}
			if i == last && buf[i] == 'c' {
				return last - start + 1
			}
		} else if buf[start+2] >= '0' && buf[start+2] <= '9' {
			i := start + 2
			for i < len(buf) && buf[i] >= '0' && buf[i] <= '9' {
				i++
			}
			if i < len(buf) && buf[i] == ';' {
				i++
				for i < len(buf) && buf[i] >= '0' && buf[i] <= '9' {
					i++
				}
				if i == last && buf[i] == 'R' {
					return last - start + 1
				}
			}
		}
	}
	return 0
}

func safePrefixLengthOutgoing(buf []byte) int {
	if len(buf) == 0 {
		return 0
	}
	if buf[0] != esc {
		return 1
	}
	if len(buf) < 2 {
		return 0
	}
	if buf[1] != csi {
		return 1
	}
	if len(buf) < 3 {
		return 0
	}
	if buf[2] == '?' || (buf[2] >= '0' && buf[2] <= '9') {
		return 0
	}
	for k := 3; k < len(buf); k++ {
		if buf[k] == esc {
			return k
		}
	}
	return len(buf)
}

func (o *OutgoingANSI) Feed(b byte) (toForward []byte) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.buf = append(o.buf, b)
	if len(o.buf) > maxOutgoingANSI {
		n := safePrefixLengthOutgoing(o.buf)
		if n == 0 {
			n = 1
		}
		toForward = make([]byte, n)
		copy(toForward, o.buf[:n])
		o.buf = o.buf[n:]
		return toForward
	}
	if n := replyLengthAtEndScan(o.buf); n > 0 {
		o.buf = o.buf[:len(o.buf)-n]
		return nil
	}
	safe := safePrefixLengthOutgoing(o.buf)
	if safe > 0 {
		toForward = make([]byte, safe)
		copy(toForward, o.buf[:safe])
		o.buf = o.buf[safe:]
	}
	return toForward
}
