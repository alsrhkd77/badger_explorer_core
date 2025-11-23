package main

import (
	"flag"
	"fmt"
	"os"

	"badger_explorer_core/api"
	"badger_explorer_core/config"
	"badger_explorer_core/db"
	"badger_explorer_core/locale"
	"badger_explorer_core/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	standalone := flag.Bool("standalone", true, "Run in standalone TUI mode")
	flag.Parse()

	// Load Config
	cfg, err := config.LoadConfig("config.json")
	if err != nil {
		// If fails, use default, but maybe warn?
		// For now, just proceed with defaults (which LoadConfig returns on error if not exist)
	}

	// Init Locale
	if err := locale.Init(cfg.Localization); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to init locale: %v\n", err)
	}

	// Init DB Client
	dbClient := db.NewDBClient()
	defer dbClient.Close()

	if *standalone {
		// TUI Mode
		p := tea.NewProgram(ui.NewAppModel(cfg, dbClient), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Printf("Alas, there's been an error: %v", err)
			os.Exit(1)
		}
	} else {
		// Subprocess Mode
		handler := api.NewHandler(dbClient, os.Stdout)
		handler.Run(os.Stdin)
	}
}
