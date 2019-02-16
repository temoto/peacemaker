package main

import (
	"flag"
	"log"
	"time"
)

const MiB = 1 << 20

type Process struct {
	Pid        uint
	ParentPid  uint
	Name       string
	Cmdline    string
	TimeReal   time.Duration
	TimeUser   time.Duration
	TimeSystem time.Duration
	MemReal    uint64 // including shared
	MemShared  uint64
	MemPrivate uint64 // Real - shared
	MemVirtual uint64

	Source ProcessLister
}

type ProcessLister interface {
	List() ([]*Process, error)
}

func main() {
	flagDebug := flag.Bool("debug", false, "")
	pollInterval := flag.Duration("interval", 1666*time.Millisecond, "")
	limitMiB := flag.Float64("limit-mb", 100, "")
	limitPercent := flag.Float64("limit-percent", 5, "")
	flag.Parse()
	log.Printf("Configuration: pollInterval=%s limitMiB=%.1f limitPercent=%.2f%%\n",
		*pollInterval, *limitMiB, *limitPercent)

	for {
		step(*flagDebug, *limitMiB, *limitPercent)
		time.Sleep(*pollInterval)
	}
}

func chooseVictimByMemory(ps []*Process) *Process {
	if len(ps) == 0 {
		log.Println("No processes to select target from")
		return nil
	}

	var result *Process
	for _, p := range ps {
		if result == nil || p.MemReal > result.MemReal {
			result = p
		}
	}
	return result
}

func step(debug bool, limitMiB, limitPercent float64) {
	ps, err := ProcSource("/proc").List()
	if err != nil {
		log.Fatalln(err)
	}

	mi, err := ProcSource("/proc").Meminfo()
	if err != nil {
		log.Fatalln(err)
	}

	availableMiB := float64(mi["MemAvailable"]) / MiB
	totalMiB := float64(mi["MemTotal"]) / MiB
	availablePercent := availableMiB / totalMiB * 100
	if debug {
		log.Printf("Available memory: %.1f / %.1f MiB = %.1f%%\n",
			availableMiB, totalMiB, availablePercent)
	}

	if availableMiB < limitMiB || availablePercent < limitPercent {
		log.Println("Memory limit")
		victim := chooseVictimByMemory(ps)
		if victim == nil {
			log.Println("  failed to choose victim process to free memory")
		}
		log.Printf("  going to kill %s pid=%d memory=%d (%.1f%%)\n",
			victim.Name, victim.Pid, victim.MemReal/MiB, float32(victim.MemReal)*100.0/float32(mi["MemTotal"]))
		victim.Terminate()
	}
}
