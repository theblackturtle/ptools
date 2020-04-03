package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
)

func main() {
	flag.Parse()
	if flag.NArg() != 2 {
		fmt.Fprintln(os.Stderr, "Check your input again")
		os.Exit(1)
	}
	file1 := flag.Arg(0)
	file2 := flag.Arg(1)

	f1, err := os.Open(file1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open %s: %s\n", file1, err)
		os.Exit(1)
	}
	defer f1.Close()

	f2, err := os.Open(file2)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open %s: %s\n", file2, err)
		os.Exit(1)
	}
	defer f2.Close()

	ipsMap := make(map[string][]string)
	sc2 := bufio.NewScanner(f2)
	for sc2.Scan() {
		line := strings.TrimSpace(sc2.Text())
		if line == "" {
			continue
		}
		lineArgs := strings.SplitN(line, ":", 2)
		if len(lineArgs) != 2{continue}
		ipAddress := lineArgs[0]
		port := lineArgs[1]
		if _, ok := ipsMap[ipAddress]; ok {
			ipsMap[ipAddress] = append(ipsMap[ipAddress], port)
		} else {
			ipsMap[ipAddress] = []string{port}
		}
	}

	sc1 := bufio.NewScanner(f1)
	for sc1.Scan() {
		line := strings.TrimSpace(sc1.Text())
		if line == "" {
			continue
		}
		lineArgs := strings.SplitN(line, ",", 2)
		if len(lineArgs) != 2{continue}
		domainIp := lineArgs[1]
		ports := ipsMap[domainIp]
		for _, p := range ports {
			fmt.Printf("%s:%v\n", lineArgs[0], p)
		}
	}
}
