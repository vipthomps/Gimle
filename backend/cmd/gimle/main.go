// @title Gimlé API
// @version 1.0.0
// @description Home lab dashboard: network discovery, a map of hosts, bookmarks, views and docs
// @contact.url   https://github.com/vipthomps/Gimle
// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT
// @BasePath /api/

package main

import (
	"flag"
	"fmt"
	// "net/http"

	// _ "net/http/pprof"

	// Import Swagger docs
	_ "github.com/vipthomps/gimle/backend/docs"

	"github.com/vipthomps/gimle/backend/internal/backup"
	"github.com/vipthomps/gimle/backend/internal/conf"
	"github.com/vipthomps/gimle/backend/internal/gdb"
	"github.com/vipthomps/gimle/backend/internal/routines"
	"github.com/vipthomps/gimle/backend/internal/web"
)

const dirPath = "/var/lib/gimle"
const nodePath = ""

func main() {
	dirPtr := flag.String("d", dirPath, "Path to config dir")
	nodePtr := flag.String("n", nodePath, "Path to node modules")
	versionPtr := flag.Bool("v", false, "Print the version and exit")
	flag.Parse()

	if *versionPtr {
		fmt.Println(web.Version())
		return
	}

	// pprof - memory leak detect
	// go tool pprof -alloc_space http://localhost:8085/debug/pprof/heap
	// (pprof) web
	// (pprof) list db.Select
	//
	// go func() {
	// 	http.ListenAndServe("localhost:8085", nil)
	// }()

	// Make AppConfig
	conf.Start(*dirPtr, *nodePtr)

	gdb.Start()
	routines.SeedSubnets()

	routines.ScanRestart()
	routines.HistoryTrim()
	routines.SeedFirstSeen()
	routines.StatsTrim()
	routines.PortScanRestart()
	routines.StartConnectors()
	backup.Start()

	web.Gui()
}
