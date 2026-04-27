package modules

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"fmt"

	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	// Assuming these packages exist in your Go project based on the Java imports
	// "net.tc/data/db"
	// "net.tc/utils"

	"github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"
	Global "testsuite/src/global"
)

// TestBase holds the configuration and state for tests.
type TestBase struct {
	ConnectionProvider           ConnectionProvider
	Config                       map[string]any
	PippoParameters              map[string]string
	Parameters                   Global.Params
	Loops                        int
	Sleep                        int
	Verbose                      bool
	Summary                      bool
	PrintConnectionTime          bool
	ReportCSV                    bool
	SchemaName                   string
	FormattingLengthForNano      int
	StartConnTimes               []int64
	EndConnTimes                 []int64
	StartTime                    float64
	EndTime                      float64
	AwsMMSessionConsistencyLevel string
}

type DataNode struct {
	Url         string
	Connection  *sql.DB
	CParms      Global.ConnectionParameters
	NodeTCPDown bool
}

func (dataNode *DataNode) Close() {
	if dataNode.Connection != nil {
		dataNode.Connection.Close()
	}
}

// NewTestBase acts as a constructor, initializing default values and maps.
func NewTestBase(params Global.Params) *TestBase {
	return &TestBase{
		Config:                  make(map[string]any),
		Parameters:              params,
		Loops:                   50,
		Sleep:                   0,
		Verbose:                 false,
		Summary:                 false,
		PrintConnectionTime:     true,
		ReportCSV:               false,
		FormattingLengthForNano: 15,
		StartConnTimes:          []int64{},
		EndConnTimes:            []int64{},
	}
}

// ExecuteLocalAction is a placeholder hook for subclasses/implementations.
func (t *TestBase) ExecuteLocalAction() {
	// Generate this in the local implementation
}

// LocalInit is a placeholder hook for subclasses/implementations.
func (t *TestBase) LocalInit() {
	// Add in your implementation specific settings
}

// Init parses arguments and configures the TestBase instance.
func (t *TestBase) Init() {

	t.Loops = t.Parameters.Loops
	t.Sleep = t.Parameters.Sleep

	// Parse booleans
	t.Verbose = t.Parameters.Verbose
	t.Summary = t.Parameters.Summary
	t.ReportCSV = t.Parameters.ReportCSV
	t.SchemaName = t.Parameters.Schema
}

// GetExecutionTime calculates the duration between start and end.
func (t *TestBase) GetExecutionTime() float64 {
	if t.StartTime > 0 && t.EndTime > 0 {
		return t.EndTime - t.StartTime
	}
	return 0
}

// ShowHelp returns a string containing usage instructions.
func (t *TestBase) ShowHelp() string {
	var sb strings.Builder
	sb.WriteString("******************************************\n")
	sb.WriteString("DB PippoParameters to use\n")
	sb.WriteString("PippoParameters are COMMA separated and the whole set must be pass as string\n")
	sb.WriteString("IE java -Xms2G -Xmx3G -classpath \"./*:./lib/*\" net.tc.testsuite.MySQLConnectionTest \"loops=10,parameters=&characterEncoding=UTF-8, url=jdbc:mysql://192.168.4.22:3306\" \n")
	sb.WriteString("url [url=jdbc:mysql://127.0.0.1:3306]\n")
	sb.WriteString("user [user=test_user]\n")
	sb.WriteString("password [password=test_password]\n")
	sb.WriteString("parameters [parameters=&useSSL=false&autoReconnect=true]\n")
	sb.WriteString("schema [schema=test]\n")
	sb.WriteString("\n*****************************************\nApplication PippoParameters \n")
	sb.WriteString("loops [loops=50]\n")
	sb.WriteString("sleep [sleep=0] value in milliseconds \n")
	sb.WriteString("verbose [verbose=false]\n")
	sb.WriteString("summary [summary=false]\n")
	sb.WriteString("reportCSV [reportCSV=false]\n")
	sb.WriteString("****************************************\n Optional \n")
	sb.WriteString("selectForceAutocommitOff [selectForceAutocommitOff=true]\n")
	return sb.String()
}

// --- Formatting Helpers ---

// FormatDF2 formats a float equivalent to Java's DecimalFormat("#,###,###,##0.00")
// Note: Go's standard library does not natively support comma separators for floats,
// so this provides standard 2-decimal formatting.
func (t *TestBase) FormatDF2(val float64) string {
	return fmt.Sprintf("%.2f", val)
}

// FormatCSV formats a float equivalent to Java's DecimalFormat("#.00")
func (t *TestBase) FormatCSV(val float64) string {
	return fmt.Sprintf("%.2f", val)
}

// Return a DataObject with an open connection
func (t *TestBase) GetConnection(cParams Global.ConnectionParameters) (bool, *DataNode) {
	attributes := "?timeout=1s"

	dataObject := &DataNode{
		CParms:      cParams,
		NodeTCPDown: false,
	}

	if cParams.UseSsl {
		if cParams.SslCertificatePath != "" {

			ca := filepath.Join(cParams.SslCertificatePath, cParams.SslCa)
			client := filepath.Join(cParams.SslCertificatePath, cParams.SslClient)
			key := filepath.Join(cParams.SslCertificatePath, cParams.SslKey)

			rootCertPool := x509.NewCertPool()

			pem, err := os.ReadFile(ca)
			if err != nil {
				log.Error(err, " While trying to connect to node (CA certificate) ", net.JoinHostPort(cParams.Host, strconv.Itoa(cParams.Port)))
				dataObject.NodeTCPDown = true
				return false, dataObject
			}
			if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
				log.Error("Failed to append PEM to pool, connecting to node ", net.JoinHostPort(cParams.Host, strconv.Itoa(cParams.Port)))
				dataObject.NodeTCPDown = true
				return false, dataObject
			}

			certs, err := tls.LoadX509KeyPair(client, key)
			if err != nil {
				log.Error(err, " While trying to connect to node (Key certificate) ", net.JoinHostPort(cParams.Host, strconv.Itoa(cParams.Port)))
				dataObject.NodeTCPDown = true
				return false, dataObject
			}

			tlsKey := fmt.Sprintf("custom_%s_%d", cParams.Host, cParams.Port)
			mysql.RegisterTLSConfig(tlsKey, &tls.Config{
				RootCAs:      rootCertPool,
				Certificates: []tls.Certificate{certs},
			})

			attributes += "&tls=" + tlsKey
		} else {
			attributes += "&tls=skip-verify"
		}
	}

	// 5. Build DSN cleanly using fmt.Sprintf
	hostPort := net.JoinHostPort(cParams.Host, strconv.Itoa(cParams.Port))
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/performance_schema%s", cParams.User, cParams.Password, hostPort, attributes)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Error(err.Error())
		dataObject.NodeTCPDown = true
		return false, dataObject
	}

	dataObject.Connection = db

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cParams.PingTimeout)*time.Millisecond)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		log.Error(err.Error())
		dataObject.NodeTCPDown = true

		dataObject.Connection.Close()

		return false, dataObject
	}

	return true, dataObject
}
