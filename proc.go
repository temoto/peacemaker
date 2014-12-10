package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type ProcSource string

func (root ProcSource) List() ([]*Process, error) {
	fs, err := ioutil.ReadDir(string(root))
	if err != nil {
		return nil, err
	}

	var n int64
	var p *Process
	result := make([]*Process, 0, 10000)
	for _, fi := range fs {
		if !fi.IsDir() {
			continue
		}
		n, err = strconv.ParseInt(fi.Name(), 10, 32)
		if err != nil {
			continue
		}

		p = &Process{Pid: uint(n)}
		err = readStatus(fmt.Sprintf("%s/%d/status", root, n), p)
		if err == errSkip {
			continue
		}
		if err != nil {
			log.Printf("readStatus: %s\n", err)
			continue
		}
		if p.Pid == 0 {
			log.Printf("readStatus did not parse pid: %s/%d", root, n)
		}
		// Skip threads
		if p.Pid != uint(n) {
			continue
		}

		err = readStat(fmt.Sprintf("%s/%d/stat", root, n), p)
		if err == errSkip {
			continue
		}
		if err != nil {
			log.Printf("readStat: %s\n", err)
			continue
		}

		result = append(result, p)
	}
	return result, nil
}

var errInvalidStatContent = errors.New("Invalid proc/stat content")
var errSkip = errors.New("Skip valid proc/stat")

// pid (comm) state ppid pgrp session tty_nr             1-7
// tpgid flags minflt cminflt majflt cmajflt             8-13
// utime stime cutime cstime priority nice              14-19
// _ itrealvalue starttime vsize rss rlim               20-25
// startcode endcode startstack kstkesp kstkeip signal
// blocked sigignore sigcatch wchan nswap cnswap
// exit_signal processor
var reStat = regexp.MustCompile(`^` +
	`(\d+) \((.*)\) ([RSDZTW]) (\d+) (\d+) (\d+) (\d+) ` +
	`(-?\d+) (\d+) (\d+) (\d+) (\d+) (\d+) (\d+) (\d+) (\d+) (\d+) (-?\d+) (-?\d+) ` +
	`(-?\d+) (\d+) (\d+) (\d+) (\d+) (\d+) ` +
	`(\d+) (\d+) (\d+) (\d+) (\d+) (\d+) ` +
	`(\d+) (\d+) (\d+) (\d+) (\d+) (\d+) ` +
	`(\d+) (\d+).*`)

func readStat(path string, p *Process) error {
	// TODO: handle file open error
	// TODO: limit read
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	matches := reStat.FindSubmatch(b)
	if len(matches) < 26 {
		log.Printf("readStat %s: invalid: %q\n", path, b)
		return errInvalidStatContent
	}
	p.Name = string(matches[2])
	// if string(matches[3]) != "R" && strings.Index(p.Name, "chrome") == -1 {
	// 	return errSkip
	// }
	// log.Printf("readStat %s: %q\n", path, b)
	var i int64
	i, err = strconv.ParseInt(string(matches[4]), 10, 64)
	if err != nil {
		return err
	}
	p.ParentPid = uint(i)
	i, err = strconv.ParseInt(string(matches[14]), 10, 64)
	if err != nil {
		return err
	}
	p.TimeUser = time.Duration(i*1000000/int64(Sysconf_SC_CLK_TCK)) * time.Microsecond
	i, err = strconv.ParseInt(string(matches[15]), 10, 64)
	if err != nil {
		return err
	}
	p.TimeSystem = time.Duration(i*1000000/int64(Sysconf_SC_CLK_TCK)) * time.Microsecond
	p.TimeReal = p.TimeUser + p.TimeSystem
	return nil
}

func parseSize(s string) (int64, error) {
	parts := strings.SplitN(strings.TrimSpace(s), " ", 2)
	n, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, err
	}
	if len(parts) == 1 {
		return n, nil
	}
	mod := parts[1]
	if mod == "kB" {
		n *= 1024
	} else {
		return 0, errors.New("parseSize: invalid input: " + s)
	}
	return n, nil
}

func readStatus(path string, p *Process) error {
	// TODO: handle file open error
	// TODO: limit read
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	s := string(b)
	lines := strings.Split(s, "\n")
	var parts []string
	var key string
	var i64 int64
	for _, line := range lines {
		parts = strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key = strings.TrimSpace(parts[0])
		i64, err = parseSize(parts[1])
		if key == "Tgid" && err == nil {
			p.Pid = uint(i64)
		} else if key == "PPid" && err == nil {
			p.ParentPid = uint(i64)
		} else if key == "VmSize" && err == nil {
			p.MemVirtual = uint64(i64)
		} else if key == "VmRSS" && err == nil {
			p.MemReal = uint64(i64)
		}
	}

	return nil
}

type Meminfo map[string]int64

func (root ProcSource) Meminfo() (Meminfo, error) {
	path := fmt.Sprintf("%s/meminfo", root)

	// TODO: handle file open error
	// TODO: limit read
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	s := string(b)
	lines := strings.Split(s, "\n")
	var parts []string
	var key string
	var n int64
	result := make(Meminfo)
	for _, line := range lines {
		parts = strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key = strings.TrimSpace(parts[0])
		n, err = parseSize(parts[1])
		if err != nil {
			continue
		}
		result[key] = n
	}

	return result, nil
}
