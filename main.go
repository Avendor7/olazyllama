package main

import (
	"log"

	gocui "github.com/jroimartin/gocui"
)

// layout is the GUI layout manager. gocui calls this to (re)draw the screen.
// It creates a single view named "main" that fills the entire terminal and
// writes a short welcome message the first time the view is created.
func layout(g *gocui.Gui) error {
	// Get the terminal width/height
	maxX, maxY := g.Size()

	// Try to create or retrieve a view named "main" that spans the whole screen.
	// When SetView creates a new view it returns gocui.ErrUnknownView — that's
	// the signal that we should initialize the view (title, wrap, initial text).
	if v, err := g.SetView("main", 0, 0, maxX-1, maxY-1); err != nil {
		// If the error is anything other than ErrUnknownView, propagate it.
		if err != gocui.ErrUnknownView {
			return err
		}

		// Configure the view the first time it's created.
		v.Title = "Ollama LazyCLI (hello)"
		v.Wrap = true // enable line wrapping for long text

		// Write an initial message. The Write result and error are ignored on
		// purpose because this is just a best-effort initial seed message.
		_, _ = v.Write([]byte("Hello, Ollama!\n\nPress Ctrl-Q or Ctrl-C to quit."))
	}

	// Returning nil means layout succeeded. gocui will call layout again as
	// needed (e.g., when the terminal is resized).
	return nil
}

// quit is a keybinding handler that tells gocui to exit the main loop.
// Handlers have the signature func(*gocui.Gui, *gocui.View) error.
func quit(g *gocui.Gui, v *gocui.View) error {
	// Returning gocui.ErrQuit instructs gocui's MainLoop to stop.
	return gocui.ErrQuit
}

// main initializes the gocui GUI, registers the layout and keybindings, and
// runs the main event loop. The program exits when the user presses Ctrl-Q or
// Ctrl-C, or when a fatal error occurs during setup.
func main() {
	// Create a new GUI in normal (text) output mode.
	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Fatalf("failed to init gui: %v", err)
	}
	// Ensure resources are cleaned up on exit.
	defer g.Close()

	// Tell gocui to use our layout function to manage/redraw the views.
	g.SetManagerFunc(layout)

	// Register keybindings. Here we bind Ctrl-Q and Ctrl-C to the quit handler.
	if err := g.SetKeybinding("", gocui.KeyCtrlQ, gocui.ModNone, quit); err != nil {
		log.Fatalf("keybinding failed: %v", err)
	}
	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		log.Fatalf("keybinding failed: %v", err)
	}

	// Start the main event loop. It will block until an error occurs or a
	// handler returns gocui.ErrQuit. If the error is ErrQuit we treat that as
	// a normal shutdown and don't log it as a fatal error.
	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Fatalf("main loop error: %v", err)
	}
}