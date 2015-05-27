package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

type authHandler struct {
	next http.Handler
}

func (h *authHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if _, err := r.Cookie("auth"); err == http.ErrNoCookie {
		// not authenticated
		w.Header().Set("Location", "/login")
		w.WriteHeader(http.StatusTemporaryRedirect)
	} else if err != nil {
		// some other error
		panic(err.Error())
	} else {
		// success - call the next handler
		h.next.ServeHTTP(w, r)
	}
}

func MustAuth(handler http.Handler) http.Handler {
	return &authHandler{next: handler}
}

// loginHander handles the third-party login process.
// format: /auth/{action}/{provider}
// Our loginHandler is only a function and not an object that implements the
// http. Handler interface. This is because, unlike other handlers, we don't
// need it to store any state.

// TODO: might want to consider using dedicated packages such as Goweb, Pat,
// Routes, or mux. For extremely simple cases such as ours, the built-in
// capabilities will do.

func loginHandler(w http.ResponseWriter, r *http.Request) {
	//break the path into segments using strings.Split before pulling out the
	// values for action and provider. If the action value is known, we will run
	// the specific code; otherwise, we will write out an error message and
	// return an http.StatusNotFound status code (which in the language of HTTP
	// status code, is a 404 code).
	segs := strings.Split(r.URL.Path, "/")
	action := segs[2]
	provider := segs[3]
	switch action {
	case "login":
		log.Println("TODO handle for", provider)
	default:
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Auth action %s not supported", action)
	}
}
