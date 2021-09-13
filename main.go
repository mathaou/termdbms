package main

import (
	"database/sql"
	"flag"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	_ "modernc.org/sqlite"
	"os"
	"strings"
	. "termdbms/viewer"
)

type DatabaseType string

const (
	debugPath = "" // set to whatever hardcoded path for testing
)

const (
	DatabaseSQLite DatabaseType = "sqlite"
	DatabaseMySQL  DatabaseType = "mysql"
)

var (
	path         string
	databaseType string
	theme        string
	help         bool
	ascii        bool
)

func main() {
	debug := debugPath != ""
	// if not debug, then this section parses and validates cmd line arguments
	if !debug {
		flag.Usage = func() {
			help := GetHelpText()
			lines := strings.Split(help, "\n")
			for _, v := range lines {
				println(v)
			}
		}

		argLength := len(os.Args[1:])
		if argLength > 4 || argLength == 0 {
			fmt.Printf("ERROR: Invalid number of arguments supplied: %d\n", argLength)
			flag.Usage()
			os.Exit(1)
		}

		// flags declaration using flag package
		flag.StringVar(&databaseType, "d", string(DatabaseSQLite), "Specifies the SQL driver to use. Defaults to SQLite.")
		flag.StringVar(&path, "p", "", "Path to the database file.")
		flag.StringVar(&theme, "t", "default", "sets the color theme of the app.")
		flag.BoolVar(&help, "h", false, "Prints the help message.")
		flag.BoolVar(&ascii, "a", false, "Denotes that the app should render with minimal styling to remove ANSI sequences.")

		flag.Parse()

		handleFlags()
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

	t := ""
	for i, v := range ValidThemes {
		if theme == v {
			SelectedTheme = i
			break
		}
	}

	if t == "" {
		theme = "default"
	}

	// gets a sqlite instance for the database file
	if exists, _ := FileExists(path); exists {
		fmt.Printf("ERROR: Database file could not be found at %s\n", path)
		os.Exit(1)
	}

	if valid, _ := Exists(HiddenTmpDirectoryName); valid {
		os.RemoveAll(HiddenTmpDirectoryName)
	}

	os.Mkdir(HiddenTmpDirectoryName, 0777)

	// steps
	// make a copy of the database file, load this
	dst, _, _ := CopyFile(path)
	// keep a track of the original file name
	db := GetDatabaseForFile(dst)
	defer func() {
		if db == nil {
			db.Close()
		}
	}()

	// initializes the model used by bubbletea
	m := GetNewModel(dst, db)
	InitialModel = &m
	InitialModel.InitialFileName = path
	InitialModel.SetModel(c, db)

	// creates the program
	Program = tea.NewProgram(InitialModel,
		tea.WithAltScreen(),
		tea.WithMouseAllMotion())

	if err := Program.Start(); err != nil {
		fmt.Printf("ERROR: Error initializing the sqlite viewer: %v", err)
		os.Exit(1)
	}
}

func handleFlags() {
	if path == "" {
		fmt.Printf("ERROR: no path for database.\n")
		flag.Usage()
		os.Exit(1)
	}

	if help {
		flag.Usage()
		os.Exit(0)
	}

	if ascii {
		Ascii = true
		lipgloss.SetColorProfile(termenv.Ascii)
	}

	if path != "" && !IsUrl(path) {
		fmt.Printf("ERROR: Invalid path %s\n", path)
		flag.Usage()
		os.Exit(1)
	}

	if databaseType != string(DatabaseMySQL) &&
		databaseType != string(DatabaseSQLite) {
		fmt.Printf("Invalid database driver specified: %s", databaseType)
		os.Exit(1)
	}

	DriverString = databaseType
}