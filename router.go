package main

import (
//	"fmt"
	"sync"
	"errors"
	"log"
)
type SType int

const (
	_ SType = iota
	SrcNone
	SrcHuman
	SrcMachine
)

type Router struct {
	In       []chan byte
	Out      []chan byte
	SrcType  []SType
	mu       sync.RWMutex
	LEnter   int
}


func NewRouter(n int) *Router {
	r := &Router{
		In:  make([]chan byte, n),
		Out: make([]chan byte, n),
		SrcType: make([]SType, n),
	}
	for i:=0; i<n; i++ {
		r.In[i]=make(chan byte, 4096)
		r.Out[i]=make(chan byte, 4096)
		r.SrcType[i]=SrcNone
	}
	return r
}

func (r *Router) GetFreePos() (int, error) {
    r.mu.Lock()
    defer r.mu.Unlock()
    for i, item := range r.SrcType {
        if item == SrcNone {
            return i, nil
        }
    }
    return -1, errors.New("No channel available")
}

func (r *Router)AttachAt(pos int, stype SType) error{
	r.mu.Lock()
	debugPrint(log.Printf, levelDebug, "Router channel %d attached type=%d\n", pos, stype)
	defer r.mu.Unlock()
	if r.SrcType[pos] != SrcNone {
		return errors.New("Channel is not available")
	}
	r.SrcType[pos]=stype
	return nil
}

func (r *Router)indexOfIn(targetChannel chan byte) int {
	for i, ch := range r.In {
		if ch == targetChannel {
			return i
		}
	}
	return -1
}

func (r *Router)DetachAt(pos int) error{
	r.mu.Lock()
	debugPrint(log.Printf, levelDebug, "Router channel %d detached\n", pos)
	defer r.mu.Unlock()
	if r.SrcType[pos]  == SrcNone {
		return errors.New("Channel is already free")
	}
	r.SrcType[pos]=SrcNone
	return nil
}
func (r *Router)Brodcast(excluded int, data byte) {
	debugPrint(log.Printf, levelCrazy, "Broadcast: collecting channels enter\n")
	r.mu.RLock()
	targets := make([]int, 0, len(r.SrcType))
	for i, ch := range r.SrcType{
		if ch == SrcHuman && i!= excluded {
			targets = append(targets, i)
		}
	}
	r.mu.RUnlock()
	debugPrint(log.Printf, levelCrazy, "Broadcast: collecting channels out (%v)\n", targets)
	for _, i := range targets {
		select {
			case r.In[i] <- data:
				debugPrint(log.Printf, levelCrazy, "Broadcast: send %d  to %d\n", data, i)
			default: // drop 
		}
	}
}

func (r *Router)Unicast(data byte) {
	debugPrint(log.Printf, levelCrazy, "Unicast: collecting channels enter\n")
	r.mu.RLock()
	targets := make([]int, 0, len(r.SrcType))
	for i, ch := range r.SrcType{
		if ch==SrcMachine  {
			targets = append(targets, i)
		}
	}
	r.mu.RUnlock()
	debugPrint(log.Printf, levelCrazy, "Unicast: collecting channels exit (%v)\n", targets)
	for _, i := range targets {
		select {
			case r.In[i] <- data:
				debugPrint(log.Printf, levelCrazy, "Unicast: send %d  to %d\n", data, i)
			default: // drop 
		}
	}
}

func (r *Router) Router() {
	debugPrint(log.Printf, levelInfo, "Router started")
	// for each possible slot spawn a worker
	for i := range r.Out {
		go func(idx int) {
			for data := range r.Out[idx] {
				debugPrint(log.Printf, levelCrazy, "RouterLoop.\n")
				r.mu.RLock()
				st := r.SrcType[idx]
				r.mu.RUnlock()
				if st == SrcNone {
					// skip if free (could log)
					continue
				}
				debugPrint(log.Printf, levelCrazy, "received %d from %d \n", data, idx)
				if st == SrcHuman {
					if data == '\n' {
						r.mu.Lock()
						r.LEnter = idx
						r.mu.Unlock()
					}
					r.Unicast(data) // forwards to SrcMachine
				} else if st == SrcMachine {
					r.Brodcast(idx, data) // forwards to SrcHuman(s)
				}
			}
			debugPrint(log.Printf, levelDebug, "out channel %d closed, worker exit", idx)
		}(i)
	}
}
