package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"text/template"

	"github.com/apackeer/trace"
)

// templ represents a single template
// We need to make sure that the template is compiled once. The sync.Once
// type guarantees that the function we pass as an argument will only be executed
// once, regarless of how many goroutines are calling ServerHTTP.

type templateHandler struct {
	once     sync.Once
	filename string
	templ    *template.Template
}

// ServeHTTP handles the HTTP request
func (t *templateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	t.once.Do(func() {
		t.templ = template.Must(template.ParseFiles(filepath.Join("templates", t.filename)))
	})
	// This tells the template to render itself using data that can be extracted
	// from http.Request, which happens to include the host address that we need.
	t.templ.Execute(w, r)
}

func main() {
	var addr = flag.String("addr", ":8080", "The addr of the application.")
	flag.Parse() // parse the flags

	// Create a new room instance.
	r := newRoom()
	r.tracer = trace.New(os.Stdout)

	http.Handle("/assets/", http.StripPrefix("/assets", http.FileServer(http.Dir("./assets"))))

	// Give the Hanlde function an templateHander object that has the ServeHTTP
	// function defined as per the http.Handler interface which specifies only
	// the ServeHTTP method need to be present in order for a type (class) to be
	// used to serve HTTP requests by net/http
	http.Handle("/chat", MustAuth(&templateHandler{filename: "chat.html"}))

	http.Handle("/login", &templateHandler{filename: "login.html"})

	// r (Room instance) has ServeHTTP function, which creates a client and then
	// passes it to the join channel of the room.
	http.Handle("/room", r)

	// Goroutine watches three channels inside r (join, leave and forward)
	go r.run()

	// start the web server
	log.Println("Starting web server on", *addr)
	if err := http.ListenAndServe(*addr, nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}
