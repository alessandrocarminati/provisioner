package main

import (
	"bytes"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	defaultBoardAPIPort   = "8080"
	boardStatPollInterval = 30 * time.Second
)

type WatchHandler func(line string, matches []string)

type WatchRule struct {
	Name    string
	Pattern *regexp.Regexp
	Handler WatchHandler
}

var watchRules = []WatchRule{
	{
		Name:    "PROVISIONER_MGMT_IF",
		Pattern: regexp.MustCompile(`PROVISIONER_MGMT_IF=([a-z_\-]+[0-9]+)`),
		Handler: handleMgmtIf,
	},
	{
		Name:    "PROVISIONER_MGMT_IP",
		Pattern: regexp.MustCompile(`PROVISIONER_MGMT_IP=([0-9]+\.[0-9]+\.[0-9]+\.[0-9]+)`), // Beware terminal control characters
		Handler: handleMgmtIP,
	},
}

var boardWatcherState struct {
	mu       sync.RWMutex
	boardIP  string
	boardIf  string
	lastStat []byte
	lastTime time.Time
	running  bool
	pollStop chan struct{}
}

func handleMgmtIf(line string, matches []string) {
	if len(matches) < 2 {
		return
	}
	ifName := strings.TrimSpace(matches[1])
	boardWatcherState.mu.Lock()
	boardWatcherState.boardIf = ifName
	boardWatcherState.mu.Unlock()
	debugPrint(log.Printf, levelInfo, "board watcher: mgmt if=%s", ifName)
}

func handleMgmtIP(line string, matches []string) {
	if len(matches) < 2 {
		return
	}
	ip := strings.TrimSpace(matches[1])
	boardWatcherState.mu.Lock()
	boardWatcherState.boardIP = ip
	boardWatcherState.mu.Unlock()
	debugPrint(log.Printf, levelInfo, "board watcher: mgmt ip=%s", ip)
	startPollerOnce(ip, defaultBoardAPIPort)
}

func StartBoardWatcher(router *Router) {
	pos, err := router.GetFreePos()
	if err != nil {
		debugPrint(log.Printf, levelWarning, "board watcher: no free channel: %v", err)
		return
	}
	if err := router.AttachAt(pos, SrcHuman); err != nil {
		debugPrint(log.Printf, levelWarning, "board watcher: attach: %v", err)
		return
	}
	boardWatcherState.mu.Lock()
	boardWatcherState.running = true
	boardWatcherState.pollStop = make(chan struct{})
	boardWatcherState.mu.Unlock()
	debugPrint(log.Printf, levelDebug, "board watcher: attach: success")

	go readSerialLoop(router, pos)
}

func readSerialLoop(router *Router, pos int) {
	in := router.In[pos]
	var line []byte
	const maxLine = 2048
	for b := range in {
		if b == '\n' || b == '\r' {
			if len(line) > 0 {
				processSerialLine(line)
				line = nil
			}
			continue
		}
		line = append(line, b)
		if len(line) >= maxLine {
			line = nil
		}
	}
	boardWatcherState.mu.Lock()
	boardWatcherState.running = false
	boardWatcherState.mu.Unlock()
}

func processSerialLine(line []byte) {
	s := string(bytes.TrimSpace(line))
	if s == "" {
		return
	}
	for _, rule := range watchRules {
		if rule.Pattern == nil || rule.Handler == nil {
			continue
		}
		matches := rule.Pattern.FindStringSubmatch(s)
		if matches == nil {
			continue
		}
		debugPrint(log.Printf, levelDebug, "board watcher: rule %q matched", rule.Name)
		rule.Handler(s, matches)
	}
}

var pollerStarted bool
var pollerMu sync.Mutex

func startPollerOnce(ip, port string) {
	pollerMu.Lock()
	if pollerStarted {
		pollerMu.Unlock()
		return
	}
	pollerStarted = true
	pollerMu.Unlock()
	go pollStatLoop(ip, port)
}

func pollStatLoop(ip, port string) {
	url := "http://" + ip + ":" + port + "/api/stat"
	ticker := time.NewTicker(boardStatPollInterval)
	defer ticker.Stop()
	fetchAndStore(url)
	for {
		boardWatcherState.mu.RLock()
		stop := boardWatcherState.pollStop
		boardWatcherState.mu.RUnlock()
		select {
		case <-stop:
			return
		case <-ticker.C:
			fetchAndStore(url)
		}
	}
}

func fetchAndStore(url string) {
	resp, err := http.Get(url)
	if err != nil {
		debugPrint(log.Printf, levelDebug, "board watcher: stat fetch: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		debugPrint(log.Printf, levelDebug, "board watcher: stat status %d", resp.StatusCode)
		return
	}
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return
	}
	boardWatcherState.mu.Lock()
	boardWatcherState.lastStat = buf.Bytes()
	boardWatcherState.lastTime = time.Now()
	boardWatcherState.mu.Unlock()
}

func GetBoardStat() (boardIP, boardIf string, statJSON []byte, lastTime time.Time, ok bool) {
	boardWatcherState.mu.RLock()
	defer boardWatcherState.mu.RUnlock()
	if boardWatcherState.boardIP == "" && len(boardWatcherState.lastStat) == 0 {
		return "", "", nil, time.Time{}, false
	}
	boardIP = boardWatcherState.boardIP
	boardIf = boardWatcherState.boardIf
	lastTime = boardWatcherState.lastTime
	if len(boardWatcherState.lastStat) > 0 {
		statJSON = make([]byte, len(boardWatcherState.lastStat))
		copy(statJSON, boardWatcherState.lastStat)
		ok = true
	} else {
		ok = boardIP != ""
	}
	return boardIP, boardIf, statJSON, lastTime, ok
}
