package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

import (
	"github.com/craftamap/go-shit/dto"
)

type Middleware func(http.HandlerFunc) http.HandlerFunc

func Logging() Middleware {
	return func(f http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			log.Println(r.Method, r.URL.Path, r.RemoteAddr)

			f(w, r)
		}
	}
}

func Chain(f http.HandlerFunc, middleswares ...Middleware) http.HandlerFunc {
	for _, m := range middleswares {
		f = m(f)
	}
	return f
}

var db *gorm.DB

var templates *template.Template

func main() {
	log.Print("Starting go-shit")
	log.Print("Connecting to db...")
	var err error
	db, err = gorm.Open("sqlite3", "datebase.db")
	if err != nil {
		panic("failed to connect database")
	}
	defer db.Close()
	log.Print("Connection successful! Now AutoMigrating db")
	db.AutoMigrate(&shit.Shit{})
	log.Print("Done!")

	var allFiles []string
	files, err := ioutil.ReadDir("./templates")
	if err != nil {
		fmt.Println(err)
	}
	for _, file := range files {
		filename := file.Name()
		if strings.HasSuffix(filename, ".html") {
			allFiles = append(allFiles, "./templates/"+filename)
		}
	}

	templates, err = template.ParseFiles(allFiles...) //parses all .tmpl files in the 'templates' folder

	log.Print("Setting up routes...")
	r := mux.NewRouter()
	log.Print("Setting up /shit router...")
	shitrouter := r.PathPrefix("/shit").Subrouter()

	shitrouter.HandleFunc("/", Chain(GetAllShits, Logging())).Methods("GET")
	shitrouter.HandleFunc("/", Chain(CreateShit, Logging())).Methods("POST")

	shitrouter.HandleFunc("/{id}", Chain(GetShit, Logging())).Methods("GET")
	shitrouter.HandleFunc("/{id}", Chain(DeleteShit, Logging())).Methods("DELETE")
	shitrouter.HandleFunc("/{id}", Chain(UpdateShit, Logging())).Methods("PUT")

	r.HandleFunc("/", Chain(Index, Logging())).Methods("GET")

	fs := http.FileServer(http.Dir("static/"))

	staticPrefix := r.PathPrefix("/static/")
	staticPrefix.Handler(http.StripPrefix("/static/", fs))

	log.Print("Shit is now ready to serve!")
	http.ListenAndServe(":8080", r)
}

func Index(w http.ResponseWriter, r *http.Request) {
	allShits := []shit.Shit{}
	db.Find(&allShits)

	data := map[string]interface{}{
		"shits": allShits,
	}

	templates.Lookup("index.html").Execute(w, data)
}

func GetAllShits(w http.ResponseWriter, r *http.Request) {
	var shits []shit.Shit
	db.Find(&shits)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(shits)
}

func GetShit(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var shit shit.Shit
	db.First(&shit, id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(shit)
}

func DeleteShit(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	db.Delete(&shit.Shit{}, id)
}

func CreateShit(w http.ResponseWriter, r *http.Request) {
	type AcceptedShit struct {
		Text string `json:"text"`
	}

	acShit := AcceptedShit{}
	err := json.NewDecoder(r.Body).Decode(&acShit)
	shit := shit.Shit{Text: acShit.Text}
	shit.Timestamp = time.Now()
	if err != nil {
		panic("failed!")
	}
	db.Create(&shit)
	log.Print(shit)
}

func UpdateShit(w http.ResponseWriter, r *http.Request) {
	type AcceptedShit struct {
		Text string `json:"text"`
	}

	vars := mux.Vars(r)
	id := vars["id"]
	acShit := AcceptedShit{}
	err := json.NewDecoder(r.Body).Decode(&acShit)
	if err != nil {
		log.Fatal(err)
		panic("failed!")
	}
	i, err := strconv.Atoi(id)
	if err != nil {
		panic("failed!")
	}

	var u_shit shit.Shit

	db.Model(&u_shit).Where(shit.Shit{ID: i}).Updates(shit.Shit{Text: acShit.Text})
}
