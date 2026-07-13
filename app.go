package main

import "net/http"

// App wires configuration, the directory layout and a shared HTTP client
// behind every sv command. There is no global state: everything commands
// need hangs off this struct, which makes each piece testable in isolation.
type App struct {
	cfg    Config
	paths  Paths
	client *http.Client
}

// newApp builds the App and ensures the ~/.sv directory layout exists.
// The client carries no global timeout; every request sets a context
// deadline suited to what it does (short for APIs, long for downloads).
func newApp() (*App, error) {
	paths, err := newPaths()
	if err != nil {
		return nil, err
	}
	return &App{
		cfg:    loadConfig(),
		paths:  paths,
		client: &http.Client{},
	}, nil
}
