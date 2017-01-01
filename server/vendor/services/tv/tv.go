package tv

import (
	"github.com/mmitevski/transactions/db"
	"errors"
	"net/http"
	"strconv"
	"github.com/go-zoo/bone"
	"io"
	"fmt"
	"strings"
	"log"
	"common"
	"web"
	"formatted"
)

type TV struct {
	Id       int64       `json:"id"`
	Name     string      `json:"name"`
	Location Location    `json:"location"`
	URL      string      `json:"url"`
	On       string      `json:"on"`
	Off      string      `json:"off"`
}

func (tv *TV) Path() string {
	return fmt.Sprintf("/%s/TV/%s", tv.Location.Name, tv.Name)
}

const selectTVSql string = `select a.id, a.name, a.url, a.location, l.name, a.time_on, a.time_off from tv a
                left outer join location l on l.id = a.location
                where true`

func scan(t *TV, r db.Result) {
	r.Scan(&t.Id, &t.Name, &t.URL, &t.Location.Id, &t.Location.Name, &t.On, &t.Off)
}

func LoadTVs(tx db.Transaction, tvs *[]*TV, location int64) {
	tx.Query(selectTVSql + " and a.location = $1 order by upper(a.name)", func(r db.Result) {
		tv := &TV{}
		scan(tv, r)
		*tvs = append(*tvs, tv)
	}, location)
}

func LoadTV(tx db.Transaction, tv *TV, id interface{}) {
	tx.Query(selectTVSql + " and a.id = $1", func(r db.Result) {
		scan(tv, r)
	}, id)
}

func GetTVByLocationAndName(location, name string) *TV {
	var tv *TV
	common.DB().Execute(func(tx db.Transaction) {
		var t TV
		tx.Query(selectTVSql + " and l.name = $1 and a.name = $2", func(r db.Result) {
			scan(&t, r)
		}, location, name)
		tv = &t
	})
	return tv
}

func PersistTV(tx db.Transaction, tv *TV) {
	rows := tx.Execute(
		"update tv set name = $2, url = $3, time_on = $4, time_off = $5 where id = $1",
		tv.Id, tv.Name, tv.URL, tv.On, tv.Off)
	if rows == 0 {
		tx.Query("insert into tv(location, name, url, time_on, time_off) values ($1, $2, $3, $4, $5) returning id", func(r db.Result) {
			r.Scan(&tv.Id)
		}, tv.Location.Id, tv.Name, tv.URL, tv.On, tv.Off)
	}
	if tv.Id != 0 {
		LoadTV(tx, tv, tv.Id)
	}
}

func deleteTV(tx db.Transaction, id interface{}) bool {
	defer func() {
		err := recover()
		if err != nil {
			panic(errors.New("Error deleting TV."))
		}
	}()
	rows := tx.Execute("delete from tv where id = $1", id)
	return rows > 0
}

func TVs(r *bone.Mux) {
	// MVC-specific endpoints
	r.GetFunc("/tvs/list.do", func(w http.ResponseWriter, r *http.Request) {
		var data struct {
			TVs       []*TV
			Locations []*Location
			Location  Location
		}
		v := r.URL.Query().Get("location")
		location, err := strconv.ParseInt(v, 0, 64)
		if err != nil {
			var l *Location
			common.DB().Execute(func(tx db.Transaction) {
				l = GetDefaultLocation(tx)
			})
			if l != nil {
				http.Redirect(w, r,
					fmt.Sprintf("/tvs/list.do?location=%d", l.Id),
					http.StatusFound)
			} else {
				http.Error(w,
					"No available office locations. First register at least one.",
					http.StatusBadRequest)
			}
		} else {
			common.DB().Execute(func(tx db.Transaction) {
				LoadTVs(tx, &data.TVs, location)
				LoadLocations(tx, &data.Locations)
				LoadLocation(tx, &data.Location, location)
				web.MainLayout(w, r, fmt.Sprintf(`TVs in office "%s"`, data.Location.Name), func(w io.Writer) {
					web.Layout("pages/tvs.html", w, r, data)
				})
			})
		}
	})
	type TVProvider func(tv *TV)
	edit := func(w http.ResponseWriter, r *http.Request, provider TVProvider, err error) {
		var data struct {
			TV  TV
			Err error
		}
		data.Err = err
		web.MainLayout(w, r, "Modify TV", func(w io.Writer) {
			provider(&data.TV)
			web.Layout("pages/tv.html", w, r, data)
		})
	}
	r.GetFunc("/tvs/edit.do", func(w http.ResponseWriter, r *http.Request) {
		v := r.URL.Query().Get("id")
		id, err := strconv.ParseInt(v, 0, 64)
		if err != nil {
			http.Error(w, "Invalid TV.", http.StatusBadRequest)
		} else {
			edit(w, r, func(tv *TV) {
				common.DB().Execute(func(tx db.Transaction) {
					LoadTV(tx, tv, id)
				})
			}, nil)
		}
	})
	r.GetFunc("/tvs/create.do", func(w http.ResponseWriter, r *http.Request) {
		v := r.URL.Query().Get("location")
		location, err := strconv.ParseInt(v, 0, 64)
		if err != nil {
			http.Error(w, "Invalid office location.", http.StatusBadRequest)
		} else {
			edit(w, r, func(tv *TV) {
				tv.Location.Id = location
			}, nil)
		}
	})
	http.HandleFunc("/tvs/persist.do", func(w http.ResponseWriter, r *http.Request) {
		location, err := ParseInt64(r.FormValue("location"))
		if err != nil {
			http.Error(w, "Invalid Office Location.", http.StatusBadRequest)
			return
		}
		defer func() {
			http.Redirect(w, r, fmt.Sprintf("/tvs/list.do?location=%d", location), http.StatusFound)
		}()
		if r.FormValue("persist") == "persist" {
			id, errId := ParseInt64(r.FormValue("id"))
			name := strings.TrimSpace(r.FormValue("name"))
			url := strings.TrimSpace(r.FormValue("url"))
			on := strings.TrimSpace(r.FormValue("on"))
			off := strings.TrimSpace(r.FormValue("off"))
			defer func() {
				err := recover()
				if err != nil {
					edit(w, r, func(tv *TV) {
						if errId == nil {
							tv.Id = id
						}
						tv.Name = name
						tv.Location.Id = location
					}, errors.New(fmt.Sprintf("%s", err)))
					log.Printf("Error: %s", err)
					return
				}
			}()
			if len(name) == 0 {
				panic(errors.New("TV name is required."))
			}
			common.DB().Execute(func(tx db.Transaction) {
				var tv TV
				if errId == nil {
					LoadTV(tx, &tv, id)
				}
				tv.Name = name
				tv.URL = url
				tv.On = on
				tv.Off = off
				LoadLocation(tx, &tv.Location, location)
				PersistTV(tx, &tv)
			})
		}
	})
	r.GetFunc("/tvs/delete.do", func(w http.ResponseWriter, r *http.Request) {
		params := r.URL.Query()
		location, err := ParseInt64(params.Get("location"))
		if err != nil {
			http.Error(w, "Invalid Office Location.", http.StatusBadRequest)
			return
		}
		id, err := ParseInt64(params.Get("id"))
		if err != nil {
			http.Error(w, "Invalid TV.", http.StatusBadRequest)
			return
		}
		defer func() {
			err := recover()
			http.Error(w, fmt.Sprint(err), http.StatusBadRequest)
		}()
		common.DB().Execute(func(tx db.Transaction) {
			deleteTV(tx, id)
		})
		http.Redirect(w, r, "/tvs/list.do?type=" + strconv.FormatInt(location, 10), http.StatusFound)
	})
}

func Redirects(r *bone.Mux) {
	r.GetFunc("/:location/TV/:tv", func(w http.ResponseWriter, r *http.Request) {
		locationName := bone.GetValue(r, "location")
		tvName := bone.GetValue(r, "tv")
		log.Printf("location: %s, tc: %s", locationName, tvName)
		v := GetTVByLocationAndName(locationName, tvName)
		if v == nil {
			http.Error(w, "Invalid office location or TV.", http.StatusNotFound)
			return
		}
		url := strings.TrimSpace(v.URL)
		if len(url) > 0 {
			http.Redirect(w, r, url, http.StatusFound)
		} else {
			http.Error(w, "There is no url, configured for the requested TV.", http.StatusOK)
		}
	})
	r.GetFunc("/:location/TV/:tv/config", func(w http.ResponseWriter, r *http.Request) {
		locationName := bone.GetValue(r, "location")
		tvName := bone.GetValue(r, "tv")
		log.Printf("location: %s, tc: %s", locationName, tvName)
		v := GetTVByLocationAndName(locationName, tvName)
		if v == nil {
			http.Error(w, "Invalid office location or TV.", http.StatusNotFound)
			return
		}
		var data struct {
			OnTime  string
			OffTime string
			URL     string
		}
		data.OnTime = v.On
		data.OffTime = v.Off
		data.URL = v.URL
		formatted.ServeJson(w, data)
	})
}