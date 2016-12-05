package session

import (
	"log"
	"net/http"
	"strings"
	"fmt"
	"os/exec"
	"github.com/go-zoo/bone"
	"github.com/mmitevski/sessions"
	"github.com/mmitevski/sessions/memory"
	"github.com/mmitevski/tvmagic/server/common"
	"github.com/mmitevski/sessions/security"
)

var sm *sessions.Manager
var am security.AuthenticationManager

func init() {
	config := common.GetConfig()
	sm, _ = sessions.NewManager(memory.New(), config.Session.Cookie, config.Session.MaxLifeTime, config.Session.Secure)
	am = security.NewAuthenticationManager(pwauth)
}

func Session(w http.ResponseWriter, r *http.Request) sessions.Session {
	return sm.Start(w, r)
}

func GetAuthentication(r *http.Request) security.Authentication {
	if session := sm.Get(r); session != nil {
		auth := session.Get("auth")
		m, ok := auth.(security.Authentication)
		if ok {
			return m
		}
	}
	return nil
}

func IsAuthenticated(r *http.Request) bool {
	return GetAuthentication(r) != nil
}

func pwauth(user, secret string) security.Authentication {
	log.Printf("Authenticating %s...", user)
	cmd := exec.Command(common.GetConfig().Authentication.Command)
	if in, err := cmd.StdinPipe(); err != nil {
		log.Println(err)
		return nil
	} else {
		fmt.Fprintln(in, user)
		fmt.Fprintln(in, secret)
		if err := cmd.Run(); err != nil {
			log.Println(err)
			return nil
		}
		log.Printf("Successfully authenticated %s.", user)
		return security.NewAuthentication(user)
	}
}

func AuthHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, ".do") || r.URL.Path == "/" {
			auth := GetAuthentication(r)
			switch {
			case strings.HasPrefix(r.RequestURI, "/login"):
				if auth != nil {
					http.Redirect(w, r, "/", http.StatusFound)
					return
				}
			case strings.HasPrefix(r.RequestURI, "/logon"):
				break
			default:
				if auth == nil {
					http.Redirect(w, r, "/login.do", http.StatusFound)
					return
				}
			}
		}
		h.ServeHTTP(w, r)
	})
}

func Register(r *bone.Mux) {
	r.PostFunc("/logon.do", func(w http.ResponseWriter, r *http.Request) {
		user := r.FormValue("user")
		password := r.FormValue("password")
		log.Printf("Authenticate user %s from %s, referrer %s...", user, r.RemoteAddr, r.Referer())
		a, _ := am.Authenticate(user, password)
		if a != nil {
			sm.Start(w, r).Set("auth", a)
			log.Printf("New session for user %s from %s, referrer %s.", user, r.RemoteAddr, r.Referer())
			http.Redirect(w, r, "/", http.StatusFound)
		} else {
			http.Redirect(w, r, "/login.do?error", http.StatusFound)
		}
	})
	r.GetFunc("/logout.do", func(w http.ResponseWriter, r *http.Request) {
		sm.Destroy(w, r)
		http.Redirect(w, r, "/", http.StatusFound)
	})
}
