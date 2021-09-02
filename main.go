package main

import (
	"database/sql"
	"flag"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	_ "github.com/mattn/go-sqlite3"
	"io/ioutil"
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
	debugPath = ""
)

func init() {
	initialModel = GetNewModel()

	// We keep one connection pool per database.
	dbMutex = sync.Mutex{}
	dbs = make(map[string]*sql.DB)
}

func IsUrl(fp string) bool {
	// Check if file already exists
	if _, err := os.Stat(fp); err == nil {
		return true
	}

	// Attempt to create it
	var d []byte
	if err := ioutil.WriteFile(fp, d, 0644); err == nil {
		os.Remove(fp) // And delete it
		return true
	}

	return false
}

func main() {
	var path string
	var help bool

	debug := false
	if !debug {
		flag.Usage = func(){
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

	db := getDatabaseForFile(path)
	defer db.Close()

	setModel(c, db)

	p := tea.NewProgram(initialModel,
		tea.WithAltScreen(),
		tea.WithMouseAllMotion())

	if err := p.Start(); err != nil {
		fmt.Printf("Error initializing the sqlite viewer: %v", err)
		os.Exit(1)
	}
}

func setModel(c *sql.Rows, db *sql.DB) {
	var err error
	indexMap := 0

	rows, err := db.Query(getTableNamesQuery)
	if err != nil {
		fmt.Printf("%v", err)
		os.Exit(1)
	}
	defer rows.Close()

	for rows.Next() {
		var tableName string
		rows.Scan(&tableName)

		var statement strings.Builder
		statement.WriteString("select * from ")
		statement.WriteString(tableName)

		if c != nil {
			c.Close()
			c = nil
		}
		c, err = db.Query(statement.String())
		if err != nil {
			panic(err)
		}

		names, _ := c.Columns()
		m := make(map[string][]interface{})

		for c.Next() { // each row of the table
			columns := make([]interface{}, len(names))
			columnPointers := make([]interface{}, len(names))
			// init interface array
			for i, _ := range columns {
				columnPointers[i] = &columns[i]
			}

			c.Scan(columnPointers...)

			for i, colName := range names {
				val := columnPointers[i].(*interface{})
				m[colName] = append(m[colName], *val)
			}
		}

		indexMap++
		initialModel.Table[tableName] = m
		initialModel.TableHeaders[tableName] = names
		initialModel.TableIndexMap[indexMap] = tableName
	}

	initialModel.TableSelection = 1
}

func getDatabaseForFile(database string) *sql.DB {
	dbMutex.Lock()
	defer dbMutex.Unlock()
	if db, ok := dbs[database]; ok {
		return db
	}
	db, err := sql.Open("sqlite3", database)
	if err != nil {
		panic(err)
	}
	dbs[database] = db
	return db
}
