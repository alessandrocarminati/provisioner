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
	mu       sync.Mutex
}


func NewRouter(n int) *Router {
	r := &Router{
		In:  make([]chan byte, n),
		Out: make([]chan byte, n),
		SrcType: make([]SType, n),
	}
	for i:=0; i<n; i++ {
		r.In[i]=make(chan byte, 10)
		r.Out[i]=make(chan byte, 10)
		r.SrcType[i]=SrcNone
	}
	return r
}

func (r *Router)GetFreePos() (int, error){
	for i, item := range r.SrcType {
		if item == SrcNone {
			return i, nil
		}
	}
	return -1, errors.New("No channel available")
}
func (r *Router)AttachAt(pos int, stype SType) error{
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.SrcType[pos] != SrcNone {
		return errors.New("Channel is not available")
	}
	r.SrcType[pos]=stype
	return nil
}

func (r *Router)DetachAt(pos int) error{
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.SrcType[pos]  == SrcNone {
		return errors.New("Channel is already free")
	}
	r.SrcType[pos]=SrcNone
	return nil
}
func (r *Router)Brodcast(excluded int, data byte) {
	for i, ch := range r.SrcType{
		if ch==SrcHuman {
			if i!= excluded {
				select {
				case r.In[i] <- data:
					debugPrint(log.Printf, levelCrazy, "Broadcast: send %d  to %d\n", data, i)
				}
			}
		}
	}
}

func (r *Router)Unicast(data byte) {
	for i, ch := range r.SrcType{
		if ch==SrcMachine  {
			select {
			case r.In[i] <- data:
				debugPrint(log.Printf, levelCrazy, "Unicast: send %d  to %d\n", data, i)
			}
		}
	}
}

func (r *Router)Router() {
	debugPrint(log.Printf, levelInfo, "Router started")
	go func() {
		for {
			debugPrint(log.Printf, levelCrazy, "start polling cycle")
			for i, ch := range r.SrcType{
				if ch!= SrcNone {
					debugPrint(log.Printf, levelCrazy, "polling %d\n", i)
					select {
					case data, ok := <-r.Out[i]:
						debugPrint(log.Printf, levelCrazy, "received %d from %d \n", data, i)
						if !ok {
							panic("stocazzo r")
						}
						if ch==SrcHuman {
							r.Unicast(data)
							continue
						}
						if ch==SrcMachine {
							r.Brodcast(i,data)
						}
					default:
					}
				}
			}
		}
	}()
}
