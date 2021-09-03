package main

import (
	"database/sql"
	"flag"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	_ "modernc.org/sqlite"
	"os"
	. "sqlite3-viewer/viewer"
	"strings"
	"sync"
)

var (
	initialModel TuiModel
	dbMutex      sync.Mutex
	dbs          map[string]*sql.DB
)

const (
	getTableNamesQuery = "SELECT name FROM sqlite_master WHERE type='table'"
	debugPath          = "C:\\Users\\matth\\OneDrive\\Desktop\\chinook.db" // set to whatever hardcoded path for testing
)

func init() {
	initialModel = GetNewModel()

	// We keep one connection pool per database.
	dbMutex = sync.Mutex{}
	dbs = make(map[string]*sql.DB)
}

func main() {
	var path string
	var help bool

	debug := true
	// if not debug, then this section parses and validates cmd line arguments
	if !debug {
		flag.Usage = func() {
			fmt.Println("NOTE: Mouse controls don't work for remote sessions like serial or SSH. " +
				"\nxterm-256 color mode must be enabled in the settings in order for color highlighting to function in " +
				"these environments as well.\n" +
				"MobaXterm, GitBash, and the most recent Windows terminal should all support these on Windows. Linux supports out of the box.")
			fmt.Println("Help:")
			fmt.Println("\t-p\tdatabase path (absolute)")
			fmt.Println("\t-h\tprints this message")
			fmt.Println("Controls:")
			fmt.Println("MOUSE")
			fmt.Println("\tScroll up + down to navigate table")
			fmt.Println("\tMove cursor to select cells for full screen viewing")
			fmt.Println("KEYBOARD")
			fmt.Println("\t[WASD] to move around cells")
			fmt.Println("\t[ENTER] to select selected cell for full screen view")
			fmt.Println("\t[UP/K and DOWN/J] to navigate schemas")
			fmt.Println("\t[M(scroll up) and N(scroll down)] to scroll manually")
			fmt.Println("\t[Q or CTRL+C] to quit program")
			fmt.Println("\t[B] to toggle borders!")
			fmt.Println("\t[ESC] to exit full screen view")
		}

		argLength := len(os.Args[1:])
		if argLength > 2 || argLength == 0 {
			fmt.Printf("Invalid number of arguments supplied: %d\n", argLength)
			flag.Usage()
			os.Exit(1)
		}

		// flags declaration using flag package
		flag.StringVar(&path, "p", "", "Specify username. Default is root")
		flag.BoolVar(&help, "h", false, "Specify pass. Default is password")

		flag.Parse()

		if help {
			flag.Usage()
			os.Exit(0)
		}

		if path != "" && !IsUrl(path) {
			fmt.Printf("Invalid path %s\n", path)
			flag.Usage()
			os.Exit(1)
		}
	}

	var c *sql.Rows
	defer func() {
		if c != nil {
			c.Close()
		}
	}()

	if debug {
		path = debugPath
	}

	// gets a sqlite instance for the database file
	db := getDatabaseForFile(path)
	defer db.Close()

	// initializes the model used by bubbletea
	setModel(c, db)

	// creates the program
	p := tea.NewProgram(initialModel,
		tea.WithAltScreen(),
		tea.WithoutCatchPanics(),
		tea.WithMouseAllMotion())

	if err := p.Start(); err != nil {
		fmt.Printf("Error initializing the sqlite viewer: %v", err)
		os.Exit(1)
	}
}

// setModel creates a model to be used by bubbletea using some golang wizardry
func setModel(c *sql.Rows, db *sql.DB) {
	var err error
	indexMap := 0

	// gets all the schema names of the database
	rows, err := db.Query(getTableNamesQuery)
	if err != nil {
		fmt.Printf("%v", err)
		os.Exit(1)
	}
	defer rows.Close()

	// for each schema
	for rows.Next() {
		var schemaName string
		rows.Scan(&schemaName)

		// couldn't get prepared statements working and gave up because it was very simple
		var statement strings.Builder
		statement.WriteString("select * from ")
		statement.WriteString(schemaName)

		if c != nil {
			c.Close()
			c = nil
		}
		c, err = db.Query(statement.String())
		if err != nil {
			panic(err)
		}

		columnNames, _ := c.Columns()
		columnValues := make(map[string][]interface{})

		for c.Next() { // each row of the table
			// golang wizardry
			columns := make([]interface{}, len(columnNames))
			columnPointers := make([]interface{}, len(columnNames))
			// init interface array
			for i, _ := range columns {
				columnPointers[i] = &columns[i]
			}

			c.Scan(columnPointers...)

			i := 0
			for _, colName := range columnNames {
				if colName == "" {
					continue
				}
				val := columnPointers[i].(*interface{})
				columnValues[colName] = append(columnValues[colName], *val)
				i++
			}
		}

		// onto the next schema
		indexMap++
		initialModel.Table[schemaName] = columnValues       // data for schema, organized by column
		initialModel.TableHeaders[schemaName] = columnNames // headers for the schema, for later reference
		// mapping between schema and an int ( since maps aren't deterministic), for later reference
		initialModel.TableIndexMap[indexMap] = schemaName
	}

	// set the first table to be initial view
	initialModel.TableSelection = 7
}

// getDatabaseForFile does what you think it does
func getDatabaseForFile(database string) *sql.DB {
	dbMutex.Lock()
	defer dbMutex.Unlock()
	if db, ok := dbs[database]; ok {
		return db
	}
	db, err := sql.Open("sqlite", database)
	if err != nil {
		panic(err)
	}
	dbs[database] = db
	return db
}
