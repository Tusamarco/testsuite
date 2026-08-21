package modules

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"testsuite/internal/config"

	log "github.com/sirupsen/logrus"
)

type DataGenTest struct {
	*TestBase
	TargetUsers int
	BatchSize   int
	Truncate    bool
}

func NewDataGenTest(params config.Params) *DataGenTest {
	testBase := NewTestBase(params)
	targetUsers := params.RowsNumber
	if targetUsers <= 0 {
		targetUsers = 2_000_000
	}
	batchSize := params.BatchSize
	if batchSize <= 0 {
		batchSize = 500
	}
	return &DataGenTest{
		TestBase:    testBase,
		TargetUsers: targetUsers,
		BatchSize:   batchSize,
		Truncate:    params.Truncate,
	}
}

type cityRow struct {
	id         int
	name       string
	iso3       string
	subcountry string
}

func (t *DataGenTest) Run() {
	t.Init()

	host, port, ok := t.SplitIPAndPort(t.Parameters.Url)
	if !ok {
		log.Errorf("DataGenTest: invalid URL %s", t.Parameters.Url)
		os.Exit(1)
	}
	portI, err := strconv.Atoi(port)
	if err != nil {
		log.Errorf("DataGenTest: invalid port: %v", err)
		os.Exit(1)
	}

	cParams := config.ConnectionParameters{
		User:        t.Parameters.User,
		Password:    t.Parameters.Password,
		Host:        host,
		Port:        portI,
		Attributes:  t.Parameters.Attributes,
		PingTimeout: t.Parameters.PingTimeout,
	}

	success, node := t.GetConnection(cParams)
	if !success {
		log.Errorf("DataGenTest: cannot connect")
		os.Exit(1)
	}
	db := node.Connection
	defer db.Close()

	workers := t.Parameters.Workers
	if workers <= 0 {
		workers = 4
	}
	db.SetMaxOpenConns(workers + 2)
	db.SetMaxIdleConns(workers)

	numAddresses := t.TargetUsers / 4
	if numAddresses < 100_000 {
		numAddresses = 100_000
	}

	fmt.Printf("=== DataGen ===\n")
	fmt.Printf("Target users    : %d\n", t.TargetUsers)
	fmt.Printf("Target addresses: %d\n", numAddresses)
	fmt.Printf("Workers         : %d\n", workers)
	fmt.Printf("Batch size      : %d\n", t.BatchSize)
	fmt.Printf("Truncate        : %v\n", t.Truncate)
	fmt.Println()

	start := time.Now()

	if err := t.seedReferenceTables(db); err != nil {
		log.Errorf("DataGenTest: seedReferenceTables failed: %v", err)
		os.Exit(1)
	}

	cities, err := t.loadCities(db)
	if err != nil {
		log.Errorf("DataGenTest: loadCities failed: %v", err)
		os.Exit(1)
	}
	fmt.Printf("Loaded %d cities\n", len(cities))

	t.generateAddresses(db, numAddresses, cities, workers)
	t.generateUsers(db, t.TargetUsers, numAddresses, workers)

	elapsed := time.Since(start)
	fmt.Printf("\n=== Done ===\n")
	fmt.Printf("Elapsed         : %s\n", elapsed.Round(time.Millisecond))
	fmt.Printf("Total addresses : %d\n", numAddresses)
	fmt.Printf("Total users     : %d\n", t.TargetUsers)
}

func (t *DataGenTest) seedReferenceTables(db *sql.DB) error {
	ctx := context.Background()

	tables := []string{
		"continents", "countries", "cities", "streets",
		"firstname", "lastname", "titles", "population",
	}
	for _, tbl := range tables {
		if _, err := db.ExecContext(ctx, "TRUNCATE TABLE `"+tbl+"`"); err != nil {
			return fmt.Errorf("truncate %s: %w", tbl, err)
		}
	}

	if err := t.seedContinents(ctx, db); err != nil {
		return err
	}
	if err := t.seedCountries(ctx, db); err != nil {
		return err
	}
	if err := t.seedCities(ctx, db); err != nil {
		return err
	}
	if err := t.seedStreets(ctx, db); err != nil {
		return err
	}
	if err := t.seedFirstNames(ctx, db); err != nil {
		return err
	}
	if err := t.seedLastNames(ctx, db); err != nil {
		return err
	}
	if err := t.seedTitles(ctx, db); err != nil {
		return err
	}
	if err := t.seedPopulation(ctx, db); err != nil {
		return err
	}

	return nil
}

func (t *DataGenTest) seedContinents(ctx context.Context, db *sql.DB) error {
	const batchSz = 100
	base := "INSERT INTO `continents` (`iso2`,`iso3`,`name`) VALUES "
	args := make([]interface{}, 0, len(dgContinents)*3)
	placeholders := make([]string, 0, len(dgContinents))
	for _, c := range dgContinents {
		placeholders = append(placeholders, "(?,?,?)")
		args = append(args, c.iso2, c.iso3, c.name)
		if len(placeholders) >= batchSz {
			if err := insertBatch(ctx, db, base+strings.Join(placeholders, ","), args); err != nil {
				return err
			}
			placeholders = placeholders[:0]
			args = args[:0]
		}
	}
	if len(placeholders) > 0 {
		return insertBatch(ctx, db, base+strings.Join(placeholders, ","), args)
	}
	fmt.Printf("Seeded continents: %d\n", len(dgContinents))
	return nil
}

func (t *DataGenTest) seedCountries(ctx context.Context, db *sql.DB) error {
	const batchSz = 100
	base := "INSERT INTO `countries` (`official_name_en`,`iso2`,`iso3`,`isonumeric`,`currency_alphabetic_code`," +
		"`currency_name`,`english_formal`,`english_short`,`capital`,`continent`,`languages`," +
		"`region_code`,`region_name`,`sub_region_code`,`sub_region_name`,`tld`,`is_independent`) VALUES "

	placeholders := make([]string, 0, batchSz)
	args := make([]interface{}, 0, batchSz*17)
	total := 0

	flush := func() error {
		if len(placeholders) == 0 {
			return nil
		}
		err := insertBatch(ctx, db, base+strings.Join(placeholders, ","), args)
		placeholders = placeholders[:0]
		args = args[:0]
		return err
	}

	for _, c := range dgCountries {
		placeholders = append(placeholders, "(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)")
		args = append(args,
			c.officialName, c.iso2, c.iso3, c.isoNumeric,
			c.currencyCode, c.currencyName,
			c.englishFormal, c.englishShort,
			c.capital, c.continent, c.languages,
			c.regionCode, c.regionName,
			c.subRegionCode, c.subRegionName,
			c.tld, c.isIndependent,
		)
		total++
		if len(placeholders) >= batchSz {
			if err := flush(); err != nil {
				return err
			}
		}
	}
	if err := flush(); err != nil {
		return err
	}
	fmt.Printf("Seeded countries: %d\n", total)
	return nil
}

func (t *DataGenTest) seedCities(ctx context.Context, db *sql.DB) error {
	const batchSz = 100
	base := "INSERT INTO `cities` (`name`,`country`,`subcountry`,`geonameid`,`country_iso3`) VALUES "

	placeholders := make([]string, 0, batchSz)
	args := make([]interface{}, 0, batchSz*5)
	total := 0

	flush := func() error {
		if len(placeholders) == 0 {
			return nil
		}
		err := insertBatch(ctx, db, base+strings.Join(placeholders, ","), args)
		placeholders = placeholders[:0]
		args = args[:0]
		return err
	}

	for _, c := range dgCities {
		placeholders = append(placeholders, "(?,?,?,?,?)")
		args = append(args, c.name, c.country, c.subcountry, c.geonameid, c.iso3)
		total++
		if len(placeholders) >= batchSz {
			if err := flush(); err != nil {
				return err
			}
		}
	}
	if err := flush(); err != nil {
		return err
	}
	fmt.Printf("Seeded cities: %d\n", total)
	return nil
}

func (t *DataGenTest) seedStreets(ctx context.Context, db *sql.DB) error {
	const batchSz = 100
	base := "INSERT INTO `streets` (`streetname`) VALUES "

	placeholders := make([]string, 0, batchSz)
	args := make([]interface{}, 0, batchSz)
	total := 0

	flush := func() error {
		if len(placeholders) == 0 {
			return nil
		}
		err := insertBatch(ctx, db, base+strings.Join(placeholders, ","), args)
		placeholders = placeholders[:0]
		args = args[:0]
		return err
	}

	for _, s := range dgStreets {
		placeholders = append(placeholders, "(?)")
		args = append(args, s.streetname)
		total++
		if len(placeholders) >= batchSz {
			if err := flush(); err != nil {
				return err
			}
		}
	}
	if err := flush(); err != nil {
		return err
	}
	fmt.Printf("Seeded streets: %d\n", total)
	return nil
}

func (t *DataGenTest) seedFirstNames(ctx context.Context, db *sql.DB) error {
	const batchSz = 100
	base := "INSERT INTO `firstname` (`name`,`gender`) VALUES "

	placeholders := make([]string, 0, batchSz)
	args := make([]interface{}, 0, batchSz*2)
	total := 0

	flush := func() error {
		if len(placeholders) == 0 {
			return nil
		}
		err := insertBatch(ctx, db, base+strings.Join(placeholders, ","), args)
		placeholders = placeholders[:0]
		args = args[:0]
		return err
	}

	for _, name := range dgFirstNamesMale {
		placeholders = append(placeholders, "(?,?)")
		args = append(args, name, 1)
		total++
		if len(placeholders) >= batchSz {
			if err := flush(); err != nil {
				return err
			}
		}
	}
	for _, name := range dgFirstNamesFemale {
		placeholders = append(placeholders, "(?,?)")
		args = append(args, name, 2)
		total++
		if len(placeholders) >= batchSz {
			if err := flush(); err != nil {
				return err
			}
		}
	}
	if err := flush(); err != nil {
		return err
	}
	fmt.Printf("Seeded firstnames: %d\n", total)
	return nil
}

func (t *DataGenTest) seedLastNames(ctx context.Context, db *sql.DB) error {
	const batchSz = 100
	base := "INSERT INTO `lastname` (`name`) VALUES "

	placeholders := make([]string, 0, batchSz)
	args := make([]interface{}, 0, batchSz)
	total := 0

	flush := func() error {
		if len(placeholders) == 0 {
			return nil
		}
		err := insertBatch(ctx, db, base+strings.Join(placeholders, ","), args)
		placeholders = placeholders[:0]
		args = args[:0]
		return err
	}

	for _, name := range dgLastNamesList {
		placeholders = append(placeholders, "(?)")
		args = append(args, name)
		total++
		if len(placeholders) >= batchSz {
			if err := flush(); err != nil {
				return err
			}
		}
	}
	if err := flush(); err != nil {
		return err
	}
	fmt.Printf("Seeded lastnames: %d\n", total)
	return nil
}

func (t *DataGenTest) seedTitles(ctx context.Context, db *sql.DB) error {
	const batchSz = 100
	base := "INSERT INTO `titles` (`name`,`bho`,`onlyman`,`alsofem`,`norder`,`id`) VALUES "

	placeholders := make([]string, 0, batchSz)
	args := make([]interface{}, 0, batchSz*6)
	total := 0

	flush := func() error {
		if len(placeholders) == 0 {
			return nil
		}
		err := insertBatch(ctx, db, base+strings.Join(placeholders, ","), args)
		placeholders = placeholders[:0]
		args = args[:0]
		return err
	}

	for _, ti := range dgTitles {
		placeholders = append(placeholders, "(?,?,?,?,?,?)")
		args = append(args, ti.name, ti.bho, ti.onlyman, ti.alsofem, ti.norder, ti.id)
		total++
		if len(placeholders) >= batchSz {
			if err := flush(); err != nil {
				return err
			}
		}
	}
	if err := flush(); err != nil {
		return err
	}
	fmt.Printf("Seeded titles: %d\n", total)
	return nil
}

func (t *DataGenTest) seedPopulation(ctx context.Context, db *sql.DB) error {
	const batchSz = 100
	base := "INSERT INTO `population` (`countryname`,`iso3`,`seriesname`,`seriescode`,`y2019`,`y2020`,`y2021`) VALUES "

	placeholders := make([]string, 0, batchSz)
	args := make([]interface{}, 0, batchSz*7)
	total := 0

	flush := func() error {
		if len(placeholders) == 0 {
			return nil
		}
		err := insertBatch(ctx, db, base+strings.Join(placeholders, ","), args)
		placeholders = placeholders[:0]
		args = args[:0]
		return err
	}

	for _, p := range dgPopulation {
		placeholders = append(placeholders, "(?,?,?,?,?,?,?)")
		args = append(args, p.countryname, p.iso3, p.seriesname, p.seriescode, p.y2019, p.y2020, p.y2021)
		total++
		if len(placeholders) >= batchSz {
			if err := flush(); err != nil {
				return err
			}
		}
	}
	if err := flush(); err != nil {
		return err
	}
	fmt.Printf("Seeded population: %d\n", total)
	return nil
}

func (t *DataGenTest) loadCities(db *sql.DB) ([]cityRow, error) {
	rows, err := db.QueryContext(context.Background(),
		"SELECT id, name, country_iso3, subcountry FROM cities")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cities []cityRow
	for rows.Next() {
		var c cityRow
		var iso3, subcountry sql.NullString
		if err := rows.Scan(&c.id, &c.name, &iso3, &subcountry); err != nil {
			return nil, err
		}
		c.iso3 = iso3.String
		c.subcountry = subcountry.String
		cities = append(cities, c)
	}
	return cities, rows.Err()
}

func (t *DataGenTest) generateAddresses(db *sql.DB, n int, cities []cityRow, workers int) {
	if t.Truncate {
		if _, err := db.ExecContext(context.Background(), "TRUNCATE TABLE `address_`"); err != nil {
			log.Errorf("truncate address_: %v", err)
		}
	}

	fmt.Printf("\nGenerating %d addresses with %d workers...\n", n, workers)

	streetNames := dgStreetNames

	var counter int64
	var wg sync.WaitGroup

	perWorker := n / workers

	for w := 0; w < workers; w++ {
		wg.Add(1)
		workerIdx := w
		start := workerIdx*perWorker + 1
		end := start + perWorker
		if workerIdx == workers-1 {
			end = n + 1
		}

		go func(workerIdx, start, end int) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerIdx)))
			ctx := context.Background()

			batchSz := t.BatchSize
			count := 0

			base := "INSERT INTO `address_` (`active`,`street`,`city`,`region`,`country_iso3`,`link_id`,`cityId`) VALUES "

			placeholders := make([]string, 0, batchSz)
			args := make([]interface{}, 0, batchSz*7)

			flush := func() error {
				if len(placeholders) == 0 {
					return nil
				}
				err := insertBatch(ctx, db, base+strings.Join(placeholders, ","), args)
				placeholders = placeholders[:0]
				args = args[:0]
				return err
			}

			for linkID := start; linkID < end; linkID++ {
				city := cities[rng.Intn(len(cities))]
				street := streetNames[rng.Intn(len(streetNames))] + " " + strconv.Itoa(rng.Intn(200)+1)
				active := 1
				if rng.Intn(10) == 0 {
					active = 0
				}

				placeholders = append(placeholders, "(?,?,?,?,?,?,?)")
				args = append(args, active, street, city.name, city.subcountry, city.iso3, linkID, city.id)
				count++

				if len(placeholders) >= batchSz {
					if err := flush(); err != nil {
						log.Errorf("[worker %d] insert address batch: %v", workerIdx, err)
					}
				}

				cur := atomic.AddInt64(&counter, 1)
				if cur%100_000 == 0 {
					fmt.Printf("[addresses] inserted %d / %d\n", cur, n)
				}
			}

			if err := flush(); err != nil {
				log.Errorf("[worker %d] flush address: %v", workerIdx, err)
			}

			fmt.Printf("[worker %d] inserted %d rows\n", workerIdx, count)
		}(workerIdx, start, end)
	}

	wg.Wait()
	fmt.Printf("Address generation complete: %d rows\n", n)
}

func (t *DataGenTest) generateUsers(db *sql.DB, n, numAddresses int, workers int) {
	if t.Truncate {
		if _, err := db.ExecContext(context.Background(), "TRUNCATE TABLE `users_`"); err != nil {
			log.Errorf("truncate users_: %v", err)
		}
	}

	fmt.Printf("\nGenerating %d users with %d workers...\n", n, workers)

	countryCodes := []string{"33", "44", "49", "39", "34", "351", "31", "32", "41", "43"}
	emailDomains := []string{
		"gmail.com", "yahoo.fr", "outlook.de", "hotmail.it", "protonmail.com",
		"gmx.de", "orange.fr", "libero.it", "correo.es", "btinternet.com",
		"wp.pl", "mail.ru",
	}

	var counter int64
	var wg sync.WaitGroup

	perWorker := n / workers

	for w := 0; w < workers; w++ {
		wg.Add(1)
		workerIdx := w
		start := int64(workerIdx*perWorker + 1)
		end := start + int64(perWorker)
		if workerIdx == workers-1 {
			end = int64(n) + 1
		}

		go func(workerIdx int, start, end int64) {
			defer wg.Done()
			rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(workerIdx)*1000))
			ctx := context.Background()

			batchSz := t.BatchSize
			count := 0

			base := "INSERT INTO `users_` (`name`,`lastname`,`age`,`phone`,`registration_date`,`address_id`,`email`,`active`,`gender`) VALUES "

			placeholders := make([]string, 0, batchSz)
			args := make([]interface{}, 0, batchSz*9)

			flush := func() error {
				if len(placeholders) == 0 {
					return nil
				}
				err := insertBatch(ctx, db, base+strings.Join(placeholders, ","), args)
				placeholders = placeholders[:0]
				args = args[:0]
				return err
			}

			now := time.Now()
			tenYearsAgo := now.AddDate(-10, 0, 0)
			rangeSeconds := int64(now.Sub(tenYearsAgo).Seconds())

			for userID := start; userID < end; userID++ {
				var gender string
				var firstName string
				if rng.Intn(2) == 0 {
					gender = "m"
					firstName = dgMaleNames[rng.Intn(len(dgMaleNames))]
				} else {
					gender = "f"
					firstName = dgFemaleNames[rng.Intn(len(dgFemaleNames))]
				}

				lastName := dgLastNamesList[rng.Intn(len(dgLastNamesList))]
				age := rng.Intn(75) + 16
				cc := countryCodes[rng.Intn(len(countryCodes))]

				var phoneSuffix strings.Builder
				for i := 0; i < 9; i++ {
					phoneSuffix.WriteByte(byte('0' + rng.Intn(10)))
				}
				phone := "+" + cc + phoneSuffix.String()

				regOffset := rng.Int63n(rangeSeconds)
				regDate := tenYearsAgo.Add(time.Duration(regOffset) * time.Second)
				regDateStr := regDate.Format("2006-01-02 15:04:05")

				addrID := rng.Int63n(int64(numAddresses)) + 1

				emailBase := strings.ToLower(firstName) + "." + strings.ToLower(lastName) +
					strconv.FormatInt(userID, 36)
				domain := emailDomains[rng.Intn(len(emailDomains))]
				email := emailBase + "@" + domain

				active := 1
				if rng.Intn(20) == 0 {
					active = 0
				}

				placeholders = append(placeholders, "(?,?,?,?,?,?,?,?,?)")
				args = append(args, firstName, lastName, age, phone, regDateStr, addrID, email, active, gender)
				count++

				if len(placeholders) >= batchSz {
					if err := flush(); err != nil {
						log.Errorf("[worker %d] insert user batch: %v", workerIdx, err)
					}
				}

				cur := atomic.AddInt64(&counter, 1)
				if cur%200_000 == 0 {
					fmt.Printf("[users] inserted %d / %d\n", cur, n)
				}
			}

			if err := flush(); err != nil {
				log.Errorf("[worker %d] flush users: %v", workerIdx, err)
			}

			fmt.Printf("[worker %d] inserted %d rows\n", workerIdx, count)
		}(workerIdx, start, end)
	}

	wg.Wait()
	fmt.Printf("User generation complete: %d rows\n", n)
}

func insertBatch(ctx context.Context, db *sql.DB, query string, args []interface{}) error {
	_, err := db.ExecContext(ctx, query, args...)
	return err
}

func (t *DataGenTest) ShowHelp() string {
	var sb strings.Builder
	sb.WriteString(t.TestBase.ShowHelp())
	sb.WriteString("\n****************************************\n DataGen module\n")
	sb.WriteString("rowsNumber   [rowsNumber=2000000]  total users to generate\n")
	sb.WriteString("batchSize    [batchSize=500]        rows per INSERT statement\n")
	sb.WriteString("workers      [workers=8]            parallel goroutines\n")
	sb.WriteString("truncate     [truncate=true]        truncate tables before loading\n")
	return sb.String()
}
