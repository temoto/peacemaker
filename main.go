package main

import (
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

var pollInterval = 1666 * time.Millisecond

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

func step() {
	ps, err := ProcSource("/proc").List()
	if err != nil {
		log.Fatalln(err)
	}

	mi, err := ProcSource("/proc").Meminfo()
	if err != nil {
		log.Fatalln(err)
	}

	availableRatio := float32(mi["MemAvailable"]) / float32(mi["MemTotal"])
	log.Printf("Available memory ratio: %.2f\n", availableRatio)

	if mi["MemAvailable"] < (100*MiB) || availableRatio < 0.05 {
		log.Println("Memory limit")
		victim := chooseVictimByMemory(ps)
		log.Printf("  going to kill %s pid=%d memory=%d (%.1f%%)\n",
			victim.Name, victim.Pid, victim.MemReal/MiB, float32(victim.MemReal)*100.0/float32(mi["MemTotal"]))
		victim.Terminate()
	}
}

func main() {
	for {
		step()
		time.Sleep(pollInterval)
	}
}
