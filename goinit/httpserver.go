package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

const (
	apiPrefix     = "/api/"
	apiStat       = "stat"
	apiReadDevice = "read"
	apiWriteDevice = "write"
	apiReboot     = "reboot"
)

// StatResponse is the JSON summary for GET /api/stat.
type StatResponse struct {
	CPU     CPUInfo      `json:"cpu"`
	Memory  MemoryInfo   `json:"memory"`
	Storage StorageInfo  `json:"storage"`
	Network NetworkInfo  `json:"network"`
}

type CPUInfo struct {
	ModelName string `json:"model_name,omitempty"`
	NumCPU    int    `json:"num_cpu"`
}

type MemoryInfo struct {
	MemTotalKB uint64 `json:"mem_total_kb"`
	MemFreeKB  uint64 `json:"mem_free_kb"`
}

type StorageDevice struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Size uint64 `json:"size"` // bytes
	Type string `json:"type"` // "scsi" | "mmc" | "nvme" | "raw"
}

type StorageInfo struct {
	Devices []StorageDevice `json:"devices"`
}

type NetInterface struct {
	Name string   `json:"name"`
	Up   bool     `json:"up"`
	IPs  []string `json:"ips,omitempty"`
}

type NetworkInfo struct {
	Interfaces  []NetInterface `json:"interfaces"`
	MgmtIf      string        `json:"mgmt_if,omitempty"`
	MgmtIP      string        `json:"mgmt_ip,omitempty"`
}

func parseMeminfo() (total, free uint64) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			fmt.Sscanf(strings.TrimSpace(strings.TrimPrefix(line, "MemTotal:")), "%d", &total)
		}
		if strings.HasPrefix(line, "MemFree:") {
			fmt.Sscanf(strings.TrimSpace(strings.TrimPrefix(line, "MemFree:")), "%d", &free)
		}
	}
	return total, free
}

func parseCPUinfo() (model string, num int) {
	f, err := os.Open("/proc/cpuinfo")
	if err != nil {
		return "", 0
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "model name") {
			if i := strings.Index(line, ":"); i >= 0 {
				model = strings.TrimSpace(line[i+1:])
			}
		}
		if strings.HasPrefix(line, "processor") {
			num++
		}
	}
	if num == 0 {
		num = 1
	}
	return model, num
}

func storageType(name string) string {
	if strings.HasPrefix(name, "sd") {
		return "scsi"
	}
	if strings.HasPrefix(name, "mmcblk") {
		return "mmc"
	}
	if strings.HasPrefix(name, "nvme") {
		return "nvme"
	}
	return "raw"
}

func listStorage() []StorageDevice {
	var out []StorageDevice
	blockDir := "/sys/block"
	entries, err := os.ReadDir(blockDir)
	if err != nil {
		return out
	}
	for _, e := range entries {
		name := e.Name()
		// skip loop and ram
		if strings.HasPrefix(name, "loop") || strings.HasPrefix(name, "ram") {
			continue
		}
		sizePath := filepath.Join(blockDir, name, "size")
		data, err := os.ReadFile(sizePath)
		if err != nil {
			continue
		}
		blocks, _ := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
		size := blocks * 512
		out = append(out, StorageDevice{
			Name: name,
			Path: "/dev/" + name,
			Size: size,
			Type: storageType(name),
		})
	}
	return out
}

func listNetwork() []NetInterface {
	var out []NetInterface
	netDir := "/sys/class/net"
	entries, err := os.ReadDir(netDir)
	if err != nil {
		return out
	}
	for _, e := range entries {
		name := e.Name()
		flagsPath := filepath.Join(netDir, name, "flags")
		up := false
		if data, err := os.ReadFile(flagsPath); err == nil {
			flags, _ := strconv.ParseUint(strings.TrimSpace(string(data)), 0, 32)
			// IFF_UP = 1
			up = (flags & 1) != 0
		}
		iface := NetInterface{Name: name, Up: up}
		// Try to read IP from /sys/class/net/name/address or we could run ip (not in initramfs). Skip for now; we set MgmtIP globally.
		out = append(out, iface)
	}
	return out
}

func handleStat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	model, num := parseCPUinfo()
	total, free := parseMeminfo()
	storage := listStorage()
	netIfs := listNetwork()
	resp := StatResponse{
		CPU:     CPUInfo{ModelName: model, NumCPU: num},
		Memory:  MemoryInfo{MemTotalKB: total, MemFreeKB: free},
		Storage: StorageInfo{Devices: storage},
		Network: NetworkInfo{
			Interfaces: netIfs,
			MgmtIf:     MgmtIfName,
			MgmtIP:     MgmtIP,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func isBlockDevice(path string) bool {
	base := strings.TrimPrefix(path, "/dev/")
	if base == "" || strings.Contains(base, "/") {
		return false
	}
	// Must exist under /sys/block
	_, err := os.Stat(filepath.Join("/sys/block", base))
	return err == nil
}

func handleReadDevice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	device := r.URL.Query().Get("device")
	if device == "" {
		http.Error(w, "missing device query", http.StatusBadRequest)
		return
	}
	if !isBlockDevice(device) {
		http.Error(w, "invalid or unsupported device", http.StatusBadRequest)
		return
	}
	f, err := os.Open(device)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if st.Mode()&os.ModeDevice == 0 {
		http.Error(w, "not a device", http.StatusBadRequest)
		return
	}
	size := st.Size()
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(device)+".bin")
	if size > 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	}
	io.Copy(w, f)
}

func handleWriteDevice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	device := r.URL.Query().Get("device")
	if device == "" {
		http.Error(w, "missing device query", http.StatusBadRequest)
		return
	}
	if !isBlockDevice(device) {
		http.Error(w, "invalid or unsupported device", http.StatusBadRequest)
		return
	}
	f, err := os.OpenFile(device, os.O_WRONLY, 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer f.Close()
	n, err := io.Copy(f, r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"written": n})
}

func handleReboot(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "rebooting"})
	go func() {
		syscall.Sync()
		syscall.Reboot(syscall.LINUX_REBOOT_CMD_RESTART)
	}()
}

func apiHandler(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, apiPrefix)
	path = strings.Trim(path, "/")
	switch path {
	case apiStat:
		handleStat(w, r)
	case apiReadDevice:
		handleReadDevice(w, r)
	case apiWriteDevice:
		handleWriteDevice(w, r)
	case apiReboot:
		handleReboot(w, r)
	default:
		http.NotFound(w, r)
	}
}

// StartHTTPServer starts the control API server on listenAddr (e.g. ":8080").
// It runs in a goroutine; use the returned channel to wait for server exit (e.g. on error).
func StartHTTPServer(listenAddr string) chan error {
	errCh := make(chan error, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		apiHandler(w, r)
	})
	go func() {
		errCh <- http.ListenAndServe(listenAddr, mux)
	}()
	return errCh
}
