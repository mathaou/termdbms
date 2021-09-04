# tuitty:  A TUI for viewing and (eventually) editing databases, written in Go

###### Database Support
    SQLite

### made with modernc.org/sqlite, charmbracelet/bubbletea, and charmbracelet/lipgloss

#### Works with keyboard:

![Keyboard Control](https://i.imgur.com/ryDLroi.gif)

#### ... And mouse!

![Mouse Control](https://i.imgur.com/O8DT9q5.gif)


#### Building (generally a go build should be enough, architecture included for completeness)

##### Linux

    GOOS=linux GOARCH=amd64 go build

##### Windows

    GOOS=windows GOARCH=amd64 go build

##### OSX

    GOOS=darwin GOARCH=amd64 go build

#### NOTE: Mouse controls don't work for remote sessions like serial or SSH. 
xterm-256 color mode must be enabled in the settings in order for color highlighting to function in these environments as well.
MobaXterm, GitBash, and the most recent Windows terminal should all support these on Windows. Linux supports out of the box.

#### Known issues:
Large databases (tens of thousands of rows) make it slow sometimes. PRs open for optimization/ batching etc...
Headers wig out sometimes in column expansion or selection view.

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
	[M(scroll up) and N(scroll down)] to scroll manually
	[Q or CTRL+C] to quit program
    [B] to toggle borders!
    [C] to expand column
    [P] in selection mode to write cell to file
	[ESC] to exit full screen view
