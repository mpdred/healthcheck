package main

import (
	"log"
	"time"

	"github.com/mpdred/healthcheck/v2/pkg/healthcheck"
	"github.com/prometheus/client_golang/prometheus"
)

func main() {
	log.Println("create health check server...")
	e := healthcheck.NewExecutor()
	h := healthcheck.NewHandler(9999, e, "mynamespace", prometheus.NewRegistry())

	log.Println("start healthcheck server...")
	go h.Start()
	defer h.Stop()

	// Create some probes

	components := []string{"foo", "bar", "baz"}
	componentsStatus := map[string]bool{
		"foo": true,
		"bar": false,
		"baz": true,
	}

	probes := healthcheck.NewProbeBuilder().BuildForComponents(healthcheck.Readiness, components, componentsStatus)

	// Register the probes
	h.RegisterProbes(probes...)

	log.Println("doing other things...")
	time.Sleep(5 * time.Minute)
	log.Println("main() finished")

	// $ curl localhost:9999/health | jq .
	//  % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
	//                                 Dload  Upload   Total   Spent    Left  Speed
	// 100   211  100   211    0     0  40053      0 --:--:-- --:--:-- --:--:--  206k
	// [
	//  {
	//    "probe": {
	//      "kind": "readiness",
	//      "name": "component baz"
	//    }
	//  },
	//  {
	//    "probe": {
	//      "kind": "readiness",
	//      "name": "component foo"
	//    }
	//  },
	//  {
	//    "probe": {
	//      "kind": "readiness",
	//      "name": "component bar"
	//    },
	//    "err": "readiness for component set to 'false'"
	//  }
	// ]
	//
	// $ curl localhost:9999/metrics | grep component
	//  % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current
	//                                 Dload  Upload   Total   Spent    Left  Speed
	// 100  5166    0  5166    0     0  1167k      0 --:--:-- --:--:-- --:--:-- 5044k
	// mynamespace_healthcheck_status{kind="readiness",probe="component bar"} 1
	// mynamespace_healthcheck_status{kind="readiness",probe="component baz"} 0
	// mynamespace_healthcheck_status{kind="readiness",probe="component foo"} 0
}
