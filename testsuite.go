package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	global "testsuite/src/global"
	"testsuite/src/modules"

	log "github.com/sirupsen/logrus"
)

// TIP <p>To run your code, right-click the code and select <b>Run</b>.</p> <p>Alternatively, click
// the <icon src="AllIcons.Actions.Execute"/> icon in the gutter and select the <b>Run</b> menu item from here.</p>
func main() {

	const (
		version = "0.0.1"
	)
	params := *global.GetParams()
	//initialize help
	help := new(global.HelpText)
	help.Init()

	//return version adn exit
	if len(os.Args) == 2 {
		if os.Args[1] == "--version" {
			fmt.Println("test suite version: ", version)
			exitWithCode(0)
		} else if os.Args[1] == "--help" {
			print(help.GetHelpText())
		}
	}

	//Manage config and parameters from conf file [start]
	//Mandatory from command line
	flag.StringVar(&params.Module, "module", "", "Name of the module to run IE staleRead")

	flag.StringVar(&params.Url, "url", params.Url, "Primary Database URL")
	flag.StringVar(&params.UrlRead, "url-read", params.UrlRead, "Reader Database URL for StaleReadTest")
	flag.StringVar(&params.User, "user", params.User, "Database user")
	flag.StringVar(&params.Password, "password", params.Password, "Database password")
	flag.StringVar(&params.Schema, "schema", params.Schema, "Database schema")
	flag.StringVar(&params.Attributes, "attributes", params.Attributes, "Connection attributes IE: &autoReconnect=true")

	flag.IntVar(&params.Loops, "loops", params.Loops, "Number of test loops to execute")
	flag.IntVar(&params.Sleep, "sleep", params.Sleep, "Sleep time in milliseconds")
	flag.IntVar(&params.RowsNumber, "rowsNumber", params.RowsNumber, "Number of rows to generate")

	flag.BoolVar(&params.Verbose, "verbose", params.Verbose, "Enable verbose output")
	flag.BoolVar(&params.Summary, "summary", params.Summary, "Print execution summary")
	flag.BoolVar(&params.ReportCSV, "reportCSV", params.ReportCSV, "Format output as CSV")
	flag.BoolVar(&params.PrintConnectionTime, "printConnectionTime", params.PrintConnectionTime, "Print connection times")
	flag.BoolVar(&params.PrintStatusDone, "printStatusDone", params.PrintStatusDone, "Print % process increase")

	flag.StringVar(&params.AwsMMSessionConsistencyLevel, "awsMMsessionConsistencyLevel", params.AwsMMSessionConsistencyLevel, "Consistency level (INSTANCE_RAW or REGIONAL_RAW)")

	// Execute the parsing
	flag.Parse()

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "\n%s\n", help.GetHelpText())
		flag.PrintDefaults()
	}
	flag.Parse()

	log.Infof(params.Module)
	if params.Module == "" {
		print(help.GetHelpText())
	}

	if strings.ToLower(params.Module) == "staleread" {
		staleReadTest := modules.NewStaleReadTest(params)
		staleReadTest.Run()

	}

}

func exitWithCode(errorCode int) {
	log.Debug("Exiting execution with code ", errorCode)
	os.Exit(errorCode)
}
