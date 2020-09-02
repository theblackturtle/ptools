package main

import (
    "bufio"
    "fmt"
    "net"
    "net/url"
    "os"
    "strings"
)

// Get from AMASS by @caffix

// ReservedCIDRs includes all the networks that are reserved for special use.
var ReservedCIDRs = []string{
    "192.168.0.0/16",
    "172.16.0.0/12",
    "10.0.0.0/8",
    "127.0.0.0/8",
    "224.0.0.0/4",
    "240.0.0.0/4",
    "100.64.0.0/10",
    "198.18.0.0/15",
    "169.254.0.0/16",
    "192.88.99.0/24",
    "192.0.0.0/24",
    "192.0.2.0/24",
    "192.94.77.0/24",
    "192.94.78.0/24",
    "192.52.193.0/24",
    "192.12.109.0/24",
    "192.31.196.0/24",
    "192.0.0.0/29",
}

// The reserved network address ranges
var reservedAddrRanges []*net.IPNet

func init() {
    for _, cidr := range ReservedCIDRs {
        if _, ipnet, err := net.ParseCIDR(cidr); err == nil {
            reservedAddrRanges = append(reservedAddrRanges, ipnet)
        }
    }
}

// IsReservedAddress checks if the addr parameter is within one of the address ranges in the ReservedCIDRs slice.
func IsReservedAddress(addr string) (bool, string) {
    ip := net.ParseIP(addr)
    if ip == nil {
        return false, ""
    }

    var cidr string
    for _, block := range reservedAddrRanges {
        if block.Contains(ip) {
            cidr = block.String()
            break
        }
    }

    if cidr != "" {
        return true, cidr
    }
    return false, ""
}

func main() {
    sc := bufio.NewScanner(os.Stdin)
    for sc.Scan() {
        line := strings.TrimSpace(sc.Text())
        if line == "" {
            continue
        }
        var ipAddress string
        if strings.HasPrefix(line, "http") {
            if u, err := url.Parse(line); err == nil {
                ipAddress = u.Hostname()
            }
        } else {
            ipAddress = line
        }
        if private, _ := IsReservedAddress(ipAddress); !private {
            fmt.Println(line)
        }
    }
}
