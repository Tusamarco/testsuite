package modules

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"testsuite/src/utils" // Replace with your actual module path
)

// ConnectionProvider represents the interface expected by this test,
// mimicking your net.tc.data.db.ConnectionProvider logic.
type ConnectionProvider interface {
	GetMySQLConnection(ctx context.Context) (*sql.Conn, error)
	ReturnConnection(conn *sql.Conn) error
}

// MySQLConnectionTest tests MySQL connection opening and closing times.
type MySQLConnectionTest struct {
	*TestBase // Embeds TestBase to inherit its fields and methods

	ConnectionProvider      ConnectionProvider
	PrintConnectionTime     bool
	FormattingLengthForNano int
	StartConnTimes          []int64
	EndConnTimes            []int64
}

// NewMySQLConnectionTest creates and initializes a new MySQLConnectionTest.
func NewMySQLConnectionTest() *MySQLConnectionTest {
	tb := NewTestBase()
	return &MySQLConnectionTest{
		TestBase:                tb,
		PrintConnectionTime:     true,
		FormattingLengthForNano: 15,
		StartConnTimes:          make([]int64, 0),
		EndConnTimes:            make([]int64, 0),
	}
}

// Run acts as the entry point replacing the Java main method.
func (t *MySQLConnectionTest) Run(args []string) {
	if len(args) == 0 || len(args) > 1 || (len(args) >= 1 && strings.Contains(args[0], "help")) || args[0] == "" {
		fmt.Println(t.ShowHelp())
		os.Exit(0)
	}

	// Remove spaces and split by comma
	cleanArgs := strings.ReplaceAll(args[0], " ", "")
	argsLoc := strings.Split(cleanArgs, ",")

	t.GenerateConfig(DefaultsConnection)
	t.Init(argsLoc)
	t.LocalInit(argsLoc)

	// In a real application, you'd initialize your actual ConnectionProvider here using t.Config
	// t.ConnectionProvider = db.NewConnectionProvider(t.Config)

	t.ExecuteLocal()
	os.Exit(0)
}

// LocalInit initializes local test properties from configuration.
func (t *MySQLConnectionTest) LocalInit(argsLoc []string) {
	if val, ok := t.Config["printConnectionTime"]; ok && val != nil {
		if strVal, isStr := val.(string); isStr {
			parsed, err := strconv.ParseBool(strVal)
			if err == nil {
				t.PrintConnectionTime = parsed
				return
			}
		}
	}
	t.PrintConnectionTime = true // Default
}

// ExecuteLocal runs the primary benchmark loop.
func (t *MySQLConnectionTest) ExecuteLocal() {
	var hostName string

	if t.ReportCSV {
		prefix := ""
		if t.Verbose {
			prefix = "SqlOutput,"
		}
		fmt.Println(prefix + "OpenTime(ns),OpenTime(us),CloseTime(ns),CloseTime(us)")
	}

	ctx := context.Background()

	// TODO: make this loop become a call to a runnableJOB/Goroutine
	for iLoop := 0; iLoop <= t.Loops; iLoop++ {
		startOpen := time.Now()
		conn, err := t.ConnectionProvider.GetMySQLConnection(ctx)
		endOpen := time.Now()

		if err != nil {
			fmt.Printf("Error getting connection: %v\n", err)
			continue
		}

		hostName = t.executeSQL(ctx, conn)

		if t.Sleep > 0 {
			time.Sleep(time.Duration(t.Sleep) * time.Millisecond)
		}

		startClose := time.Now()
		err = t.ConnectionProvider.ReturnConnection(conn) // or conn.Close()
		endClose := time.Now()

		if err != nil {
			fmt.Printf("Error closing connection: %v\n", err)
		}

		openDur := endOpen.Sub(startOpen).Nanoseconds()
		closeDur := endClose.Sub(startClose).Nanoseconds()

		t.StartConnTimes = append(t.StartConnTimes, openDur)
		t.EndConnTimes = append(t.EndConnTimes, closeDur)

		open := t.StartConnTimes[len(t.StartConnTimes)-1]
		close := t.EndConnTimes[len(t.EndConnTimes)-1]
		msOpen := open / 1000
		msClose := close / 1000

		if t.PrintConnectionTime {
			if !t.ReportCSV {
				prefix := ""
				if t.Verbose {
					prefix = hostName + " "
				}
				fmt.Printf("%sOpen Connection time (nano) = %s microS = %s Close Connection time (nano) = %s microS = %s\n",
					prefix,
					utils.FormatNumberToPrint(t.FormattingLengthForNano, t.FormatDF2(float64(open))),
					utils.FormatNumberToPrint(10, t.FormatDF2(float64(msOpen))),
					utils.FormatNumberToPrint(t.FormattingLengthForNano, t.FormatDF2(float64(close))),
					utils.FormatNumberToPrint(10, t.FormatDF2(float64(msClose))),
				)
			} else {
				prefix := ""
				if t.Verbose {
					prefix = hostName + ","
				}
				fmt.Printf("%s%d,%d,%d,%d\n", prefix, open, msOpen, close, msClose)
			}
		}
	}

	if t.Summary {
		fmt.Println(t.calculateSummary())
	}
}

// calculateSummary generates average and standard deviation summaries.
func (t *MySQLConnectionTest) calculateSummary() string {
	if len(t.StartConnTimes) <= 1 {
		return "Insufficient data for summary (need > 1 iteration)."
	}

	var totOpen, totClose int64
	var maxOpen, maxClose int64
	minOpen := int64(math.MaxInt64)
	minClose := int64(math.MaxInt64)

	// Taking the number from instance 1 and not 0 to skip the object initialization cost
	for i := 1; i < len(t.StartConnTimes); i++ {
		openVal := t.StartConnTimes[i]
		closeVal := t.EndConnTimes[i]

		totOpen += openVal
		totClose += closeVal

		if openVal > maxOpen {
			maxOpen = openVal
		}
		if closeVal > maxClose {
			maxClose = closeVal
		}
		if openVal < minOpen {
			minOpen = openVal
		}
		if closeVal < minClose {
			minClose = closeVal
		}
	}

	count := float64(len(t.StartConnTimes) - 1)
	avgOpen := float64(totOpen) / count
	avgClose := float64(totClose) / count

	// Utilizes the MathU logic converted in earlier prompts
	standardDevOpen := utils.CalculateStandardDeviation(t.StartConnTimes[1:], avgOpen)
	standardDevClose := utils.CalculateStandardDeviation(t.EndConnTimes[1:], avgClose)

	var averageReport strings.Builder

	if !t.ReportCSV {
		averageReport.WriteString("\n Summary \n")
		averageReport.WriteString(fmt.Sprintf("Average Time Open ns = %s microS = %s \n",
			utils.FormatNumberToPrint(t.FormattingLengthForNano, t.FormatDF2(avgOpen)),
			utils.FormatNumberToPrint(8, t.FormatDF2(avgOpen/1000))))
		averageReport.WriteString(fmt.Sprintf("Average Time Close ns = %s microS = %s \n",
			utils.FormatNumberToPrint(t.FormattingLengthForNano, t.FormatDF2(avgClose)),
			utils.FormatNumberToPrint(8, t.FormatDF2(avgClose/1000))))

		averageReport.WriteString(fmt.Sprintf("Max Open (time in nano seconds)  = %s\n", utils.FormatNumberToPrint(t.FormattingLengthForNano, t.FormatDF2(float64(maxOpen)))))
		averageReport.WriteString(fmt.Sprintf("Max Close (time in nano seconds) = %s\n", utils.FormatNumberToPrint(t.FormattingLengthForNano, t.FormatDF2(float64(maxClose)))))

		averageReport.WriteString(fmt.Sprintf("Min Open (time in nano seconds)  = %s\n", utils.FormatNumberToPrint(t.FormattingLengthForNano, t.FormatDF2(float64(minOpen)))))
		averageReport.WriteString(fmt.Sprintf("Min Close (time in nano seconds) = %s\n", utils.FormatNumberToPrint(t.FormattingLengthForNano, t.FormatDF2(float64(minClose)))))

		averageReport.WriteString("\n Standard deviation \n")
		averageReport.WriteString(fmt.Sprintf("StdDev connection Open   = %s\n", utils.FormatNumberToPrint(15, t.FormatDF2(standardDevOpen))))
		averageReport.WriteString(fmt.Sprintf("StdDev connection Close  = %s\n", utils.FormatNumberToPrint(15, t.FormatDF2(standardDevClose))))
	} else {
		averageReport.WriteString("AVG_Open(ns),AVG_open(us),AVG_close(ns),AVG_close(us),MaxOpen(ns),MaxClose(ns),MinOpen(ns),MinClose(ns),StdvOpen,StdvClose\n")
		averageReport.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s,%s,%s,%s",
			t.FormatCSV(avgOpen),
			t.FormatCSV(avgOpen/1000),
			t.FormatCSV(avgClose),
			t.FormatCSV(avgClose/1000),
			t.FormatCSV(float64(maxOpen)),
			t.FormatCSV(float64(maxClose)),
			t.FormatCSV(float64(minOpen)),
			t.FormatCSV(float64(minClose)),
			t.FormatCSV(standardDevOpen),
			t.FormatCSV(standardDevClose),
		))
	}

	return averageReport.String()
}

// executeSQL fires a simple query against a provided *sql.Conn.
func (t *MySQLConnectionTest) executeSQL(ctx context.Context, conn *sql.Conn) string {
	var hostname string
	// In Go, database/sql automatically handles statements beneath the hood for simple queries.
	err := conn.QueryRowContext(ctx, "SELECT @@hostname as name").Scan(&hostname)
	if err != nil {
		fmt.Printf("Error executing SQL: %v\n", err)
		return ""
	}
	return hostname
}

// ShowHelp overrides the TestBase help payload with subclass-specific text.
func (t *MySQLConnectionTest) ShowHelp() string {
	var sb strings.Builder
	// Append super.showHelp() equivalent
	sb.WriteString(t.TestBase.ShowHelp())

	sb.WriteString("\n****************************************\n Optional For the test\n")
	sb.WriteString(" printConnectionTime [printConnectionTime=true]\n")

	return sb.String()
}
