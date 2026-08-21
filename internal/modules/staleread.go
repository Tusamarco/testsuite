package modules

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"testsuite/internal/config"
	"testsuite/internal/utils"

	log "github.com/sirupsen/logrus"
)

// StaleReadTest tests for replication lag/stale reads between writer and reader nodes.
type StaleReadTest struct {
	*TestBase

	RowsNumber       int
	ToleranceNanosec int64
	Results          map[string][]int64
	StaleReads       int
	PrintStatusDone  bool
}

func NewStaleReadTest(params config.Params) *StaleReadTest {
	testBase := NewTestBase(params)
	return &StaleReadTest{
		TestBase:         testBase,
		RowsNumber:       params.RowsNumber,
		ToleranceNanosec: params.ToleranceNanosec,
		Results:          make(map[string][]int64),
	}
}

func (staleR *StaleReadTest) LocalInit() {}

func (staleR *StaleReadTest) Run() {
	staleR.Init()
	staleR.LocalInit()

	if staleR.Sleep <= 0 {
		staleR.Sleep = 2000
	}
	if staleR.Sleep < 1000 {
		staleR.Sleep = 1000
	}

	if staleR.ReportCSV && staleR.PrintStatusDone {
		staleR.PrintStatusDone = false
		fmt.Println("\n[WARNING] you cannot print the % increase when output is in CSV." +
			"\n if nothing is printed out and you need to check the status of the test" +
			"\n try to check with SHOW PROCESSLIST\nPrint % increase is now disabled\n")
	}

	staleR.executeLocal()
	os.Exit(0)
}

func (staleR *StaleReadTest) executeLocal() {
	ctx := context.Background()

	host, port, success := staleR.TestBase.SplitIPAndPort(staleR.Parameters.Url)
	if !success {
		log.Errorf("StaleReadTest: SplitIPAndPort failed")
	}

	porti, err := strconv.Atoi(port)
	if err != nil {
		log.Errorf("StaleReadTest: Port is not valid: %s", err)
	}

	cParams := config.ConnectionParameters{
		User:        staleR.Parameters.User,
		Password:    staleR.Parameters.Password,
		Host:        host,
		Port:        porti,
		Attributes:  staleR.Parameters.Attributes,
		PingTimeout: staleR.Parameters.PingTimeout,
	}

	success, writeConn := staleR.TestBase.GetConnection(cParams)
	if !success {
		log.Errorf("Error getting write connection: %v\n", err)
		os.Exit(1)
	}

	hostR, portR, successR := staleR.TestBase.SplitIPAndPort(staleR.Parameters.UrlRead)
	portiR, _ := strconv.Atoi(portR)

	cParams.Host = hostR
	cParams.Port = portiR

	if !successR {
		log.Errorf("StaleReadTest: SplitIPAndPort failed for reader")
	}

	successR, readConn := staleR.TestBase.GetConnection(cParams)
	if !successR {
		log.Errorf("Error getting read connection: %v\n", err)
		os.Exit(1)
	}

	if !staleR.checkReaderNode(ctx, writeConn.Connection, readConn.Connection) {
		fmt.Println(" Writer and Reader must be different hosts.\n Unable to get different hosts\nExit ")
		os.Exit(1)
	}

	if !staleR.createTable(ctx, writeConn.Connection) {
		fmt.Println(" Error Cannot create table ")
	}

	if !staleR.fillTable(ctx, writeConn.Connection) {
		fmt.Println(" Error Cannot fill table with data")
	}

	if !staleR.executeStaleReadTest(ctx, writeConn.Connection, readConn.Connection) {
		fmt.Println(" Error Cannot execute test")
	}

	writeConn.Connection.Close()
	readConn.Connection.Close()

	if staleR.Summary {
		staleR.printReport()
	}
}

func (staleR *StaleReadTest) checkReaderNode(ctx context.Context, write, read *sql.DB) bool {
	for iCountdown := 5; iCountdown > 0; iCountdown-- {
		var wHostName, rHostName string

		errW := write.QueryRowContext(ctx, "select @@hostname as host").Scan(&wHostName)
		errR := read.QueryRowContext(ctx, "select @@hostname as host").Scan(&rHostName)

		if errW == nil && errR == nil && wHostName != rHostName {
			fmt.Println("Ok I have identified valid hosts")
			return true
		}
		time.Sleep(500 * time.Millisecond)
	}
	return false
}

func (staleR *StaleReadTest) createTable(ctx context.Context, connWrite *sql.DB) bool {
	drop := fmt.Sprintf("DROP TABLE IF EXISTS `%s`.`staleread`;", staleR.SchemaName)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s`.`staleread` (", staleR.SchemaName))
	sb.WriteString("`id` bigint(11) NOT NULL AUTO_INCREMENT,")
	sb.WriteString("`s` char(255) DEFAULT NULL,")
	sb.WriteString("`staleR` datetime NOT NULL,")
	sb.WriteString("`g` bigint(11) NOT NULL,")
	sb.WriteString(" KEY(`id`, `staleR`),")
	sb.WriteString(" PRIMARY KEY(`id`)")
	sb.WriteString(" ) ENGINE=InnoDB DEFAULT CHARSET=utf8;")

	if staleR.AwsMMSessionConsistencyLevel != "" {
		connWrite.ExecContext(ctx, fmt.Sprintf("SET SESSION aurora_mm_session_consistency_level='%s'", staleR.AwsMMSessionConsistencyLevel))
	}

	if _, err := connWrite.ExecContext(ctx, drop); err != nil {
		fmt.Printf("Error dropping table: %v\n", err)
		return false
	}

	if _, err := connWrite.ExecContext(ctx, sb.String()); err != nil {
		fmt.Printf("Error creating table: %v\n", err)
		return false
	}

	return true
}

func (staleR *StaleReadTest) fillTable(ctx context.Context, writeConn *sql.DB) bool {
	dateStart := utils.GetTimeStampFormatted(time.Now().UnixMilli(), "2006-01-02 15:04:05")
	fmt.Println("Loading table ... please wait")

	if staleR.AwsMMSessionConsistencyLevel != "" {
		writeConn.ExecContext(ctx, fmt.Sprintf("SET SESSION aurora_mm_session_consistency_level='%s'", staleR.AwsMMSessionConsistencyLevel))
	}

	insertQuery := fmt.Sprintf("INSERT INTO %s.staleread VALUES (NULL, uuid(), time('%s'), (FLOOR( 1 + RAND( ) *60 )))", staleR.SchemaName, dateStart)

	totRows := 0
	for totRows < staleR.RowsNumber {
		tx, err := writeConn.BeginTx(ctx, nil)
		if err != nil {
			return false
		}

		stmt, err := tx.PrepareContext(ctx, insertQuery)
		if err != nil {
			tx.Rollback()
			return false
		}

		for subBatch := 1; subBatch <= 100 && totRows < staleR.RowsNumber; subBatch++ {
			if _, err = stmt.ExecContext(ctx); err == nil {
				totRows++
			}
		}

		stmt.Close()
		tx.Commit()

		fmt.Printf(" wrote %d of %d\n", totRows, staleR.RowsNumber)
	}

	fmt.Println("Loading table ... done")
	return true
}

func (staleR *StaleReadTest) executeStaleReadTest(ctx context.Context, writeConn, readConn *sql.DB) bool {
	if staleR.AwsMMSessionConsistencyLevel != "" {
		writeConn.ExecContext(ctx, fmt.Sprintf("SET SESSION aurora_mm_session_consistency_level='%s'", staleR.AwsMMSessionConsistencyLevel))
		readConn.ExecContext(ctx, fmt.Sprintf("SET SESSION aurora_mm_session_consistency_level='%s'", staleR.AwsMMSessionConsistencyLevel))
	}

	fmt.Println("Going to sleep for two seconds before executing")
	time.Sleep(time.Duration(staleR.Sleep) * time.Millisecond)
	fmt.Println("Executing:")

	staleR.StartTime = float64(time.Now().UnixNano())

	ids := staleR.getIds(ctx, writeConn)

	var writeTime []int64
	var readTime []int64
	var lagTime []int64

	dateRun := utils.GetTimeStampFormatted(time.Now().UnixMilli(), "2006-01-02 15:04:05")
	var sb strings.Builder

	if !staleR.ReportCSV {
		sb.WriteString(fmt.Sprintf("%s (%s Exceeding the tollerance of %d)\n",
			utils.FormatStringToPrint(6, "ID"),
			utils.FormatStringToPrint(6, " #loop"),
			staleR.ToleranceNanosec))
	} else {
		sb.WriteString("ID,#loop,writeTime,readTime,lagTime\n")
	}

	lengthIds := len(ids)
	if staleR.Loops < lengthIds {
		lengthIds = staleR.Loops
	}

	iCounter := 0
	iDone := 0

	for _, id := range ids {
		iCounter++
		iDone++

		var sqlW string
		if utils.IsEvenNumber(int64(iCounter)) {
			sqlW = fmt.Sprintf("UPDATE %s.staleread SET staleR = '%s' WHERE id = %d", staleR.SchemaName, dateRun, id)
		} else {
			sqlW = fmt.Sprintf("REPLACE INTO %s.staleread VALUES (%d, uuid(), time('%s'), (FLOOR( 1 + RAND( ) *60 )))", staleR.SchemaName, id, dateRun)
		}
		sqlR := fmt.Sprintf("SELECT id FROM %s.staleread WHERE id = %d and staleR = '%s'", staleR.SchemaName, id, dateRun)

		startTimeWrite := time.Now().UnixNano()
		writeConn.ExecContext(ctx, sqlW)
		startTimeRead := time.Now().UnixNano()
		writeTimei := startTimeRead - startTimeWrite

		if staleR.ReportCSV {
			sb.WriteString(fmt.Sprintf("%d,%d,", id, iCounter))
		} else {
			sb.WriteString(fmt.Sprintf("%s (%s)", utils.FormatStringToPrint(6, fmt.Sprint(id)), utils.FormatStringToPrint(6, fmt.Sprint(iCounter))))
		}

		var checkRecords [2]int64
		lag := false

		for checkRecords[0] < 1 {
			checkRecords = staleR.checkRecord(ctx, readConn, sqlR)
			if checkRecords[0] < 1 {
				lag = true
			}
		}

		readTimei := checkRecords[1]
		endTimeRead := time.Now().UnixNano()

		if lag {
			staleR.StaleReads++
		}

		lagTimei := endTimeRead - startTimeRead

		writeTime = append(writeTime, writeTimei)
		readTime = append(readTime, readTimei)
		lagTime = append(lagTime, lagTimei)

		if staleR.Verbose {
			staleR.printVerbose(&sb, iCounter, id, writeTimei, readTimei, lagTimei, lag)
		} else {
			fmt.Print(".")
		}

		if staleR.Verbose && staleR.PrintStatusDone && iDone >= 100 {
			pctDone := float64(iCounter*100) / float64(lengthIds)
			iDone = 0
			fmt.Printf("Currently executed %.4f %%\n", pctDone)
		}

		sb.Reset()

		if staleR.Loops < staleR.RowsNumber && iCounter >= staleR.Loops {
			break
		}
	}

	fmt.Println("\n Done \n ")
	staleR.Results["writeTime"] = writeTime
	staleR.Results["readTime"] = readTime
	staleR.Results["lagTime"] = lagTime

	staleR.EndTime = float64(time.Now().UnixNano())

	return true
}

func (staleR *StaleReadTest) checkRecord(ctx context.Context, rstmt *sql.DB, sqlR string) [2]int64 {
	var values [2]int64
	readStart := time.Now().UnixNano()

	var throwawayID int64
	err := rstmt.QueryRowContext(ctx, sqlR).Scan(&throwawayID)

	if err == nil {
		values[0] = 1
	} else if err != sql.ErrNoRows {
		fmt.Printf("Error checking record: %v\n", err)
	}

	values[1] = time.Now().UnixNano() - readStart
	return values
}

func (staleR *StaleReadTest) getIds(ctx context.Context, wstmt *sql.DB) []int64 {
	var ids []int64
	rows, err := wstmt.QueryContext(ctx, fmt.Sprintf("SELECT id from %s.staleread", staleR.SchemaName))
	if err != nil {
		fmt.Printf("Error fetching IDs: %v\n", err)
		return ids
	}
	defer rows.Close()

	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

func (staleR *StaleReadTest) printVerbose(sb *strings.Builder, iCounter int, id int64, writeTimei, readTimei, lagTimei int64, lag bool) {
	if staleR.ReportCSV {
		sb.WriteString(fmt.Sprintf("%d,%d,%d,%d,%d", id, iCounter, writeTimei, readTimei, lagTimei))
		fmt.Println(sb.String())
	} else {
		if lag && lagTimei > staleR.ToleranceNanosec {
			sb.WriteString(fmt.Sprintf("%s write Time (ns) ", utils.FormatNumberToPrint(14, fmt.Sprint(writeTimei))))
			sb.WriteString(fmt.Sprintf("%s read Time (ns)  ", utils.FormatNumberToPrint(14, fmt.Sprint(readTimei))))
			sb.WriteString(fmt.Sprintf("%s lag Time (ns)   ", utils.FormatNumberToPrint(14, fmt.Sprint(lagTimei))))
			fmt.Println(sb.String())
		}
	}
}

func (staleR *StaleReadTest) printReport() {
	var averageReport strings.Builder

	writeTimeData := staleR.Results["writeTime"]
	readTimeData := staleR.Results["readTime"]
	lagTimeData := staleR.Results["lagTime"]

	averageWrite, _ := utils.GetAverage(writeTimeData)
	averageRead, _ := utils.GetAverage(readTimeData)
	averageLag, _ := utils.GetAverage(lagTimeData)

	stdWrite := utils.CalculateStandardDeviation(writeTimeData, averageWrite)
	stdRead := utils.CalculateStandardDeviation(readTimeData, averageRead)
	stdLag := utils.CalculateStandardDeviation(lagTimeData, averageLag)

	maxWrite, minWrite := utils.GetMaxMin(writeTimeData)
	maxRead, minRead := utils.GetMaxMin(readTimeData)
	maxLag, minLag := utils.GetMaxMin(lagTimeData)

	pct := float32(0.0)
	if staleR.Loops > staleR.RowsNumber && staleR.RowsNumber > 0 {
		pct = float32(staleR.StaleReads*100) / float32(staleR.RowsNumber)
	} else if staleR.Loops > 0 {
		pct = float32(staleR.StaleReads*100) / float32(staleR.Loops)
	}

	if staleR.ReportCSV {
		averageReport.WriteString("loops,stale reads found,stale read % on total," +
			" total time,avg write time,avg read time,avg lag time," +
			" min write,max write,min read,max read,max lag, min lag, \n")

		averageReport.WriteString(fmt.Sprintf("%d,%d,%.3f,%.3f,%.3f,%.3f,%.3f,",
			staleR.RowsNumber, staleR.StaleReads, pct, staleR.GetExecutionTime(), averageWrite, averageRead, averageLag))

		averageReport.WriteString(fmt.Sprintf("%d,%d,%d,%d,%d,%d,",
			minWrite, maxWrite, minRead, maxRead, minLag, maxLag))

		averageReport.WriteString(fmt.Sprintf("%.3f,%.3f,%.3f\n", stdWrite, stdRead, stdLag))
	} else {
		averageReport.WriteString("\n============ Summary ===========")
		averageReport.WriteString(fmt.Sprintf("\nTotal loops  = %d", staleR.Loops))
		averageReport.WriteString(fmt.Sprintf("\nTotal rows  = %d", staleR.RowsNumber))
		averageReport.WriteString(fmt.Sprintf("\nStale reads found = %d", staleR.StaleReads))
		averageReport.WriteString(fmt.Sprintf("\nStale reads found%% = %.3f", pct))
		averageReport.WriteString("\n============ Time in nano seconds")
		averageReport.WriteString(fmt.Sprintf("\nTotal execution time = %.3f", staleR.GetExecutionTime()))
		averageReport.WriteString(fmt.Sprintf("\nAverage write time = %.3f", averageWrite))
		averageReport.WriteString(fmt.Sprintf("\nAverage read time = %.3f", averageRead))
		averageReport.WriteString(fmt.Sprintf("\nAverage lag time = %.3f", averageLag))
		averageReport.WriteString(fmt.Sprintf("\nMax/Min Write time = %d/%d", maxWrite, minWrite))
		averageReport.WriteString(fmt.Sprintf("\nMax/Min Read time = %d/%d", maxRead, minRead))
		averageReport.WriteString(fmt.Sprintf("\nMax/Min lag time = %d/%d", maxLag, minLag))
		averageReport.WriteString(fmt.Sprintf("\nstd Dev Write time = %.3f", stdWrite))
		averageReport.WriteString(fmt.Sprintf("\nstd Dev Read time = %.3f", stdRead))
		averageReport.WriteString(fmt.Sprintf("\nstd Dev lag time = %.3f", stdLag))
	}

	fmt.Println(averageReport.String())
}

func (staleR *StaleReadTest) ShowHelp() string {
	var sb strings.Builder
	sb.WriteString(staleR.TestBase.ShowHelp())
	sb.WriteString("\n****************************************\n Optional For the test: Stale read ")
	sb.WriteString(" urlRead=jdbc:mysql://127.0.0.1:3307 \n")
	sb.WriteString("rowsNumber [rowsNumber=10000]\n")
	sb.WriteString("=============")
	sb.WriteString("sleep in this context refer to the time the test will wait after table load\n" +
		"Default sleep = 2000 ms (2 seconds)")
	sb.WriteString("\n\nawsMMsessionConsistencyLevel [awsMMsessionConsistencyLevel=null| INSTANCE_RAW|REGIONAL_RAW]\n")
	sb.WriteString("\n\nprintStatusDone [printStatusDone=false]\n")
	return sb.String()
}
