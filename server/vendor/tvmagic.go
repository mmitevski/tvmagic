package main

import (
	"net/http"
	"github.com/go-zoo/bone"
	"log"
	"time"
	"github.com/nytimes/gziphandler"
	"common"
	"services"
	"services/tv"
	"web"
	"services/session"
)

func LoggingHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t1 := time.Now()
		log.Printf("[%s] REQUEST BEGIN %q %v\n", r.Method, r.URL.String(), t1)
		defer func() {
			log.Printf("[%s] REQUEST END %q %v\n", r.Method, r.URL.String(), time.Now().Sub(t1))
		}()
		defer func() {
			str := recover()
			if str != nil {
				http.Error(w, "Internal server error.", http.StatusInternalServerError)
				log.Printf(`[%s] PANIC %q "%s"`, r.Method, r.URL.String(), str)
			}
		}()
		h.ServeHTTP(w, r)
	})
}

func main() {
	mux := bone.New()
	tv.Locations(mux)
	tv.TVs(mux)
	tv.Redirects(mux)
	services.Index(mux)
	session.Register(mux)
	http.Handle("/", gziphandler.GzipHandler(session.AuthHandler(LoggingHandler(mux))))
	web.Register()
	http.ListenAndServe(common.GetConfig().Server.Address, nil)
}
