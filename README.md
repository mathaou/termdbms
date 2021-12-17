# termdbms:  A TUI for viewing and (eventually) editing databases, written in Go

###### Database Support
    SQLite

### made with modernc.org/sqlite, charmbracelet/bubbletea, and charmbracelet/lipgloss

#### Works with keyboard:

![Keyboard Control](https://i.imgur.com/ryDLroi.gif)

#### ... And mouse!

![Mouse Control](https://i.imgur.com/O8DT9q5.gif)

#### Roadmap

- Run SQL queries and display results
- Add/remove rows/columns/cells
- Rename anything
- Filter tables by fuzzy search
- MySQL/ PostgreSQL support
- Edit during selection mode!

#### Building (generally a go build should be enough, architecture included for completeness)

##### Linux

    GOOS=linux GOARCH=amd64 go build

##### Armv7

    // I tried getting modernc.org/sqlite working, but it gave me tons of errors.
    // this set up got me a working binary on an ARM device with mattn/go-sqlite3
    // you will need to fix the import in main.go accordingly, and update the driver
    // string in database.go 
    GOOS="linux" GOARCH="arm" GOARM=7 CGO_ENABLED=1 go build

##### Windows

    GOOS=windows GOARCH=amd64 go build

##### OSX

    GOOS=darwin GOARCH=amd64 go build

#### Terminal settings
Whatever terminal emulator used should support ANSI escape sequences. If there is an option for 256 color mode, enable it.

#### Known Issues
 - The headers wig out sometimes in selection mode
 - Possible race conditions with getting data initialized, only happens when debugging?
 - Serializing a numeric string change (like "1234") sometimes appends a decimal at the end, even though go recognizes it as a string when serializing. This is likely a bug at the database driver level, or I am not good at this.

##### Help:
	-p	database path (absolute)
	-h	prints this message
##### Controls:
###### MOUSE
	Scroll up + down to navigate table
	Move cursor to select cells for full screen viewing
###### KEYBOARD
	[WASD] to move around cells
	[ENTER] to select selected cell for full screen view
	[UP/K and DOWN/J] to navigate schemas
    [LEFT/H and RIGHT/L] to navigate columns if there are more than the screen allows
	[M(scroll up) and N(scroll down)] to scroll manually
	[Q or CTRL+C] to quit program
    [B] to toggle borders!
    [C] to expand column
    [P] in selection mode to write cell to file
	[ESC] to exit full screen view
###### EDIT MODE (cosmetic until serialization is working)
    When a cell is selected, press [:] to enter edit mode
    The text field in the header will be populated with the selected cells text. Modifications can be made freely.
    [ESC] to clear text field
    [ENTER] to save text. Anything besides one of the reserved strings below will overwrite the current cell.
    [R] to redo actions, if applicable.
    [U] to undo actions, if applicable.
    [:q] to exit edit mode
    [:s] to save database to a new file
    [:s!] to overwrite original database file
    [:h] to display help text
