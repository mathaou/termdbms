# termdbms: A TUI for viewing and editing databases, written in pure Go

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
- Better editing
- No-style mode for terminal emulators that don't support ANSI / low power machines.
- Support for 32-bit machines.

#### 
<details>
    <summary>How To Build</summary>

##### Linux

    GOOS=linux GOARCH=amd64 go build

##### ARM (runs kind of slow depending on the specs of the system)

    GOOS=linux GOARCH=arm GOARM=7 go build

##### Windows

    GOOS=windows GOARCH=amd64 go build

##### OSX

    GOOS=darwin GOARCH=amd64 go build

</details>

#### Terminal settings
Whatever terminal emulator used should support ANSI escape sequences. If there is an option for 256 color mode, enable it.

#### Known Issues
 - The headers wig out sometimes in selection mode
 - Possible race conditions with getting data initialized, only happens when debugging?
 - Serializing a numeric string change (like "1234") sometimes appends a decimal at the end, even though go recognizes it as a string when serializing. This is likely a bug at the database driver level, or I am not good at this.
 - Mouse down does not work in Windows Terminal, but it does work in Command Prompt.

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
    [LEFT/H and RIGHT/L] to navigate columns if there are more than the screen allows.
        Also to control the cursor of the text editor in edit mode.
    [M(scroll up) and N(scroll down)] to scroll manually
	[Q or CTRL+C] to quit program
    [B] to toggle borders!
    [C] to expand column
    [P] in selection mode to write cell to file
	[ESC] to exit full screen view, or to enter edit mode
###### EDIT MODE (cosmetic until serialization is working)
    [ESC] to enter edit mode with no pre-loaded text input from selection
    When a cell is selected, press [:] to enter edit mode with selection pre-loaded
    The text field in the header will be populated with the selected cells text. Modifications can be made freely.
    [ESC] to clear text field in edit mode
    [ENTER] to save text. Anything besides one of the reserved strings below will overwrite the current cell.
    [R] to redo actions, if applicable.
    [U] to undo actions, if applicable.
    [:q] to exit edit mode
    [:s] to save database to a new file
    [:s!] to overwrite original database file
    [:h] to display help text
