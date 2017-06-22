package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"text/template"
	"time"
)

const contentHost = "www.bandnewsfm.com.br/colunista/"

type item struct {
	ID     string
	Date   string
	Title  string
	Author string
}

type rss struct {
	Title string
	Date  string
	Items []item
	Image string
	Desc  string
	URL   string
	Host  string
}

type token struct {
	beginItems *regexp.Regexp
	endItems   *regexp.Regexp
	beginTitle string
	endTitle   string
}

var t1 = token{regexp.MustCompile(`<div class="vc_tta-container"`), regexp.MustCompile(`</div></div></div></div></div></div></div></div>`), `<span class="vc_tta-title-text">`, `</span>`}

var t2 = token{regexp.MustCompile(`<div class="vc_row wpb_row td-pb-row">`), regexp.MustCompile(`<footer>`), `<p>`, `</p>`}

var rssBody = template.Must(template.New("rssBody").Parse(`<?xml version="1.0" encoding="ISO-8859-1"?>
<rss version="2.0" xmlns:itunes="http://www.itunes.com/DTDs/Podcast-1.0.dtd" xmlns:media="http://search.yahoo.com/mrss/">
<channel>
<title>{{.Title}}</title>
<link>{{.URL}}</link>
<description>{{.Desc}}</description>
<itunes:subtitle>{{.Title}}</itunes:subtitle>
<language>pt-br</language>
<copyright>{{.Host}}</copyright>
<pubDate>{{.Date}}</pubDate>
<itunes:image href="{{.Image}}"/>
<itunes:category text="Information" />
<itunes:category text="News" />
<itunes:category text="International">
<itunes:category text="Brazilian" />
</itunes:category>
<itunes:keywords>{{.Title}}</itunes:keywords>
{{range .Items}}
<item>
<title>
<![CDATA[
{{.Title}}
]]>
</title>
<description/>
<itunes:subtitle/>
<pubDate>{{.Date}}</pubDate>
<enclosure url="http://video.m.mais.uol.com.br/{{.ID}}.mp3" type="audio/mpeg"/>
<itunes:duration>01:00</itunes:duration>
<itunes:summary>{{.Title}}</itunes:summary>
<itunes:author>{{.Author}}</itunes:author>
<itunes:keywords/>
</item>
{{end}}
</channel>
</rss>`))

func main() {
	http.HandleFunc("/show/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			a_path := strings.Split(r.URL.Path, "/")

			if "" != a_path[2] { //by id

				data := rss{}
				col := a_path[2]
				if "karnal" == a_path[2] {
					col = "careca-de-saber-com-leandro-karnal"
				}
				err := data.load(col)

				if err != nil {
					http.Error(w, err.Error(), http.StatusNotFound)
					return
				}

				w.Header().Set("Content-Type", "application/xml; charset=utf-8")
				w.WriteHeader(http.StatusOK)
				/*
					if err := json.NewEncoder(w).Encode(loadItens(strBody[begin:end])); err != nil {
						log.Println("SEVERE: %v error returning json response \n", err)
					}*/
				err = rssBody.Execute(w, data)
				if err != nil {
					http.Error(w, err.Error(), http.StatusNotFound)
					return
				}
			}
		}

	})
	var port = os.Getenv("PORT")
	// Set a default port if there is nothing in the environment
	if port == "" {
		port = "8080"
		log.Println("INFO: No PORT environment variable detected, defaulting to " + port)
	}
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func (i item) String() string {
	return fmt.Sprintf("{\"title\":%q, \"date\":%q, \"id\":%q}", i.Title, i.Date, i.ID)
}

func loadTitle(body []byte) string {
	return loadMetaData(`<meta property="og:title" content="(?P<title>.+)" />`, body)
}

func loadImage(body []byte) string {
	return loadMetaData(`<meta property="og:image" content="(?P<title>.+)" />`, body)
}

func loadDesc(column string, body []byte) string {
	log.Println("loading description")
	if resp, err := http.Get("http://www.bandnewsfm.com.br/colunistas/"); err == nil {
		defer resp.Body.Close()

		if body, err := ioutil.ReadAll(resp.Body); err == nil {

			validDesc := regexp.MustCompile(`<p class="listaIntroColunista"><a href="http://www.bandnewsfm.com.br/colunista/` + column + `/">(?P<desc>.+)</a></p>`)
			descriptions := validDesc.FindAllStringSubmatch(string(body), -1)
			if len(descriptions) == 1 {
				return descriptions[0][1]
			}

		}
	}

	return loadMetaData(`<meta property="og:description" content="(?P<title>.+)" />`, body)
}

func loadURL(body []byte) string {
	return loadMetaData(`<meta property="og:url" content="(?P<title>.+)" />`, body)
}

func loadHost(u string) string {
	url, err := url.Parse(u)
	if err != nil {
		log.Println("erro parsing URL %v - %v", u, err)
		return ""
	}
	return url.Host
}

func loadMetaData(pattern string, body []byte) string {
	valid := regexp.MustCompile(pattern)
	data := valid.FindSubmatch(body)
	if len(data) == 2 {
		return string(data[1])
	}
	return ""
}

func loadItems(t token, body, author string) []item {
	titleBegin := t.beginTitle
	titleEnd := t.endTitle
	validDate := regexp.MustCompile(`[0-9]+\/[0-9]+\/[0-9]+`)
	validMediaID := regexp.MustCompile(`mediaId=(?P<mediaId>[0-9]+)"`)
	itens := []item{}
	for {
		begin := strings.Index(body[0:len(body)], titleBegin)
		end := strings.Index(body[0:len(body)], titleEnd)
		if begin == -1 || end == -1 {
			break
		}
		title := body[begin+len(titleBegin) : end]
		date := validDate.Find([]byte(title))
		if date == nil {
			log.Println("Date not found - " + title)
		} else {
			mediaID := validMediaID.FindStringSubmatch(body[end:])
			if mediaID == nil || len(mediaID) != 2 {
				log.Println("mediaID not found - " + title)
			} else {
				itens = append(itens, item{Date: string(date), Title: title[len(string(date)):], ID: mediaID[1], Author: author})
			}
		}
		body = body[end+len(titleEnd) : len(body)]
	}
	return itens
}

func getIndexes(t token, body []byte) (begin, end int) {
	indexes := t.beginItems.FindIndex(body)
	if len(indexes) != 2 {
		return -1, -1
	}
	begin = indexes[1]
	end = t.endItems.FindIndex(body)[0]
	return
}

func (r *rss) load(columnist string) error {
	body, err := getPageBody(columnist)
	if err != nil {
		log.Printf("error loading %v: %v\n", columnist, err)
		return err
	}
	t := t1
	r.Date = time.Now().Format("02/01/2006")
	begin, end := getIndexes(t, body)
	if begin == -1 || end == -1 {
		t = t2
		begin, end = getIndexes(t, body)
	}
	r.Title = loadTitle(body)
	r.Image = loadImage(body)
	r.URL = loadURL(body)
	r.Host = loadHost(r.URL)
	r.Desc = loadDesc(columnist, body)
	if "" == r.Desc {
		r.Desc = r.Title
	}
	r.Items = loadItems(t, string(body[begin:end]), r.Title)
	return nil
}

func getPageBody(columnist string) ([]byte, error) {
	log.Println("requesting " + columnist + " page")
	resp, err := http.Get("http://" + contentHost + columnist)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

func buildReadme() {
	log.Println("building readme")
	resp, err := http.Get("http://www.bandnewsfm.com.br/colunistas/")
	if err != nil {
		return
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		return
	}

	validNome := regexp.MustCompile(`<p class="listaNomeColunista"><a href="http://www.bandnewsfm.com.br/colunista/(?P<nome>.+)/">(?P<shortDesc>.+)</a></p>`)
	validDesc := regexp.MustCompile(`<p class="listaIntroColunista"><a href="http://www.bandnewsfm.com.br/colunista/(?P<nome>.+)/">(?P<desc>.+)</a></p>`)
	descriptions := validDesc.FindAllStringSubmatch(string(body), -1)

	for i, s := range validNome.FindAllStringSubmatch(string(body), -1) {
		fmt.Printf("[%v](https://desbravantefm.herokuapp.com/show/%v) - %v\n\n", s[2], s[1], descriptions[i][2])
	}

}
