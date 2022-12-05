package boomer

import (
	"log"
	"time"

	"github.com/shirou/gopsutil/mem"
	"github.com/shirou/gopsutil/v3/cpu"
)

type Storage struct {
	Name       string
	FileSystem string
	Total      uint64
	Free       uint64
}

type storageInfo struct {
	Name       string
	Size       uint64
	FreeSpace  uint64
	FileSystem string
}

//func getStorageInfo() {
//	var storageinfo []storageInfo
//	var loaclStorages []Storage
//	err := wmi.Query("Select * from Win32_LogicalDisk", &storageinfo)
//	if err != nil {
//		return
//	}
//
//	for _, storage := range storageinfo {
//		info := Storage{
//			Name:       storage.Name,
//			FileSystem: storage.FileSystem,
//			Total:      storage.Size / 1024 / 1024 / 1024,
//			Free:       storage.FreeSpace / 1024 / 1024 / 1024,
//		}
//		if info.Total >= 1 {
//			fmt.Printf("%s总大小%dG，可用%dG\n", info.Name, info.Total, info.Free)
//			loaclStorages = append(loaclStorages, info)
//		}
//	}
//	//fmt.Printf("localStorages:= %v\n", loaclStorages)
//}

type ComputerMonitor struct {
	CPU float64 `json:"cpu"`
	Mem float64 `json:"mem"`
}

// GetCPUPercent 获取CPU使用率
func GetCPUPercent() float64 {
	percent, err := cpu.Percent(time.Second, false)
	if err != nil {
		log.Fatalln(err.Error())
		return -1
	}
	return percent[0]
}

// GetMemPercent 获取内存使用率
func GetMemPercent() float64 {
	memInfo, err := mem.VirtualMemory()
	if err != nil {
		log.Fatalln(err.Error())
		return -1
	}
	return memInfo.UsedPercent
}

func GetCpuMem() ComputerMonitor {
	var res ComputerMonitor
	res.CPU = GetCPUPercent()
	res.Mem = GetMemPercent()
	//fmt.Printf("%v", res)
	//fmt.Printf("cpu使用率：%.2f%%\n", res.CPU)
	//fmt.Printf("内存使用率：%.2f%%\n", res.Mem)
	return res
}
