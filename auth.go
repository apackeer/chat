package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/stretchr/gomniauth"
	"github.com/stretchr/objx"
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
		// use the gomniauth.Provider function to get the provider object that
		// matches the object specified in the URL (such as google or github)
		provider, err := gomniauth.Provider(provider)
		if err != nil {
			log.Fatalln("Error when trying to get provider", provider, "-", err)
		}

		// use the GetBeginAuthURL method to get the location where we must send
		// users in order to start the authentication process.
		// The GetBeginAuthURL(nil, nil) arguments are for the state and options
		// respectively, which we are not going to use for our chat application.
		// The first argument is a state map of data that is encoded, and signed
		// and sent to the authentication provider. The provider doesn't do
		// anything with the state, it just sends it back to our callback endpoint.
		// This is useful if, for example, we want to redirect the user back to
		// the original page they were trying to access before the authentication
		// process intervened. For our purpose, we have only the /chat endpoint,
		// so we don't need to worry about sending any state.

		// The second argument is a map of additional options that will be sent to
		// the authentication provider, which somehow modifies the behavior of the
		// authentication process. For example, you can specify your own scope
		// parameter, which allows you to make a request for permission to access
		// additional information from the provider. For more information about
		// the available options, search for OAuth2 on the Internet or read the
		// documentation for each provider, as these values differ from service
		// to service.
		loginUrl, err := provider.GetBeginAuthURL(nil, nil)
		if err != nil {
			log.Fatalln("Error when trying to GetBeginAuthURL for", provider, "-", err)
		}

		// If our code gets no error from the GetBeginAuthURL call, we simply
		// redirect the user's browser to the returned URL.
		w.Header().Set("Location", loginUrl)
		w.WriteHeader(http.StatusTemporaryRedirect)

		// When the authentication provider redirects the users back after they have
		// granted permission, the URL specifies that it is a callback action
	case "callback":
		provider, err := gomniauth.Provider(provider)
		if err != nil {
			log.Fatalln("Error when trying to get provider", provider, "-", err)
		}

		creds, err := provider.CompleteAuth(objx.MustFromURLQuery(r.URL.RawQuery))
		if err != nil {
			log.Fatalln("Error when trying to complete auth for", provider, "-", err)
		}

		user, err := provider.GetUser(creds)
		if err != nil {
			log.Fatalln("Error when trying to get user from", provider, "-", err)
		}

		// TODO: Storing non-signed cookies like this is fine for incidental
		// information such as a user's name, however, you should avoid storing
		// any sensitive information using non-signed cookies, as it's easy for
		// people to access and change the data.
		authCookieValue := objx.New(map[string]interface{}{
			"name": user.Name(),
		}).MustBase64()

		http.SetCookie(w, &http.Cookie{
			Name:  "auth",
			Value: authCookieValue,
			Path:  "/"})

		w.Header()["Location"] = []string{"/chat"}
		w.WriteHeader(http.StatusTemporaryRedirect)

	default:
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Auth action %s not supported", action)
	}
}
