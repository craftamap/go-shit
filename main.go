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
	w.Header().Set("Content-Type", "application/json")
	var shits []shit.Shit
	db.Find(&shits)

	response := shit.Response{
		Status: shit.Success,
		Data:   shits,
	}

	json.NewEncoder(w).Encode(response)
}

func GetShit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	id := vars["id"]

	var gshit shit.Shit
	db.First(&gshit, id)
	if gshit.ID == 0 {
		json.NewEncoder(w).Encode(shit.Response{
			Status: shit.Fail,
			Data:   "Id not found",
		})
		return
	}

	json.NewEncoder(w).Encode(shit.Response{Data: gshit})
}

func DeleteShit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	id := vars["id"]

	dshit := shit.Shit{}

	db.Where("ID = ?", id).First(&dshit)
	if dshit.ID == 0 {
		json.NewEncoder(w).Encode(shit.Response{
			Status: shit.Fail,
			Data:   "Id not found",
		})
		return
	}

	db.Delete(&dshit)
	json.NewEncoder(w).Encode(shit.Response{
		Status: shit.Success,
		Data:   nil,
	})
}

func CreateShit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	type AcceptedShit struct {
		Text string `json:"text"`
	}

	acShit := AcceptedShit{}
	err := json.NewDecoder(r.Body).Decode(&acShit)
	if err != nil {
		json.NewEncoder(w).Encode(shit.Response{
			Status: shit.Fail,
			Data:   "Invalid input data",
		})
		return
	}
	cshit := shit.Shit{Text: acShit.Text}
	cshit.Timestamp = time.Now()
	db.Create(&cshit)

	json.NewEncoder(w).Encode(shit.Response{
		Status: shit.Success,
		Data:   &cshit,
	})
}

func UpdateShit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	type AcceptedShit struct {
		Text string `json:"text"`
	}

	vars := mux.Vars(r)
	id := vars["id"]
	acShit := AcceptedShit{}
	err := json.NewDecoder(r.Body).Decode(&acShit)
	if err != nil {
		log.Fatal(err)
		json.NewEncoder(w).Encode(shit.Response{
			Status: shit.Fail,
			Data:   "Invalid input data",
		})
	}
	i, err := strconv.Atoi(id)
	if err != nil {
		json.NewEncoder(w).Encode(shit.Response{
			Status: shit.Fail,
			Data:   "Invalid id",
		})
	}

	var u_shit shit.Shit

	db.Model(&u_shit).Where(shit.Shit{ID: i}).Updates(shit.Shit{Text: acShit.Text})
	json.NewEncoder(w).Encode(shit.Response{
		Status: shit.Success,
		Data:   &u_shit,
	})
}
