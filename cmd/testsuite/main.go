package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"testsuite/internal/config"
	"testsuite/internal/modules"

	log "github.com/sirupsen/logrus"
)

const version = "0.0.1"

func main() {
	params := *config.GetParams()
	help := new(config.HelpText)
	help.Init()

	if len(os.Args) == 2 {
		switch os.Args[1] {
		case "--version":
			fmt.Println("test suite version:", version)
			os.Exit(0)
		case "--help":
			fmt.Print(help.GetHelpText())
		}
	}

	flag.StringVar(&params.Module, "module", "", "Name of the module to run (staleread, connectionpool, datagen)")

	flag.StringVar(&params.Url, "url", params.Url, "Primary Database URL (host:port)")
	flag.StringVar(&params.UrlRead, "url-read", params.UrlRead, "Reader Database URL for StaleReadTest")
	flag.StringVar(&params.User, "user", params.User, "Database user")
	flag.StringVar(&params.Password, "password", params.Password, "Database password")
	flag.StringVar(&params.Schema, "schema", params.Schema, "Database schema")
	flag.StringVar(&params.Attributes, "attributes", params.Attributes, "Connection attributes (e.g. &autoReconnect=true)")

	flag.IntVar(&params.Loops, "loops", params.Loops, "Number of test loops to execute")
	flag.IntVar(&params.Sleep, "sleep", params.Sleep, "Sleep time in milliseconds between iterations")
	flag.IntVar(&params.RowsNumber, "rowsNumber", params.RowsNumber, "Number of rows to generate")

	flag.BoolVar(&params.Verbose, "verbose", params.Verbose, "Enable verbose output")
	flag.BoolVar(&params.Summary, "summary", params.Summary, "Print execution summary")
	flag.BoolVar(&params.ReportCSV, "reportCSV", params.ReportCSV, "Format output as CSV")
	flag.BoolVar(&params.PrintConnectionTime, "printConnectionTime", params.PrintConnectionTime, "Print connection times")
	flag.BoolVar(&params.PrintStatusDone, "printStatusDone", params.PrintStatusDone, "Print % process increase")

	flag.StringVar(&params.AwsMMSessionConsistencyLevel, "awsMMsessionConsistencyLevel", params.AwsMMSessionConsistencyLevel, "Consistency level (INSTANCE_RAW or REGIONAL_RAW)")

	// ConnectionPoolTest flags
	flag.IntVar(&params.Workers, "workers", params.Workers, "Concurrent goroutines for ConnectionPoolTest")
	flag.IntVar(&params.MaxOpenConns, "maxOpenConns", params.MaxOpenConns, "sql.DB MaxOpenConns (app pool size)")
	flag.IntVar(&params.MaxIdleConns, "maxIdleConns", params.MaxIdleConns, "sql.DB MaxIdleConns (idle connections kept open)")
	flag.IntVar(&params.ConnMaxLifetimeSec, "connMaxLifetimeSec", params.ConnMaxLifetimeSec, "sql.DB ConnMaxLifetime in seconds (0=unlimited)")
	flag.StringVar(&params.PayloadSize, "payloadSize", params.PayloadSize, "Query payload size: small|medium|large|xlarge")
	flag.BoolVar(&params.RampWorkers, "rampWorkers", params.RampWorkers, "Ramp workers from 1 up to --workers across scenarios")
	flag.IntVar(&params.WarmupLoops, "warmupLoops", params.WarmupLoops, "Warmup iterations before measurement")

	// DataGenTest flags
	flag.IntVar(&params.BatchSize, "batchSize", params.BatchSize, "Rows per INSERT statement (datagen)")
	flag.BoolVar(&params.Truncate, "truncate", params.Truncate, "Truncate tables before loading data (datagen)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "\n%s\n", help.GetHelpText())
		flag.PrintDefaults()
	}
	flag.Parse()

	log.Infof(params.Module)
	if params.Module == "" {
		fmt.Print(help.GetHelpText())
		os.Exit(1)
	}

	switch strings.ToLower(params.Module) {
	case "staleread":
		modules.NewStaleReadTest(params).Run()
	case "connectionpool":
		modules.NewConnectionPoolTest(params).Run()
	case "datagen":
		modules.NewDataGenTest(params).Run()
	default:
		fmt.Fprintf(os.Stderr, "Unknown module: %q\nAvailable: staleread, connectionpool, datagen\n", params.Module)
		os.Exit(1)
	}
}
