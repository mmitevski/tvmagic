package tv

import (
	"errors"
	"github.com/mmitevski/transactions/db"
	"io"
	"net/http"
	"strconv"
	"github.com/go-zoo/bone"
	"fmt"
	"log"
	"strings"
	"web"
	"common"
)

type Location struct {
	Id   int64       `json:"id"`
	Name string      `json:"name"`
}

const selectLocationSql string = `select a.id, a.name from location a where true`

func LoadLocations(tx db.Transaction, locations *[]*Location) {
	tx.Query(selectLocationSql + " order by upper(a.name)", func(r db.Result) {
		l := &Location{}
		r.Scan(&l.Id, &l.Name)
		*locations = append(*locations, l)
	})
}

func GetDefaultLocation(tx db.Transaction) *Location {
	var location *Location
	tx.Query(selectLocationSql + " order by upper(a.name) limit 1", func(r db.Result) {
		l := &Location{}
		r.Scan(&l.Id, &l.Name)
		location = l
	})
	return location
}

func LoadLocation(tx db.Transaction, location *Location, locationId interface{}) {
	tx.Query(selectLocationSql + " and a.id = $1", func(r db.Result) {
		r.Scan(&location.Id, &location.Name)
	}, locationId)
}

func PersistLocation(tx db.Transaction, location *Location) {
	rows := tx.Execute("update location set name = $2 where id = $1", location.Id, location.Name)
	if rows == 0 {
		tx.Query("insert into location(name) values ($1) returning id", func(r db.Result) {
			r.Scan(&location.Id)
		}, location.Name)
	}
	if location.Id != 0 {
		LoadLocation(tx, location, location.Id)
	}
	rows++
}

func deleteLocation(tx db.Transaction, id interface{}) bool {
	defer func() {
		err := recover()
		if err != nil {
			panic(errors.New("Error deleting Location. Are you sure there are no registered TVs in it?"))
		}
	}()
	rows := tx.Execute("delete from location where id = $1", id)
	return rows > 0
}

func ParseInt64(str string) (int64, error) {
	v, err := strconv.ParseInt(str, 0, 64)
	if err != nil {
		return 0, err
	} else {
		return v, nil
	}
}

func Locations(b *bone.Mux) {
	// MVC-specific endpoints
	b.GetFunc("/locations/list.do", func(w http.ResponseWriter, r *http.Request) {
		var data struct {
			Items []*Location
		}
		web.MainLayout(w, r, "Office locations", func(w io.Writer) {
			common.DB().Execute(func(tx db.Transaction) {
				LoadLocations(tx, &data.Items)
			})
			web.Layout("pages/locations.html", w, r, data)
		})
	})
	type LocationProvider func(location *Location)
	edit := func(w http.ResponseWriter, r *http.Request, provider LocationProvider, err error) {
		var data struct {
			Location Location
			Err      error
		}
		data.Err = err
		web.MainLayout(w, r, "Office location", func(w io.Writer) {
			provider(&data.Location)
			web.Layout("pages/location.html", w, r, data)
		})
	}
	b.GetFunc("/locations/edit.do", func(w http.ResponseWriter, r *http.Request) {
		v := r.URL.Query().Get("id")
		locationId, err := strconv.ParseInt(v, 0, 64)
		if err != nil {
			http.Error(w, "Invalid location.", http.StatusBadRequest)
		} else {
			edit(w, r, func(location *Location) {
				common.DB().Execute(func(tx db.Transaction) {
					LoadLocation(tx, location, locationId)
				})
			}, nil)
		}
	})
	b.GetFunc("/locations/create.do", func(w http.ResponseWriter, r *http.Request) {
		edit(w, r, func(location *Location) {
		}, nil)
	})
	http.HandleFunc("/locations/persist.do", func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			http.Redirect(w, r, "/locations/list.do", http.StatusFound)
		}()
		if r.FormValue("persist") == "persist" {
			id, errId := ParseInt64(r.FormValue("id"))
			name := strings.TrimSpace(r.FormValue("name"))
			defer func() {
				err := recover()
				if err != nil {
					edit(w, r, func(location *Location) {
						if errId == nil {
							location.Id = id
						}
						location.Name = name
					}, errors.New(fmt.Sprintf("%s", err)))
					log.Printf("Error: %s", err)
					return
				}
			}()
			if len(name) == 0 {
				panic(errors.New("Location is required."))
			}
			common.DB().Execute(func(tx db.Transaction) {
				var location Location
				if errId == nil {
					LoadLocation(tx, &location, id)
				}
				location.Name = name
				PersistLocation(tx, &location)
			})
		}
	})
	b.GetFunc("/locations/delete.do", func(w http.ResponseWriter, r *http.Request) {
		params := r.URL.Query()
		id, err := ParseInt64(params.Get("id"))
		if err != nil {
			http.Error(w, "Invalid location.", http.StatusBadRequest)
			return
		}
		defer func() {
			err := recover()
			http.Error(w, fmt.Sprint(err), http.StatusBadRequest)
		}()
		common.DB().Execute(func(tx db.Transaction) {
			deleteLocation(tx, id)
		})
		http.Redirect(w, r, "/locations/list.do", http.StatusFound)
	})
}
