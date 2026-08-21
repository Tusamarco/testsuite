package config

type Params struct {
	// --- Database Configuration ---
	Url        string
	UrlRead    string // Used in StaleReadTest to compare Writer vs Reader
	User       string
	Password   string
	Schema     string
	Attributes string // Raw connection parameters (e.g., "&useSSL=false")

	// --- Base Test Settings ---
	Loops     int  // Number of iterations
	Sleep     int  // Sleep time in milliseconds
	Verbose   bool // Enable verbose logging
	Summary   bool // Print summary at the end
	ReportCSV bool // Format output as CSV

	// --- Specific Test Toggles & Thresholds ---
	PrintConnectionTime          bool
	SelectForceAutocommitOff     bool
	RowsNumber                   int
	PrintStatusDone              bool
	AwsMMSessionConsistencyLevel string // E.g., "INSTANCE_RAW", "REGIONAL_RAW"
	ToleranceNanosec             int64  // Tolerance threshold for stale reads

	Module      string
	PingTimeout int

	// --- ConnectionPoolTest settings ---
	Workers            int    // Number of concurrent goroutines
	MaxOpenConns       int    // sql.DB SetMaxOpenConns
	MaxIdleConns       int    // sql.DB SetMaxIdleConns
	ConnMaxLifetimeSec int    // sql.DB SetConnMaxLifetime (seconds, 0 = unlimited)
	PayloadSize        string // small|medium|large|xlarge
	RampWorkers        bool   // If true, iterate worker counts from 1 up to Workers
	WarmupLoops        int    // Warmup iterations before measurement

	// --- DataGenTest settings ---
	BatchSize int  // Number of rows per INSERT batch
	Truncate  bool // Truncate tables before loading data
}

func NewParams() *Params {
	return &Params{
		// Database Defaults
		Url:         "",
		User:        "app_test",
		Password:    "test",
		Schema:      "mysql",
		Attributes:  "&autoReconnect=true",
		PingTimeout: 1000,

		// Base Test Defaults
		Loops:     50,
		Sleep:     0,
		Verbose:   false,
		Summary:   false,
		ReportCSV: false,

		// Specific Test Defaults
		PrintConnectionTime: true,
		RowsNumber:          10000,
		PrintStatusDone:     false,
		ToleranceNanosec:    5000,

		// ConnectionPoolTest Defaults
		Workers:            8,
		MaxOpenConns:       10,
		MaxIdleConns:       5,
		ConnMaxLifetimeSec: 0,
		PayloadSize:        "small",
		RampWorkers:        false,
		WarmupLoops:        5,

		BatchSize: 500,
		Truncate:  true,
	}
}

type ConnectionParameters struct {
	User               string
	Password           string
	Host               string
	Port               int
	Attributes         string
	UseSsl             bool
	SslCertificatePath string
	SslCa              string
	SslClient          string
	SslKey             string
	PingTimeout        int
}

func GetParams() *Params {
	return NewParams()
}
