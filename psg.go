// psg.go
package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"unicode/utf8"
	"unsafe"

	"github.com/disiqueira/gotree"
)

type ulong int32
type ulong_ptr uintptr

type PROCESSENTRY32 struct {
	dwSize              ulong
	cntUsage            ulong
	th32ProcessID       ulong
	th32DefaultHeapID   ulong_ptr
	th32ModuleID        ulong
	cntThreads          ulong
	th32ParentProcessID ulong
	pcPriClassBase      ulong
	dwFlags             ulong
	szExeFile           [260]byte
}

const LARGE_BUFFER_SIZE = 256 * 1024 * 1024

var (
	kernel32                = syscall.NewLazyDLL("kernel32.dll")
	psapi                   = syscall.NewLazyDLL("psapi.dll")
	GetProcessImageFileName = psapi.NewProc("GetProcessImageFileNameA")
)

type Process struct {
	Name string
	Pid  int
	PPid int
}

type ProcSlice []Process

func (p ProcSlice) Len() int {
	return len(p)
}
func (p ProcSlice) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}
func (p ProcSlice) Less(i, j int) bool {
	return p[i].Pid < p[j].Pid
}

func main() {
	var (
		Name string
		Pid  int = -1
		PPid int = -1
		Tree bool
	)
	flag.IntVar(&Pid, "p", -1, "pid")
	flag.BoolVar(&Tree, "t", false, "tree")
	flag.Parse()
	procs := getAllProcess()

	if flag.NArg() > 0 {
		Name = flag.Arg(0)
		Ppid, err := strconv.Atoi(Name)
		if err == nil {
			PPid = Ppid
		}
	}
	if !sort.IsSorted(ProcSlice(procs)) {
		sort.Sort(ProcSlice(procs))
	}

	if Tree {
		PrintTree(procs, Pid)
		os.Exit(0)
	}

	fmt.Println("PID\t PPID\t CMD")
	for _, p := range procs {
		if Pid != -1 && p.Pid != Pid {
			continue
		}
		if flag.NArg() > 0 {
			if p.PPid == PPid || strings.Contains(p.Name, Name) {
				fmt.Println(p.Pid, "\t", p.PPid, "\t", p.Name)
			}
		} else {
			fmt.Println(p.Pid, "\t", p.PPid, "\t", p.Name)
		}
	}
}

type tree struct {
	pid   int
	name  string
	child []int
}

func (t tree) String() string {
	return fmt.Sprintf("(%d)%s", t.pid, t.name)
}

func (t *tree) Tree(Pmaps map[int]*tree) gotree.Tree {
	artist := gotree.New(t.String())
	for _, v := range t.child {
		if v1, ok := Pmaps[v]; ok {
			artist.AddTree(v1.Tree(Pmaps))
		}
	}
	return artist
}

func PrintTree(procs []Process, pid int) {
	Pmaps := make(map[int]*tree)
	h, _ := os.Hostname()
	host := tree{00, h, make([]int, 0, 20)}
	Pmaps[00] = &host
	for _, v := range procs {
		t := tree{v.Pid, v.Name, make([]int, 0, 20)}
		Pmaps[v.Pid] = &t
	}

	for _, v := range procs {
		v1, ok := Pmaps[v.PPid]
		if ok && v.Pid != v.PPid {
			v1.child = append(v1.child, v.Pid)
		} else {
			host.child = append(host.child, v.Pid)
		}
	}

	if pid != -1 {
		if v, ok := Pmaps[pid]; ok {
			fmt.Println(v.Tree(Pmaps).Print())
		}
	} else {
		fmt.Println(host.Tree(Pmaps).Print())
	}

}

func getAllProcess() []Process {
	snapshot := kernel32.NewProc("CreateToolhelp32Snapshot")
	pHandle, _, _ := snapshot.Call(uintptr(0x2), uintptr(0x0))
	if int(pHandle) == -1 {
		return nil
	}
	Process32Next := kernel32.NewProc("Process32Next")
	procs := make([]Process, 0, 300)
	for {
		var proc PROCESSENTRY32
		proc.dwSize = ulong(unsafe.Sizeof(proc))
		if rt, _, _ := Process32Next.Call(uintptr(pHandle), uintptr(unsafe.Pointer(&proc))); int(rt) == 1 {
			//name, err := queryProc(int(proc.th32ProcessID))
			//if err != nil {
			name := string(proc.szExeFile[0:23])
			//}
			procs = append(procs, Process{name, int(proc.th32ProcessID), int(proc.th32ParentProcessID)})
			//fmt.Println(name, ":", proc.th32ProcessID)
		} else {
			break
		}
	}
	CloseHandle := kernel32.NewProc("CloseHandle")
	_, _, _ = CloseHandle.Call(pHandle)

	return procs
}
func queryProc(pid int) (string, error) {
	var szExeFile [260]byte
	handle, err := syscall.OpenProcess(0X0400|0X0010, false, uint32(pid))
	if err != nil {
		//fmt.Println(err)
		return "", err
	}

	defer syscall.CloseHandle(handle)

	GetProcessImageFileName.Call(uintptr(handle), uintptr(unsafe.Pointer(&szExeFile)), 260)

	fullName := string(szExeFile[0:])
	name := PathMap[fullName[0:23]]
	result := strings.Replace(fullName, fullName[0:23], name, 23)

	return result, nil
}

/*func FullName() {
		GetProcessImageFileName := psapi.NewProc("GetProcessImageFileNameA")
	GetProcessImageFileName.Call(uintptr(handle), uintptr(unsafe.Pointer(&szExeFile)), 260)


mystring CCommon::DosDevicePath2LogicalPath(LPCTSTR lpszDosPath)
{
 mystring strResult;

 // Translate path with device name to drive letters.
 TCHAR szTemp[MAX_PATH];
 szTemp[0] = '\0';

 if ( lpszDosPath==NULL || !GetLogicalDriveStrings(_countof(szTemp)-1, szTemp) ){
  return strResult;
 }


 TCHAR szName[MAX_PATH];
 TCHAR szDrive[3] = TEXT(" :");
 BOOL bFound = FALSE;
 TCHAR* p = szTemp;

 do{
  // Copy the drive letter to the template string
  *szDrive = *p;

  // Look up each device name
  if ( QueryDosDevice(szDrive, szName, _countof(szName)) ){
   UINT uNameLen = (UINT)_tcslen(szName);

   if (uNameLen < MAX_PATH)
   {
    bFound = _tcsnicmp(lpszDosPath, szName, uNameLen) == 0;

    if ( bFound ){
     // Reconstruct pszFilename using szTemp
     // Replace device path with DOS path
     TCHAR szTempFile[MAX_PATH];
     _stprintf_s(szTempFile, TEXT("%s%s"), szDrive, lpszDosPath+uNameLen);
     strResult = szTempFile;
    }
   }
  }

  // Go to the next NULL character.
  while (*p++);
 } while (!bFound && *p); // end of string

 return strResult;
}
}*/
var PathMap map[string]string

func getDiskInfo() {
	PathMap = make(map[string]string)
	GetLogicalDriveStringsW := kernel32.NewProc("GetLogicalDriveStringsW")
	lpBuffer := make([]byte, 254)
	lpBuffer1 := make([]byte, 100)
	diskret, _, _ := GetLogicalDriveStringsW.Call(
		uintptr(len(lpBuffer)),
		uintptr(unsafe.Pointer(&lpBuffer[0])))
	if diskret == 0 {
		return
	}

	QueryDosDeviceW := kernel32.NewProc("QueryDosDeviceW")
	for _, v := range lpBuffer {
		if v >= 65 && v <= 90 {

			path := string(v) + ":"
			if path == "A:" || path == "B:" {
				continue
			}

			_, _, e := QueryDosDeviceW.Call(uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(path))),
				uintptr(unsafe.Pointer(&lpBuffer1[0])),
				uintptr(len(lpBuffer1)))
			if e != nil {
				//fmt.Println(e)
			}
			//fmt.Println(path, "=", string(lpBuffer1[0:]), len(lpBuffer1[0:]))
			a := TrimAll(lpBuffer1[0:], "\x00")
			//fmt.Println(path, "=", string(a), len(a))
			PathMap[string(a)] = path
		}
	}

}
func TrimAll(s []byte, cutset string) []byte {
	return TrimAllFunc(s, makeCutsetFunc(cutset))
}

func TrimAllFunc(s []byte, f func(r rune) bool) []byte {
	n := make([]byte, 0, len(s))
	start := 0
	for start < len(s) {
		wid := 1
		r := rune(s[start])
		if r >= utf8.RuneSelf {
			r, wid = utf8.DecodeRune(s[start:])
		}
		if f(r) == false {
			n = append(n, s[start])
		}
		start += wid
	}
	return n
}
func makeCutsetFunc(cutset string) func(r rune) bool {
	if len(cutset) == 1 && cutset[0] < utf8.RuneSelf {
		return func(r rune) bool {
			return r == rune(cutset[0])
		}
	}
	if as, isASCII := makeASCIISet(cutset); isASCII {
		return func(r rune) bool {
			return r < utf8.RuneSelf && as.contains(byte(r))
		}
	}
	return func(r rune) bool {
		for _, c := range cutset {
			if c == r {
				return true
			}
		}
		return false
	}
}

type asciiSet [8]uint32

// makeASCIISet creates a set of ASCII characters and reports whether all
// characters in chars are ASCII.
func makeASCIISet(chars string) (as asciiSet, ok bool) {
	for i := 0; i < len(chars); i++ {
		c := chars[i]
		if c >= utf8.RuneSelf {
			return as, false
		}
		as[c>>5] |= 1 << uint(c&31)
	}
	return as, true
}

// contains reports whether c is inside the set.
func (as *asciiSet) contains(c byte) bool {
	return (as[c>>5] & (1 << uint(c&31))) != 0
}
