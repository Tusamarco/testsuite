package Global

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
	//ConnParams ConnectionParameters
}

// NewParams creates a Params struct initialized with the default values
// found in your original Java implementation.
func NewParams() *Params {
	return &Params{
		// Database Defaults
		Url:         "",
		User:        "app_test",
		Password:    "test",
		Schema:      "mysql",
		Attributes:  "&autoReconnect=true",
		PingTimeout: 1000, //TODO Check Ping timeout in millisecond

		// Base Test Defaults
		Loops:     50,
		Sleep:     0, // Note: StaleReadTest explicitly overrides this to 2000 if it is <= 0
		Verbose:   false,
		Summary:   false,
		ReportCSV: false,

		// Specific Test Defaults
		PrintConnectionTime: true,
		RowsNumber:          10000,
		PrintStatusDone:     false,
		ToleranceNanosec:    5000, // 5 microseconds default tolerance
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
