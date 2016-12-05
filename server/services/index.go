package services

import (
	"net/http"
	"github.com/go-zoo/bone"
	"io"
	"github.com/mmitevski/tvmagic/server/web"
	"github.com/mmitevski/tvmagic/server/common"
	"github.com/mmitevski/transactions/db"
)

type locationInfo struct {
	Name  string
	Count int64
}

func Index(r *bone.Mux) {
	r.GetFunc("/", func(w http.ResponseWriter, r *http.Request) {
		type data struct {
			Locations []*locationInfo
			IntroSubTitle *string
		}
		var d data
		d.IntroSubTitle = &(common.GetConfig().UI.IntroSubTitle)
		common.DB().Execute(func(tx db.Transaction) {
			tx.Query(`select a.name, count(t.id) from location a
			left outer join tv t on t.location = a.id
			group by a.name
			order by upper(a.name);`, func(r db.Result) {
				l := &locationInfo{}
				r.Scan(&l.Name, &l.Count)
				d.Locations = append(d.Locations, l)
			})
		})
		web.MainLayout(w, r, "", func(w io.Writer) {
			web.Layout("pages/index.html", w, r, d)
		})
	})
	r.GetFunc("/login.do", func(w http.ResponseWriter, r *http.Request) {
		var data struct {
		}
		web.MainLayout(w, r, "User Login", func(w io.Writer) {
			web.Layout("pages/login.html", w, r, data)
		})
	})
}