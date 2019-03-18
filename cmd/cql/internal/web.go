package internal

import (
	"net/http"

	"github.com/CovenantSQL/CovenantSQL/sqlchain/observer"
	"github.com/CovenantSQL/CovenantSQL/utils"
)

var (
	webAddr string // Web addr

	webService    *observer.Service
	webHTTPServer *http.Server
)

// CmdWeb is cql web command.
var CmdWeb = &Command{
	UsageLine:   "cql web [-tmp-path path] [-bg-log-level level] [address]",
	Description: "Web command serve a database chain explorer, e.g. :8546",
}

func init() {
	CmdWeb.Run = runWeb

	addCommonFlags(CmdWeb)
	addBgServerFlag(CmdWeb)
}

func startWebServer(webAddr string) func() {
	var err error
	webService, webHTTPServer, err = observer.StartObserver(webAddr, Version)
	if err != nil {
		ConsoleLog.WithError(err).Error("start explorer failed")
		SetExitStatus(1)
		return nil
	}

	ConsoleLog.Infof("web server started on %s", webAddr)

	return func() {
		_ = observer.StopObserver(webService, webHTTPServer)
		ConsoleLog.Info("explorer stopped")
	}
}

func runWeb(cmd *Command, args []string) {
	configInit()
	bgServerInit()

	if len(args) != 1 {
		ConsoleLog.Error("Web command need listern address as param")
		SetExitStatus(1)
		return
	}
	webAddr = args[0]

	cancelFunc := startWebServer(webAddr)
	ExitIfErrors()
	defer cancelFunc()

	ConsoleLog.Printf("Ctrl + C to stop web server on %s\n", webAddr)
	<-utils.WaitForExit()
}
