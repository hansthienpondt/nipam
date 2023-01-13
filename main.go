package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/netip"
	"os"
	"strconv"
	"strings"

	"github.com/hansthienpondt/nipam/pkg/table"
	log "github.com/sirupsen/logrus"
	"go4.org/netipx"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

func init() {
	// Log as JSON instead of the default ASCII formatter.
	//log.SetFormatter(&log.JSONFormatter{})
	log.SetFormatter(&log.TextFormatter{})
	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	log.SetLevel(log.DebugLevel)
	//log.SetLevel(log.WarnLevel)
	// Set Calling method to true
	log.SetReportCaller(true)
}
func main() {
	flag.Parse()
	log.Debugf("Program Initialized")

	rtable := table.NewRIB()
	fmt.Println(rtable.Get(netip.MustParsePrefix("192.168.0.0/24")))
	cidrs := map[string]map[string]string{
		"10.0.0.0/8": {
			"description": "rfc1918",
		},
		"10.0.0.0/16": {
			"description": "10.0/16-subnet",
		},
		"10.1.0.0/16": {
			"description": "10.1/16-subnet",
		},
		"10.0.0.0/24": {
			"description": "10.0.0/24-subnet",
		},
		"10.0.1.0/24": {
			"description": "10.0.1/24-subnet",
		},
		"192.0.0.0/12": {
			"description": "test1",
			"rir":         "RIPE",
		},
		"192.168.0.0/16": {
			"description": "test2",
			"type":        "aggregate",
		},
		"192.169.0.0/16": {
			"description": "test3",
			"type":        "aggregate",
		},
		"192.168.0.0/24": {
			"description": "test4",
			"type":        "prefix",
		},
		"192.168.0.0/25": {
			"type":        "prefix",
			"description": "hans1",
		},
		"192.168.0.128/25": {
			"type":        "prefix",
			"description": "hans2",
		},
		"85.255.192.0/12": {
			"type":        "prefix",
			"rir":         "RIPE",
			"description": "test5",
		},
		"100.255.254.1/31": {
			"type":        "prefix",
			"rir":         "RIPE",
			"description": "test6",
		},
		"100.255.254.1/35": {
			"type":        "prefix",
			"rir":         "RIPE",
			"description": "test6",
		},
		"2a02:1800::/24": {
			"type":        "prefix",
			"rir":         "RIPE",
			"family":      "ipv6",
			"description": "test7",
		},
	}

	for k, v := range cidrs {
		p, err := netip.ParsePrefix(k)
		if err != nil {
			log.Errorf("Error parsing, skipping %s with error %v", k, err)
			continue
		}
		r := table.NewRoute(p, v, nil)
		//r = r.UpdateLabel(v)
		rtable.Add(r)
	}

	// Printing the size of the Radix/Patricia tree
	fmt.Println("The tree contains", rtable.Size(), "prefixes")
	// Printing the Stdout seperator
	fmt.Println(strings.Repeat("#", 64))

	for _, h := range rtable.GetTable() {
		fmt.Println(h)
	}

	fmt.Println(strings.Repeat("#", 64))

	// Inserting an additional single route with a label.
	ipam2Route := table.NewRoute(netip.MustParsePrefix("192.168.0.192/26"), nil, nil)
	//ipam2Route := table.NewRoute(netaddr.MustParseIPPrefix("192.168.0.128/25"))
	ipam2Route = ipam2Route.UpdateLabel(map[string]string{"foo": "bar", "boo": "hoo"})
	if err := rtable.Add(ipam2Route); err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("Adding CIDR %q with labels %q to the IPAM table\n", ipam2Route.Prefix(), ipam2Route.Labels())
		fmt.Printf("CIDR %q has label foo: %t\n", ipam2Route.Prefix(), ipam2Route.Has("foo"))
	}
	// Printing the Stdout seperator
	fmt.Println(strings.Repeat("#", 64))

	// Lookup Methods in the routing table.
	route1, _ := netip.ParsePrefix("192.168.0.255/32")
	fmt.Println("Finding the parents for route -- " + route1.String())
	// Marshal it as a JSON.
	c, _ := json.MarshalIndent(rtable.Parents(route1), "", "  ")
	fmt.Println(string(c))

	route2, _ := netip.ParsePrefix("10.0.0.0/16")
	fmt.Println("Finding the children for route -- " + route2.String())
	d := rtable.Children(route2)
	fmt.Println(d)
	// Printing the Stdout seperator
	fmt.Println(strings.Repeat("#", 64))

	// Find free prefixes within a certain prefix
	findfree := netip.MustParsePrefix("10.0.0.0/16")
	var bitlen uint8 = 19
	fmt.Println("Finding a free /" + strconv.Itoa(int(bitlen)) + " prefix in: " + findfree.String())
	pfx := rtable.GetAvailablePrefixes(findfree)
	fmt.Printf("All free prefixes are: %s\n", pfx)

	pfx2 := rtable.GetAvailablePrefixByBitLen(findfree, bitlen)
	fmt.Println("Returned free prefix is: " + pfx2.String())

	//Alternate method: get Route object, search for free prefixes
	//lpm := rtable.LPM(findfree)
	//pfx3 := lpm.GetAvailablePrefixByBitLen(rtable, bitlen)
	//fmt.Println("Alt Method, Returned free prefix is: " + pfx3.String())

	// Printing the Stdout seperator
	fmt.Println(strings.Repeat("#", 64))

	// Create a selector (label filter) to get routes by label
	selector := labels.NewSelector()
	req, _ := labels.NewRequirement("type", selection.NotIn, []string{"aggregate"})
	selector = selector.Add(*req)
	// Alternate definition of a selector, define by string.
	sel, _ := labels.Parse("type notin (aggregate), description!=hans1")

	dumprtable1 := rtable.GetByLabel(selector)
	dumprtable2 := rtable.GetByLabel(sel)

	fmt.Println("Printing GetByLabel1 -- " + selector.String())
	for _, v := range dumprtable1 {
		fmt.Println(v.Prefix(), v.Labels())
	}
	fmt.Println("")
	fmt.Println("Printing GetByLabel2 -- " + sel.String())
	for _, v := range dumprtable2 {
		fmt.Println(v.Prefix(), v.Labels())
	}
	// Printing the Stdout seperator
	fmt.Println(strings.Repeat("#", 64))

	tst := netip.MustParsePrefix("192.122.0.0/24")

	fmt.Println(netipx.PrefixLastIP(tst))

	fR := netip.MustParsePrefix("192.168.0.0/25")
	myR, _ := rtable.Get(fR)

	myR = myR.UpdateLabel(map[string]string{"key": "added", "adjustments": "made"})
	rtable.Set(myR)

	for _, r := range rtable.GetTable() {
		fmt.Println(r)
	}
}
