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

	"testsuite/internal/config"

	"github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"
)

// TestBase holds the configuration and state shared by all test modules.
type TestBase struct {
	Parameters                   config.Params
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
	PingTimeout                  int
}

type DataNode struct {
	Url         string
	Connection  *sql.DB
	CParms      config.ConnectionParameters
	NodeTCPDown bool
}

func (dataNode *DataNode) Close() {
	if dataNode.Connection != nil {
		dataNode.Connection.Close()
	}
}

func NewTestBase(params config.Params) *TestBase {
	return &TestBase{
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
		SchemaName:              params.Schema,
	}
}

func (t *TestBase) ExecuteLocalAction() {}
func (t *TestBase) LocalInit()          {}

func (t *TestBase) Init() {
	t.Loops = t.Parameters.Loops
	t.Sleep = t.Parameters.Sleep
	t.Verbose = t.Parameters.Verbose
	t.Summary = t.Parameters.Summary
	t.ReportCSV = t.Parameters.ReportCSV
	t.SchemaName = t.Parameters.Schema
}

func (t *TestBase) GetExecutionTime() float64 {
	if t.StartTime > 0 && t.EndTime > 0 {
		return t.EndTime - t.StartTime
	}
	return 0
}

func (t *TestBase) ShowHelp() string {
	var sb strings.Builder
	sb.WriteString("******************************************\n")
	sb.WriteString("DB Parameters to use\n")
	sb.WriteString("url [url=jdbc:mysql://127.0.0.1:3306]\n")
	sb.WriteString("user [user=test_user]\n")
	sb.WriteString("password [password=test_password]\n")
	sb.WriteString("schema [schema=test]\n")
	sb.WriteString("\n*****************************************\nApplication Parameters \n")
	sb.WriteString("loops [loops=50]\n")
	sb.WriteString("sleep [sleep=0] value in milliseconds \n")
	sb.WriteString("verbose [verbose=false]\n")
	sb.WriteString("summary [summary=false]\n")
	sb.WriteString("reportCSV [reportCSV=false]\n")
	sb.WriteString("****************************************\n Optional \n")
	sb.WriteString("selectForceAutocommitOff [selectForceAutocommitOff=true]\n")
	return sb.String()
}

func (t *TestBase) FormatDF2(val float64) string {
	return fmt.Sprintf("%.2f", val)
}

func (t *TestBase) FormatCSV(val float64) string {
	return fmt.Sprintf("%.2f", val)
}

func (t *TestBase) GetConnection(cParams config.ConnectionParameters) (bool, *DataNode) {
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

	hostPort := net.JoinHostPort(cParams.Host, strconv.Itoa(cParams.Port))
	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s%s", cParams.User, cParams.Password, hostPort, t.SchemaName, attributes)

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

func (t *TestBase) SplitIPAndPort(address string) (host string, port string, ok bool) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		log.Errorf("SplitIPAndPort failed: %s", err)
		return "", "", false
	}
	return host, port, true
}
