package main

import (
	"database/sql"
	"flag"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	_ "modernc.org/sqlite"
	"os"
	"strings"
	. "termdbms/viewer"
)

const (
	debugPath = "/home/mfarstad/Desktop/megastore.db" // set to whatever hardcoded path for testing
)

func main() {
	var path string
	var help bool

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
		if argLength > 2 || argLength == 0 {
			fmt.Printf("ERROR: Invalid number of arguments supplied: %d\n", argLength)
			flag.Usage()
			os.Exit(1)
		}

		// flags declaration using flag package
		flag.StringVar(&path, "p", "", "Specify username. Default is root")
		flag.BoolVar(&help, "h", false, "Specify pass. Default is password")

		flag.Parse()

		if flag.NFlag() == 0 {
			fmt.Printf("ERROR: Path to database file must be given with the -p flag.\n")
			flag.Usage()
			os.Exit(1)
		}

		if help {
			flag.Usage()
			os.Exit(0)
		}

		if path != "" && !IsUrl(path) {
			fmt.Printf("ERROR: Invalid path %s\n", path)
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
	p := tea.NewProgram(InitialModel,
		tea.WithAltScreen(),
		tea.WithMouseAllMotion())

	if err := p.Start(); err != nil {
		fmt.Printf("ERROR: Error initializing the sqlite viewer: %v", err)
		os.Exit(1)
	}
}
