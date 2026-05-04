package showports

import (
	"bufio"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	showpidscmd "github.com/jeffinity/otter/internal/service/command/showpids"
)

const listenState = "LISTEN"

type Dependencies struct {
	PidFinder  showpidscmd.Finder
	ConnFinder ConnFinder
	Out        io.Writer
}

type ConnFinder interface {
	Find(ctx context.Context, pid int32) ([]Connection, error)
}

type Connection struct {
	Type   int
	Status string
	IP     string
	Port   uint32
}

func Run(ctx context.Context, args []string, deps Dependencies) error {
	pids, err := findPids(ctx, args, deps)
	if err != nil {
		return err
	}
	connections := findConnections(ctx, pids, deps)

	out := deps.Out
	if out == nil {
		out = os.Stdout
	}
	for _, conn := range connections {
		if conn.Status != listenState {
			continue
		}
		if _, err := fmt.Fprintf(out, "%s Listen %s:%d\n", netType(conn), conn.IP, conn.Port); err != nil {
			return err
		}
	}
	return nil
}

func findPids(ctx context.Context, args []string, deps Dependencies) ([]int32, error) {
	finder := deps.PidFinder
	if finder == nil {
		finder = showpidscmd.DefaultFinder{}
	}

	pids := make([]int32, 0)
	for _, arg := range args {
		serviceName := strings.TrimSuffix(arg, ".service")
		servicePids, err := finder.Find(ctx, serviceName)
		if err != nil {
			return nil, fmt.Errorf("find pids for service '%s' failed: %w", serviceName, err)
		}
		pids = append(pids, servicePids...)
	}
	return pids, nil
}

func findConnections(ctx context.Context, pids []int32, deps Dependencies) []Connection {
	finder := deps.ConnFinder
	if finder == nil {
		finder = ProcConnFinder{}
	}

	connections := make([]Connection, 0)
	for _, pid := range pids {
		pidConnections, err := finder.Find(ctx, pid)
		if err != nil {
			continue
		}
		connections = append(connections, pidConnections...)
	}
	return connections
}

func netType(conn Connection) string {
	switch conn.Type {
	case syscall.SOCK_STREAM:
		return "TCP"
	case syscall.SOCK_DGRAM:
		return "UDP"
	default:
		return "???"
	}
}

type ProcConnFinder struct {
	Root string
}

func (f ProcConnFinder) Find(ctx context.Context, pid int32) ([]Connection, error) {
	_ = ctx

	root := f.Root
	if root == "" {
		root = "/proc"
	}

	inodes, err := socketInodes(filepath.Join(root, strconv.Itoa(int(pid)), "fd"))
	if err != nil {
		return nil, err
	}

	connections := make([]Connection, 0)
	for _, spec := range procNetFiles(root) {
		items, err := readProcNet(spec.path, spec.sockType, spec.ipv6, inodes)
		if err != nil {
			return nil, err
		}
		connections = append(connections, items...)
	}
	return connections, nil
}

func socketInodes(fdPath string) (map[string]struct{}, error) {
	entries, err := os.ReadDir(fdPath)
	if err != nil {
		return nil, err
	}
	inodes := map[string]struct{}{}
	for _, entry := range entries {
		target, err := os.Readlink(filepath.Join(fdPath, entry.Name()))
		if err != nil {
			continue
		}
		if strings.HasPrefix(target, "socket:[") && strings.HasSuffix(target, "]") {
			inodes[strings.TrimSuffix(strings.TrimPrefix(target, "socket:["), "]")] = struct{}{}
		}
	}
	return inodes, nil
}

type procNetSpec struct {
	path     string
	sockType int
	ipv6     bool
}

func procNetFiles(root string) []procNetSpec {
	return []procNetSpec{
		{path: filepath.Join(root, "net", "tcp"), sockType: syscall.SOCK_STREAM},
		{path: filepath.Join(root, "net", "tcp6"), sockType: syscall.SOCK_STREAM, ipv6: true},
		{path: filepath.Join(root, "net", "udp"), sockType: syscall.SOCK_DGRAM},
		{path: filepath.Join(root, "net", "udp6"), sockType: syscall.SOCK_DGRAM, ipv6: true},
	}
}

func readProcNet(path string, sockType int, ipv6 bool, inodes map[string]struct{}) ([]Connection, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	connections := make([]Connection, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		conn, ok := parseProcNetLine(scanner.Text(), sockType, ipv6, inodes)
		if ok {
			connections = append(connections, conn)
		}
	}
	return connections, scanner.Err()
}

func parseProcNetLine(line string, sockType int, ipv6 bool, inodes map[string]struct{}) (Connection, bool) {
	fields := strings.Fields(line)
	if len(fields) < 10 || !strings.HasSuffix(fields[0], ":") {
		return Connection{}, false
	}
	if _, ok := inodes[fields[9]]; !ok {
		return Connection{}, false
	}
	ip, port, err := parseLocalAddress(fields[1], ipv6)
	if err != nil {
		return Connection{}, false
	}
	return Connection{Type: sockType, Status: procStatus(fields[3]), IP: ip, Port: port}, true
}

func parseLocalAddress(value string, ipv6 bool) (string, uint32, error) {
	addr, portHex, ok := strings.Cut(value, ":")
	if !ok {
		return "", 0, fmt.Errorf("invalid local address")
	}
	port, err := strconv.ParseUint(portHex, 16, 32)
	if err != nil {
		return "", 0, err
	}
	if ipv6 {
		ip, err := parseIPv6(addr)
		return ip, uint32(port), err
	}
	ip, err := parseIPv4(addr)
	return ip, uint32(port), err
}

func parseIPv4(value string) (string, error) {
	raw, err := hex.DecodeString(value)
	if err != nil {
		return "", err
	}
	for i, j := 0, len(raw)-1; i < j; i, j = i+1, j-1 {
		raw[i], raw[j] = raw[j], raw[i]
	}
	return net.IP(raw).String(), nil
}

func parseIPv6(value string) (string, error) {
	raw, err := hex.DecodeString(value)
	if err != nil {
		return "", err
	}
	if len(raw) != net.IPv6len {
		return "", fmt.Errorf("invalid ipv6 address")
	}
	for i := 0; i < len(raw); i += 4 {
		raw[i], raw[i+3] = raw[i+3], raw[i]
		raw[i+1], raw[i+2] = raw[i+2], raw[i+1]
	}
	return net.IP(raw).String(), nil
}

func procStatus(value string) string {
	if value == "0A" {
		return listenState
	}
	return value
}
