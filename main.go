//go:generate pkger -include /templates -include /static

package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/markbates/pkger"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
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

func Auth() Middleware {
	return func(f http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			username, password, authOK := r.BasicAuth()
			if authOK == false {
				http.Error(w, "Not authorized", 401)
				return
			}
			var user dto.User
			db.Where("username = ?", username).First(&user)
			if !user.CheckPasswd(password) {
				http.Error(w, "Not authorized", 401)
				return
			}
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
	db.AutoMigrate(&dto.Shit{}, &dto.User{})
	log.Print("Done!")
	var count int
	db.Find(&dto.User{}).Count(&count)
	if count == 0 {
		log.Print("Found no user! Creating init user admin:admin!")
		user := &dto.User{}
		user.Username = "admin"
		user.SetPasswd("admin")
		db.Save(&user)
	}
	templates = template.New("")

	dir := pkger.Include("/templates")

	err = pkger.Walk(dir, func(path string, info os.FileInfo, _ error) error {
	    if info.IsDir() || !strings.HasSuffix(path, ".html") {
		return nil
	    }

	    f, err := pkger.Open(path)
	    if err != nil {
		fmt.Print(err)
	    }
	    sl, err := ioutil.ReadAll(f)
	    if err != nil {
		fmt.Print(err)
	    }

	    templates.Parse(string(sl))
	    return nil
	})
	fmt.Println(templates)

	log.Print("Setting up routes...")
	r := mux.NewRouter()
	log.Print("Setting up /shit router...")
	shitrouter := r.PathPrefix("/shit").Subrouter()

	shitrouter.HandleFunc("/", Chain(GetAllShits, Logging())).Methods("GET")
	shitrouter.HandleFunc("/{id}", Chain(GetShit, Logging())).Methods("GET")

	shitrouter.HandleFunc("/", Chain(CreateShit, Logging())).Methods("POST")
	shitrouter.HandleFunc("/{id}", Chain(DeleteShit, Logging())).Methods("DELETE")
	shitrouter.HandleFunc("/{id}", Chain(UpdateShit, Logging())).Methods("PUT")

	r.HandleFunc("/", Chain(Index, Logging())).Methods("GET")
	
	fsdir, _ := pkger.Open("/static")

	fs := http.FileServer(fsdir)
	

	staticPrefix := r.PathPrefix("/static/")
	staticPrefix.Handler(http.StripPrefix("/static/", fs))

	log.Print("Shit is now ready to serve!")
	http.ListenAndServe(":8080", r)
}

func Index(w http.ResponseWriter, r *http.Request) {
	allShits := []dto.Shit{}
	db.Find(&allShits)

	data := map[string]interface{}{
		"shits": allShits,
	}

	templates.ExecuteTemplate(w, "index", data)
}

func GetAllShits(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var shits []dto.Shit
	db.Find(&shits)

	response := dto.Response{
		Status: dto.Success,
		Data:   shits,
	}

	json.NewEncoder(w).Encode(response)
}

func GetShit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	vars := mux.Vars(r)
	id := vars["id"]

	var gshit dto.Shit
	db.First(&gshit, id)
	if gshit.ID == 0 {
		json.NewEncoder(w).Encode(dto.Response{
			Status: dto.Fail,
			Data:   "Id not found",
		})
		return
	}

	json.NewEncoder(w).Encode(dto.Response{Data: gshit})
}

func DeleteShit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	id := vars["id"]

	dshit := dto.Shit{}

	db.Where("ID = ?", id).First(&dshit)
	if dshit.ID == 0 {
		json.NewEncoder(w).Encode(dto.Response{
			Status: dto.Fail,
			Data:   "Id not found",
		})
		return
	}

	db.Delete(&dshit)
	json.NewEncoder(w).Encode(dto.Response{
		Status: dto.Success,
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
		json.NewEncoder(w).Encode(dto.Response{
			Status: dto.Fail,
			Data:   "Invalid input data",
		})
		return
	}
	cshit := dto.Shit{Text: acShit.Text}
	cshit.Timestamp = time.Now()
	db.Create(&cshit)

	json.NewEncoder(w).Encode(dto.Response{
		Status: dto.Success,
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
		json.NewEncoder(w).Encode(dto.Response{
			Status: dto.Fail,
			Data:   "Invalid input data",
		})
	}
	i, err := strconv.Atoi(id)
	if err != nil {
		json.NewEncoder(w).Encode(dto.Response{
			Status: dto.Fail,
			Data:   "Invalid id",
		})
	}

	var u_shit dto.Shit

	db.Model(&u_shit).Where(dto.Shit{ID: i}).Updates(dto.Shit{Text: acShit.Text})
	json.NewEncoder(w).Encode(dto.Response{
		Status: dto.Success,
		Data:   &u_shit,
	})
}
