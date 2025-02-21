package util

import (
	"bufio"
	"fmt"
	"go.uber.org/zap"
	"os"
	"runtime"
	"strconv"
	"strings"
)

func LogMem() {
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	fmt.Printf("Total allocated memory: %d bytes\n", mem.TotalAlloc)
	fmt.Printf("Number of memory allocations: %d\n", mem.Mallocs)
}

func LogMemS(log *zap.SugaredLogger, location string) {
	return
	//var mem runtime.MemStats
	//runtime.ReadMemStats(&mem)
	log.Info("-----------------")
	log.Infof("Memory usage at %s", location)
	//log.Infof("Alloc = %.2f MiB", bToMb(mem.Alloc))
	//log.Infof("Total allocated memory: %.2f MiB", bToMb(mem.TotalAlloc))
	//log.Infof("Sys = %.2f MiB", bToMb(mem.Sys))
	//log.Infof("NumGC = %v", mem.NumGC)
	mem := readMemoryStats()
	log.Infof("MemTotal = %s", mem)
}

func bToMb(b uint64) float64 {
	return float64(b / 1024 / 1024)
}

type Memory struct {
	MemTotal     int
	MemFree      int
	MemAvailable int
}

func (m Memory) String() string {
	return fmt.Sprintf(
		"Memory{MemTotal: %.2f MiB, MemFree: %.2f MiB, MemAvailable: %.2f MiB}",
		bToMb(uint64(m.MemTotal)), bToMb(uint64(m.MemFree)), bToMb(uint64(m.MemAvailable)),
	)
}

func readMemoryStats() Memory {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	bufio.NewScanner(file)
	scanner := bufio.NewScanner(file)
	res := Memory{}
	for scanner.Scan() {
		key, value := parseLine(scanner.Text())
		switch key {
		case "MemTotal":
			res.MemTotal = value
		case "MemFree":
			res.MemFree = value
		case "MemAvailable":
			res.MemAvailable = value
		}
	}
	return res
}

func parseLine(raw string) (key string, value int) {
	text := strings.ReplaceAll(raw[:len(raw)-2], " ", "")
	keyValue := strings.Split(text, ":")
	return keyValue[0], toInt(keyValue[1])
}

func toInt(raw string) int {
	if raw == "" {
		return 0
	}
	res, err := strconv.Atoi(raw)
	if err != nil {
		panic(err)
	}
	return res
}
