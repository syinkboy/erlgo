package erlc

var Package = struct {
	Name        string
	Version     string
	Description string
	License     string
	Files       []string
	Dependencies []string
}{
	Name:        "NickIsADev/erlua-go",
	Version:     "2.0.0",
	Description: "A library providing dynamic ratelimiting, custom functions, and easy access to the ER:LC API.",
	License:     "MIT",
	Files:       []string{"*.go"},
	Dependencies: []string{
		"net/http",
		"encoding/json",
		"sync",
		"time",
	},
}