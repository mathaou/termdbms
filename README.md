# sqlite-tui:  A TUI for viewing sqlite databases, written in Go

### made with mattn/go-sqlite3, charmbracelet/bubbletea, and charmbracelet/lipgloss

#### Works with keyboard:

![Keyboard Control](https://i.imgur.com/ryDLroi.gif)

#### ... And mouse!

![Mouse Control](https://i.imgur.com/O8DT9q5.gif)

#### NOTE: Mouse controls don't work for remote sessions like serial or SSH. 
xterm-256 color mode must be enabled in the settings in order for color highlighting to function in these environments as well.
MobaXterm, GitBash, and the most recent Windows terminal should all support these on Windows. Linux supports out of the box.

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
	[ESC] to exit full screen view
