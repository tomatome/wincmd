// sysinfo.go
package main

import (
	"flag"
	"fmt"
	"math"
	"net"
	"os"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/tomatome/win"
)

const (
	SHOW_CPU  = (1 << 1)
	SHOW_MEM  = (1 << 2)
	SHOW_DISK = (1 << 3)
	SHOW_NET  = (1 << 4)
)

var ShowFlag int

func main() {
	var (
		cpu, mem, disk, net bool
	)
	flag.BoolVar(&cpu, "c", false, "Show CPU")
	flag.BoolVar(&mem, "m", false, "Show Mem")
	flag.BoolVar(&disk, "d", false, "Show Disk")
	flag.BoolVar(&net, "n", false, "Show Net")
	flag.Parse()

	ShowFlag = 0
	if cpu || mem || disk || net {
		ShowFlag = 1
		if cpu {
			ShowFlag |= SHOW_CPU
		}
		if mem {
			ShowFlag |= SHOW_MEM
		}
		if disk {
			ShowFlag |= SHOW_DISK
		}
		if net {
			ShowFlag |= SHOW_NET
		}
	}

	SystemInfo()
	PrintCpuUsage()
	MemInfo()
	SwapMemory()
	PrintNetIO()
	PrintDisks()
}
func MemInfo() {
	if ShowFlag != 0 && ShowFlag&SHOW_MEM == 0 {
		return
	}
	var memInfo win.MEMORYSTATUSEX
	memInfo.CbSize = win.DWORD(unsafe.Sizeof(memInfo))
	ret := win.GlobalMemoryStatusEx(&memInfo)
	if !ret {
		fmt.Println("Mem error")
		return
	}

	fmt.Println(" 内存:")
	fmt.Println("  Total:", Size(memInfo.UllTotalPhys), "  Free:", Size(memInfo.UllAvailPhys))
}

//swap
func SwapMemory() {
	if ShowFlag != 0 && ShowFlag&SHOW_MEM == 0 {
		return
	}
	var perfInfo win.PERFORMANCE_INFORMATION
	perfInfo.Cb = win.DWORD(unsafe.Sizeof(perfInfo))
	ret := win.GetPerformanceInfo(&perfInfo, perfInfo.Cb)
	if !ret {
		fmt.Println("Swap error")
		return
	}
	total := perfInfo.CommitLimit * perfInfo.PageSize
	used := perfInfo.CommitTotal * perfInfo.PageSize
	free := total - used

	fmt.Println(" 交换区:")
	fmt.Println("  Total:", Size(uint64(total)), "  Free:", Size(uint64(free)))
}

func DiskUsage(path string) (uint64, uint64) {
	lpFreeBytesAvailable := uint64(0)
	var (
		total uint64
		free  uint64
	)
	ret := win.GetDiskFreeSpaceEx(path,
		lpFreeBytesAvailable,
		&total,
		&free)
	if !ret {
		return 0, 0
	}
	return total, free
}
func IOCTL_STORAGE_GET_DEVICE_NUMBER() win.DWORD {
	return (((0x0000002d) << 16) | ((0) << 14) | ((0x0420) << 2) | (0))
}

type STORAGE_DEVICE_NUMBER struct {
	DeviceType      win.DWORD
	DeviceNumber    win.ULONG
	PartitionNumber win.ULONG
}

func volumeInfo(path string) int {
	var num uint32
	var number STORAGE_DEVICE_NUMBER

	d := "\\\\.\\" + path
	h := win.CreateFile(d, 0, 0x00000001|0x00000002, nil, 3, 0, win.HANDLE(uintptr(0)))
	win.DeviceIoControl(h, IOCTL_STORAGE_GET_DEVICE_NUMBER(), win.LPVOID(0), 0, win.LPVOID(uintptr(unsafe.Pointer(&number))), 254, &num, nil)

	win.CloseHandle(h)

	return int(number.DeviceNumber)
}

type DiskInfo struct {
	Path      string
	DeviceNum int
	Total     uint64
	Free      uint64
	ReadRate  float64
	WriteRate float64
}

//硬盘信息
func Disks() []DiskInfo {
	lpBuffer := make([]uint16, 254)
	ret := win.GetLogicalDriveStrings(win.DWORD(len(lpBuffer)), win.LPWSTR(&lpBuffer[0]))
	if ret == 0 {
		return nil
	}

	infos := make([]DiskInfo, 0, 10)
	for _, v := range lpBuffer {
		if v >= 65 && v <= 90 {
			path := string(v) + ":"
			if path == "A:" || path == "B:" {
				continue
			}

			var info DiskInfo
			info.Path = path
			info.DeviceNum = volumeInfo(path)
			total, free := DiskUsage(path)
			info.Total = total
			info.Free = free
			infos = append(infos, info)
		}
	}
	return infos
}

const (
	PDH_FMT_LONG   = (win.DWORD(0x00000100))
	PDH_FMT_DOUBLE = (win.DWORD(0x00000200))
	PDH_FMT_LARGE  = (win.DWORD(0x00000400))
)

type PerfomMonitor struct {
	hQuery win.PDH_HQUERY
	count  win.PDH_HCOUNTER
	count1 win.PDH_HCOUNTER
	//ReadCount Disk
}

func (pm *PerfomMonitor) value() (float64, float64) {
	//var CounterType uint32
	var DisplayValue, DisplayValue1 win.PDH_FMT_COUNTERVALUE
	win.PdhGetFormattedCounterValue(pm.count, PDH_FMT_DOUBLE, nil, &DisplayValue)
	win.PdhGetFormattedCounterValue(pm.count1, PDH_FMT_DOUBLE, nil, &DisplayValue1)
	return *DisplayValue.DoubleValue(), *DisplayValue1.DoubleValue()
}
func openPm() *PerfomMonitor {
	var pm PerfomMonitor
	win.PdhOpenQuery(nil, nil, &pm.hQuery)
	return &pm
}
func (pm *PerfomMonitor) close() {
	win.PdhCloseQuery(pm.hQuery)
}

func (pm *PerfomMonitor) addDiskCounter() {
	win.PdhAddCounter(pm.hQuery, "\\PhysicalDisk(*)\\Disk Read Bytes/sec", nil, &pm.count)
	win.PdhAddCounter(pm.hQuery, "\\PhysicalDisk(*)\\Disk Write Bytes/sec", nil, &pm.count1)
}
func (pm *PerfomMonitor) query() {
	win.PdhCollectQueryData(pm.hQuery)
}

func DiskIO() {
	pm := openPm()
	pm.addDiskCounter()

	pm.query()
	time.Sleep(1 * time.Second)
	pm.query()

	read, write := pm.value()
	fmt.Println(" Total Read:", Size(uint64(read)), " Total Write:", Size(uint64(write)))
}

func PrintDisks() {
	if ShowFlag != 0 && ShowFlag&SHOW_DISK == 0 {
		return
	}
	fmt.Println(" 磁盘:")
	DiskIO()
	for _, v := range Disks() {
		fmt.Println("  ", v.Path, "  Total:", Size(v.Total), "  Free:", Size(v.Free))
	}

}

//CPU
func CountSetBits(bitMask [10]uint32) int {
	LSHIFT := uint32(unsafe.Sizeof(bitMask))*8 - 1
	bitSetCount := 0
	bitTest := uint32(1 << LSHIFT)

	var i uint32
	for i = 0; i <= LSHIFT; i++ {
		bitSetCount = 0
		if bitMask[0]&bitTest != 0 {
			bitSetCount = 1
		}
		bitTest /= 2
	}

	return int(bitSetCount)
}
func CPUInfo() {
	lpBuffer := make([]byte, 254)
	var ln win.DWORD
	ret := win.GetLogicalProcessorInformationEx(win.RelationAll, win.LPWSTR(unsafe.Pointer(&lpBuffer[0])), &ln)
	if ret == 0 {
		lpBuffer = make([]byte, ln)
		ret = win.GetLogicalProcessorInformationEx(win.RelationAll, win.LPWSTR(unsafe.Pointer(&lpBuffer[0])), &ln)
	}

	numaNodeCount := 0
	processorCoreCount := 0
	processorPackageCount := 0
	logicalProcessorCount := 0
	n := 0
	var c1, c2, c3 win.DWORD
	var info *win.SYSTEM_LOGICAL_PROCESSOR_INFORMATION_EX
	for i := 0; i < int(ln); i += int(info.Size) {
		info = (*win.SYSTEM_LOGICAL_PROCESSOR_INFORMATION_EX)(unsafe.Pointer(&lpBuffer[i]))
		//fmt.Printf("[%d]:%+v\n", n, info.Relationship)
		n++

		switch info.Relationship {
		case win.RelationNumaNode:
			numaNodeCount++
			//node := (*NUMA_NODE_RELATIONSHIP)(unsafe.Pointer(&info.RelShip[0]))
			//fmt.Println("node:", node)
		case win.RelationProcessorCore:
			processorCoreCount++
			proc := (*win.PROCESSOR_RELATIONSHIP)(unsafe.Pointer(&info.RelShip[0]))
			//fmt.Println("proc:", proc.GroupCount)
			logicalProcessorCount += CountSetBits(proc.GroupMask[0].Mask)
		case win.RelationProcessorPackage:
			//proc := (*PROCESSOR_RELATIONSHIP)(unsafe.Pointer(&info.RelShip[0]))
			//fmt.Println("proc1:", proc.GroupCount)
			processorPackageCount++
		case win.RelationCache:
			cache := (*win.CACHE_RELATIONSHIP)(unsafe.Pointer(&info.RelShip[0]))
			if cache.Level == 1 {
				c1 += cache.LineSize
			} else if cache.Level == 2 {
				c2 += cache.LineSize
			} else if cache.Level == 3 {
				c3 += cache.LineSize
			}
		case win.RelationGroup:
		default:
		}
	}
	fmt.Println(" CPU:")
	fmt.Println("  Sockets:", numaNodeCount, "  Cores:", processorCoreCount)
	fmt.Println("  L1/L2/L3 caches:", Size(uint64(c1)), "/", Size(uint64(c2)), "/", Size(uint64(c3)))
	//fmt.Println("  logicalProcessorCount:", logicalProcessorCount)
}

func SystemInfo() {
	host, _ := os.Hostname()
	version, err := syscall.GetVersion()
	if err != nil {
		fmt.Println(err)
		return
	}

	v := fmt.Sprintf("%d.%d (%d)", byte(version), uint8(version>>8), version>>16)
	fmt.Println(host, ":")
	fmt.Println(" 系统:", runtime.GOOS, "CPU架构:", runtime.GOARCH, "内部系统版本:", v)
}

func diffFileTime(time1, time2 win.FILETIME) uint32 {
	a := time1.DwHighDateTime<<32 | time1.DwLowDateTime
	b := time2.DwHighDateTime<<32 | time2.DwLowDateTime

	return b - a

}

type TimeStat struct {
	User   win.FILETIME
	Kernel win.FILETIME
	Idle   win.FILETIME
}

func CpuTimes() TimeStat {
	var t TimeStat

	win.GetSystemTimes(&t.Idle, &t.Kernel, &t.User)
	return t
}

func CpuUsage(t1, t2 TimeStat) {
	idle := diffFileTime(t1.Idle, t2.Idle)
	kernel := diffFileTime(t1.Kernel, t2.Kernel)
	user := diffFileTime(t1.User, t2.User)

	cpuUsage := 0.0
	cpu_idle_rate := 0.0
	cpu_kernel_rate := 0.0
	cpu_user_rate := 0.0

	if (kernel + user) != 0 {
		use := (kernel + user - idle) * 100 / (kernel + user) * uint32(runtime.NumCPU())
		cpuUsage = math.Abs(float64(use))

		//空闲时间 / 总的时间 = 闲置CPU时间的比率，即闲置率
		cpu_idle_rate = math.Abs(float64(idle * 100 / (kernel + user)))

		//核心态时间 / 总的时间 = 核心态占用的比率
		cpu_kernel_rate = math.Abs(float64(kernel * 100 / (kernel + user)))

		//用户态时间 / 总的时间 = 用户态占用的比率
		cpu_user_rate = math.Abs(float64(user * 100 / (kernel + user)))
	}
	fmt.Println(" CPU使用率:")
	fmt.Println("  CPU:", cpuUsage, "%  sys:", cpu_kernel_rate, "%  user:", cpu_user_rate, "%  idle:", cpu_idle_rate, "%")
}

func PrintCpuUsage() {
	if ShowFlag != 0 && ShowFlag&SHOW_CPU == 0 {
		return
	}
	CPUInfo()
	t1 := CpuTimes()
	time.Sleep(500 * time.Millisecond)
	t2 := CpuTimes()
	CpuUsage(t1, t2)
}

type NetByte struct {
	BytesSent uint64
	BytesRecv uint64
}

func NetMonitor(Index int, Name string) NetByte {
	var r NetByte

	entry := new(win.MIB_IFROW)
	entry.DwIndex = win.IF_INDEX(Index)
	win.GetIfEntry(entry)

	if entry.DwType == 6 || entry.DwType == 71 {
		r.BytesRecv += uint64(entry.DwInOctets)
		r.BytesSent += uint64(entry.DwOutOctets)
	}

	return r
}
func B2S(bs []win.UCHAR) string {
	ba := []byte{}
	for _, b := range bs {
		ba = append(ba, byte(b))
	}
	return string(ba)
}

func NetFiltered(name string) bool {
	return strings.Contains(name, "Loopback") ||
		strings.Contains(name, "isatap") ||
		strings.Contains(name, "VirtualBox") ||
		strings.Contains(name, "vEthernet")
}

type NetWork struct {
	Index       int
	Name        string
	MacAddr     string
	SendRate    uint64
	RecvRate    uint64
	IPs         []string
	lastNetByte NetByte
}

func NetworkInfo() []NetWork {
	intf, err := net.Interfaces()
	if err != nil {
		fmt.Println("Get network info failed: ", err)
		return nil
	}

	nets := make([]NetWork, 0, len(intf))
	for _, v := range intf {
		ips, err := v.Addrs()
		if err != nil {
			fmt.Println("Get network addr failed: ", err)
			return nil
		}

		if NetFiltered(v.Name) {
			continue
		}

		var n NetWork
		n.Index = v.Index
		n.Name = v.Name
		n.MacAddr = v.HardwareAddr.String()
		n.IPs = make([]string, 0, len(ips))
		for _, ip := range ips {
			a, _, _ := net.ParseCIDR(ip.String())
			n.IPs = append(n.IPs, a.String())
		}

		nets = append(nets, n)
	}

	return nets
}
func PrintNetIO() {
	if ShowFlag != 0 && ShowFlag&SHOW_NET == 0 {
		return
	}
	NetWork := NetworkInfo()

	for i, n := range NetWork {
		r := NetMonitor(n.Index, n.Name)
		NetWork[i].lastNetByte = r
	}
	time.Sleep(1 * time.Second)
	fmt.Println(" 网络:")
	for _, n := range NetWork {
		r := NetMonitor(n.Index, n.Name)
		SendRate := r.BytesSent - n.lastNetByte.BytesSent
		RecvRate := r.BytesRecv - n.lastNetByte.BytesRecv
		fmt.Println("  Name:", n.Name, " Send:", Size(SendRate), " Recv:", Size(RecvRate))
		fmt.Println("    IP:", n.IPs, " mac:", n.MacAddr)
	}
}

func Size(size uint64) string {
	s := float64(size)
	d := "B"

	if s > 1024 {
		s = s / 1024.0
		d = "K"
	}

	if s > 1024 {
		s = s / 1024.0
		d = "M"
	}
	if s > 1024 {
		s = s / 1024.0
		d = "G"
	}
	return fmt.Sprintf("%.2f %s", s, d)
}
