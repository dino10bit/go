/*
Copyright 2015 Tamás Gulácsi

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package coord

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"gopkg.in/inconshreveable/log15.v2"
)

var (
	Log = log15.New("lib", "coord")

	DefaultTitle   = "Cím koordináták pontosítása"
	DefaultAddress = "Budapest"
)

type Interactive struct {
	Set            func(id string, loc Location) error
	Title          string
	MapCenter      Location
	Location       Location
	DefaultAddress string
	BaseURL        string
	NoDirect       bool

	inProgress   map[string]struct{} // location set in progress
	inProgressMu sync.Mutex
}
type staticParams struct {
	Address, Title             string
	MapCenterLat, MapCenterLng string
	LocLat, LocLng             string
	DefaultAddress             string
	CallbackPath               string
}

func (in *Interactive) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := genID()
	in.inProgressMu.Lock()
	if in.inProgress == nil {
		in.inProgress = make(map[string]struct{}, 8)
	}
	in.inProgress[id] = struct{}{}
	in.inProgressMu.Unlock()

	if strings.HasSuffix(r.URL.Path, "/set") {
		in.serveSet(w, r)
		return
	}
	if in.DefaultAddress == "" {
		in.DefaultAddress = DefaultAddress
	}
	if in.Title == "" {
		in.Title = DefaultTitle
	}
	vals := r.URL.Query()
	sp := staticParams{
		Address:      vals.Get("address"),
		Title:        in.Title,
		MapCenterLat: fmt.Sprintf("%+f", in.MapCenter.Lat),
		MapCenterLng: fmt.Sprintf("%+f", in.MapCenter.Lng),
		LocLat:       fmt.Sprintf("%+f", in.Location.Lat),
		LocLng:       fmt.Sprintf("%+f", in.Location.Lng),
		CallbackPath: in.BaseURL + "/set?id=" + id,
	}
	if err := tmpl.Execute(w, sp); err != nil {
		Log.Error("template with %#v: %v", sp, err)
	}
}
func (in *Interactive) serveSet(w http.ResponseWriter, r *http.Request) {
	vals := r.URL.Query()
	id := vals.Get("id")
	in.inProgressMu.Lock()
	_, ok := in.inProgress[id]
	in.inProgressMu.Unlock()
	if id == "" || !ok {
		http.Error(w, "id is required", http.StatusBadRequest)
		return
	}

	lat, lng, err := parseLatLng(vals.Get("lat"), vals.Get("lng"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	in.inProgressMu.Lock()
	delete(in.inProgress, id)
	in.inProgressMu.Unlock()

	if in.Set == nil {
		return
	}
	if err := in.Set(mp.ID, mp.Location); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

var tmpl *template.Template

func init() {
	Log.SetHandler(log15.DiscardHandler())

	tmpl = template.Must(template.New("gmapsHTML").Parse(gmapsHTML))
}

func genID() string {
	buf := make([]byte, 32)
	n, _ := io.ReadAtLeast(rand.Reader, buf, len(buf)/2)
	if n == 0 {
		n = 1
	}
	return base64.URLEncoding.EncodeToString(buf[:n])
}
func parseLatLng(latS, lngS string) (float64, float64, error) {
	lat, err := strconv.ParseFloat(latS, 64)
	if err != nil {
		return lat, 0, err
	}
	lng, err := strconv.ParseFloat(lngS, 64)
	return lat, lng, err
}