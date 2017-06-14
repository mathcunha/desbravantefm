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
	Title string
	Desc  string
	Itens []item
}

type item struct {
	Show  show
	Date  string
	Title string
}

func main() {
	http.HandleFunc("/show", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "POST":
			decoder := json.NewDecoder(r.Body)
			s := show{}
			if err := decoder.Decode(&s); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

		case "GET":
		}
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func saveItemContent(id string) error {
	resp, err := http.Get("http://" + contentHost + ".mp3")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return err
	}

	return ioutil.WriteFile(id+".mp3", body, 0666)
}

func (s *show) addShow() error {
	s, err := getSession()
	if err != nil {
		return err
	}
	defer closeSession(s)

	s.SetSafe(&mgo.Safe{FSync: true})

	return s.DB(database).C("show").Insert(show{Title: s.Title, Description: s.Desc})
}

func (i *item) addItem() error {
	s, err := getSession()
	if err != nil {
		return err
	}
	defer closeSession(s)

	s.SetSafe(&mgo.Safe{FSync: true})

	item := struct {
		Date  string
		Title string
	}{i.Date, i.Title}

	change := bson.M{"$push": bson.M{"itens": &item}}

	return s.DB(database).C("show").Update(bson.M{"_id": i.Show.title}, change)
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
