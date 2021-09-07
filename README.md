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

#### Building (generally a go build should be enough, architecture included for completeness)

##### Linux

    GOOS=linux GOARCH=amd64 go build

##### Windows

    GOOS=windows GOARCH=amd64 go build

##### OSX

    GOOS=darwin GOARCH=amd64 go build

#### Terminal settings
Whatever terminal emulator used should support ANSI escape sequences. If there is an option for 256 color mode, enable it.

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
    [:!s] to overwrite original database file
    [:h] to display help text
