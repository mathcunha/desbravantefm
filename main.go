package main

import (
	"encoding/json"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

var contentHost = "video.m.mais.uol.com.br"
var mongo_port = "localhost"
var database = "desbravante"

type show struct {
	Id    string `bson:"_id"`
	Title string
	Desc  string
	Itens []item
}

type item struct {
	Id    string
	Date  string
	Title string
}

func main() {
	http.HandleFunc("/show/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			decoder := json.NewDecoder(r.Body)
			s := show{}
			if err := decoder.Decode(&s); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			if len(s.Itens) > 0 {
				if err := s.addItem(); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				if err := s.Itens[0].saveContent(); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			} else {
				if err := s.addShow(); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
			w.WriteHeader(http.StatusCreated)

		case "GET":
			a_path := strings.Split(r.URL.Path, "/")

			if "" != a_path[2] { //by id
				s := show{Id: a_path[2]}
				if err := s.loadShow(); err != nil {
					http.Error(w, err.Error(), http.StatusNotFound)
					return
				}

				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusOK)

				if err := json.NewEncoder(w).Encode(s); err != nil {
					log.Println("SEVERE: %v error returning json response %v\n", err, s)
				}
			}
		}
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func (i *item) saveContent() error {
	resp, err := http.Get("http://" + contentHost + i.Id + ".mp3")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return err
	}

	return ioutil.WriteFile(i.Id+".mp3", body, 0666)
}

func (s *show) loadShow() error {
	session, err := getSession()
	if err != nil {
		return err
	}
	defer closeSession(session)

	return session.DB(database).C("show").Find(bson.M{"_id": s.Id}).One(s)
}

func (s *show) addShow() error {
	session, err := getSession()
	if err != nil {
		return err
	}
	defer closeSession(session)

	session.SetSafe(&mgo.Safe{FSync: true})

	return session.DB(database).C("show").Insert(show{Title: s.Title, Desc: s.Desc, Id: s.Id})
}

func (s *show) addItem() error {
	session, err := getSession()
	if err != nil {
		return err
	}
	defer closeSession(session)

	session.SetSafe(&mgo.Safe{FSync: true})

	change := bson.M{"$push": bson.M{"itens": (&s.Itens[0])}}

	return session.DB(database).C("show").Update(bson.M{"_id": s.Id}, change)
}

func init() {
	if host := os.Getenv("CONTENT_HOST"); host != "" {
		contentHost = host
		log.Printf("INFO: mp3 content host is now %v \n", host)
	}
	if m := os.Getenv("MONGO_PORT"); m != "" {
		mongo_port = strings.Replace(m, "tcp", "mongodb", 1)
		log.Printf("INFO: Mongo broker at %v \n", m)
	}
}
func getSession() (*mgo.Session, error) {
	session, err := mgo.Dial(mongo_port)
	return session, err
}

func closeSession(s *mgo.Session) {
	s.Close()
}
