// Command winze-observatory serves the ambient fleet dashboard as a standalone
// local web app — the not-a-Claude-artifact version. It reads the live corpus
// of each registered instance (or the dirs you pass) and serves an always-on
// organism view at http://localhost:7777. Point a spare monitor at it.
//
//	winze-observatory                 # fleet from the metabolize registry (or CWD)
//	winze-observatory --open .        # this corpus, open a browser
//	winze-observatory --addr :9000 a b c
package main

import (
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"strings"

	"github.com/justinstimatze/winze/internal/events"
)

//go:embed index.html
var indexHTML []byte

//go:embed app.js
var appJS []byte

func main() {
	// Default to loopback: the dashboard can render private memory stores, so it
	// must not be exposed on the LAN unless explicitly asked (--addr 0.0.0.0:PORT).
	addr := flag.String("addr", "127.0.0.1:7777", "listen address")
	open := flag.Bool("open", false, "open the dashboard in a browser on start")
	flag.Parse()
	dirs := flag.Args()

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(indexHTML)
	})
	mux.HandleFunc("/app.js", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Write(appJS)
	})
	mux.HandleFunc("/api/fleet.json", func(w http.ResponseWriter, r *http.Request) {
		data, err := buildFleet(dirs)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		json.NewEncoder(w).Encode(data)
	})
	// The fleet event stream: real metabolism phases and meld/unmeld. The page
	// polls this and animates only genuine events (a bridge lights up when you
	// actually run winze-meld; a cell pulses when metabolism actually runs).
	mux.HandleFunc("/api/events.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		json.NewEncoder(w).Encode(map[string]any{"events": events.ReadRecent(200)})
	})

	url := displayURL(*addr)
	log.Printf("winze observatory → %s  (instances: %d)", url, len(resolveTargets(dirs)))
	if *open {
		openBrowser(url)
	}
	log.Fatal(http.ListenAndServe(*addr, mux))
}

// displayURL turns a listen address into an openable URL. A bare ":7777"
// becomes localhost; "0.0.0.0" is shown as localhost too (the reachable name
// for the local browser).
func displayURL(addr string) string {
	host, port := addr, ""
	if i := strings.LastIndex(addr, ":"); i >= 0 {
		host, port = addr[:i], addr[i+1:]
	}
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "localhost"
	}
	if port == "" {
		return "http://" + host
	}
	return "http://" + host + ":" + port
}

func openBrowser(url string) {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "darwin":
		cmd = "open"
	case "windows":
		cmd, args = "cmd", []string{"/c", "start"}
	default:
		cmd = "google-chrome" // house preference; never xdg-open
	}
	args = append(args, url)
	if err := exec.Command(cmd, args...).Start(); err != nil {
		fmt.Println("open manually:", url)
	}
}
