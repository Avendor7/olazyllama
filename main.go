// Package main provides a terminal-based GUI application for managing Ollama models.
// It displays installed and running models in a split-pane interface with status updates.
package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jroimartin/gocui"

	"olazyllama/internal/ollama"
)

// View names for the GUI layout
const (
	viewInstalled = "installed" // Left pane showing installed models
	viewRunning   = "running"   // Right pane showing running models
	viewStatus    = "status"    // Bottom pane showing status messages
)

// App represents the main application state and GUI components.
// It manages the terminal interface, Ollama client connection, and model data.
type App struct {
	gui     *gocui.Gui     // Terminal GUI instance
	client  *ollama.Client // Ollama API client
	baseURL string         // Base URL for Ollama server

	installed []ollama.Model // List of locally installed models
	running   []ollama.Model // List of currently running models

	statusLines []string // Recent status messages for display
}

// newApp creates a new App instance with the specified Ollama server URL.
// If baseURL is empty, it defaults to the standard Ollama localhost address.
func newApp(baseURL string) *App {
	return &App{
		client:  ollama.NewClient(baseURL),
		baseURL: baseURL,
	}
}

// logf logs a formatted message to the status view.
// Messages are stored in a rolling buffer of the last 5 lines.
func (a *App) logf(format string, args ...any) {
	line := fmt.Sprintf(format, args...)
	a.statusLines = append(a.statusLines, line)
	if len(a.statusLines) > 5 {
		a.statusLines = a.statusLines[len(a.statusLines)-5:]
	}
	a.safeUpdate(func(g *gocui.Gui) error {
		if v, err := g.View(viewStatus); err == nil {
			v.Clear()
			fmt.Fprint(v, strings.Join(a.statusLines, " | "))
		}
		return nil
	})
}

// safeUpdate safely executes a GUI update function if the GUI is initialized.
// This prevents panics when trying to update the GUI before it's ready.
func (a *App) safeUpdate(fn func(*gocui.Gui) error) {
	if a.gui == nil {
		return
	}
	a.gui.Update(fn)
}

// layout sets up the GUI layout with three views: installed models (left),
// running models (right), and status messages (bottom).
func (a *App) layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()
	statusH := 2
	bodyH := maxY - statusH
	if bodyH < 3 {
		bodyH = maxY
	}

	halfX := maxX / 2

	if v, err := g.SetView(viewInstalled, 0, 0, halfX-1, bodyH-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Installed Models"
		v.Wrap = false
	}

	if v, err := g.SetView(viewRunning, halfX, 0, maxX-1, bodyH-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Running (ollama ps)"
		v.Wrap = false
	}

	if v, err := g.SetView(viewStatus, 0, bodyH, maxX-1, maxY-1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = "Status"
		fmt.Fprint(v, "Ready")
	}

	a.drawInstalled()
	a.drawRunning()
	return nil
}

// drawInstalled updates the installed models view with the current list.
// Shows model names and sizes in a formatted display.
func (a *App) drawInstalled() {
	a.safeUpdate(func(g *gocui.Gui) error {
		v, err := g.View(viewInstalled)
		if err != nil {
			return nil
		}
		v.Clear()
		if len(a.installed) == 0 {
			fmt.Fprintln(v, "(no models installed)")
			return nil
		}
		for _, m := range a.installed {
			line := m.Name
			if m.Size > 0 {
				line = fmt.Sprintf("%-40s  %s", m.Name, ollama.HumanSize(m.Size))
			}
			fmt.Fprintln(v, line)
		}
		return nil
	})
}

// drawRunning updates the running models view with currently active models.
func (a *App) drawRunning() {
	a.safeUpdate(func(g *gocui.Gui) error {
		v, err := g.View(viewRunning)
		if err != nil {
			return nil
		}
		v.Clear()
		if len(a.running) == 0 {
			fmt.Fprintln(v, "(nothing running)")
			return nil
		}
		for _, m := range a.running {
			fmt.Fprintln(v, m.Name)
		}
		return nil
	})
}

// refreshAll fetches the latest model data from Ollama in a background goroutine.
// Updates both installed and running model lists with error handling.
func (a *App) refreshAll() {
	a.logf("Refreshing...")
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		installed, err1 := a.client.ListLocalModels(ctx)
		running, err2 := a.client.ListRunning(ctx)

		a.safeUpdate(func(g *gocui.Gui) error {
			if err1 != nil {
				a.logf("Installed: %v", err1)
			} else {
				a.installed = installed
			}
			if err2 != nil {
				a.logf("Running: %v", err2)
			} else {
				a.running = running
			}
			a.drawInstalled()
			a.drawRunning()
			if err1 == nil && err2 == nil {
				a.logf("Refreshed")
			}
			return nil
		})
	}()
}

// bindKeys sets up keyboard shortcuts for the application.
// Supports Ctrl+C, q (quit), r, and Ctrl+R (refresh).
func (a *App) bindKeys() error {
	if err := a.gui.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, a.onQuit); err != nil {
		return err
	}
	if err := a.gui.SetKeybinding("", 'q', gocui.ModNone, a.onQuit); err != nil {
		return err
	}
	if err := a.gui.SetKeybinding("", 'r', gocui.ModNone, a.onRefresh); err != nil {
		return err
	}
	if err := a.gui.SetKeybinding("", gocui.KeyCtrlR, gocui.ModNone, a.onRefresh); err != nil {
		return err
	}
	return nil
}

// onQuit handles the quit key binding and terminates the application.
func (a *App) onQuit(_ *gocui.Gui, _ *gocui.View) error {
	return gocui.ErrQuit
}

// onRefresh handles the refresh key binding and triggers a data refresh.
func (a *App) onRefresh(_ *gocui.Gui, _ *gocui.View) error {
	a.refreshAll()
	return nil
}

// main initializes and runs the Ollama model manager GUI application.
// Sets up the terminal interface, binds keyboard shortcuts, and starts the main loop.
func main() {
	app := newApp("http://localhost:11434")

	g, err := gocui.NewGui(gocui.OutputNormal)
	if err != nil {
		log.Fatalf("failed to init gui: %v", err)
	}
	defer g.Close()
	app.gui = g

	g.SetManagerFunc(app.layout)
	if err := app.bindKeys(); err != nil {
		log.Fatalf("keybindings: %v", err)
	}

	app.refreshAll()

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Fatalf("main loop error: %v", err)
	}
}
