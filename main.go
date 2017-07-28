package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
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

var logger = log.New(os.Stdout, "desbravante: ", log.Lshortfile)

//publisher started to use this layout
type track struct {
	Src         string
	Caption     string
	Title       string
	Description string
	Image       string
}

type item struct {
	Src    string
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

var scriptBody = regexp.MustCompile(`<script type="application/json" class="wp-playlist-script">(?P<body>.+)</script>`)

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
<enclosure url="{{.Src}}" type="audio/mpeg"/>
<itunes:duration>01:00</itunes:duration>
<itunes:summary>{{.Title}}</itunes:summary>
<itunes:author>{{.Author}}</itunes:author>
<itunes:keywords/>
</item>
{{end}}
</channel>
</rss>`))

func sendRss(w http.ResponseWriter, data *rss) {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	/*
		if err := json.NewEncoder(w).Encode(loadItens(strBody[begin:end])); err != nil {
			logger.Println("SEVERE: %v error returning json response \n", err)
		}
	*/
	if err := rssBody.Execute(w, *data); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
	}
	return
}

func main() {
	cache := NewCache("careca-de-saber-com-leandro-karnal", "e-o-bicho", "futebol-com-milton-neves", "jose-simao", "karnal", "politica-com-dora-kramer", "reinaldo-azevedo", "ricardo-boechat")
	//cache := NewCache("careca-de-saber-com-leandro-karnal", "e-o-bicho")
	http.HandleFunc("/show/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			a_path := strings.Split(r.URL.Path, "/")

			if "" != a_path[2] { //by id
				col := a_path[2]
				if "karnal" == a_path[2] {
					col = "careca-de-saber-com-leandro-karnal"
				}

				if rss, has := cache.Get(col); has && r != nil {
					sendRss(w, rss)
					return
				}

				data := rss{}
				if err := data.load(col); err == nil {
					cache.Set(col, &data)
				}

				sendRss(w, &data)
				return
			}
			if content, err := json.Marshal(cache.feeds); err == nil {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusOK)
				w.Write(content)
			} else {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}

	})
	var port = os.Getenv("PORT")
	// Set a default port if there is nothing in the environment
	if port == "" {
		port = "8080"
		logger.Println("INFO: No PORT environment variable detected, defaulting to " + port)
	}
	logger.Fatal(http.ListenAndServe(":"+port, nil))
}

func (i item) String() string {
	return fmt.Sprintf("{\"title\":%q, \"date\":%q, \"id\":%q}", i.Title, i.Date, i.Src)
}

func loadTitle(body []byte) string {
	return loadMetaData(`<meta property="og:title" content="(?P<title>.+)" />`, body)
}

func loadImage(body []byte) string {
	return loadMetaData(`<meta property="og:image" content="(?P<title>.+)" />`, body)
}

func loadDesc(column string, body []byte) string {
	logger.Println("loading description")
	if resp, err := http.Get("http://www.bandnewsfm.com.br/colunistas/"); err == nil {
		defer resp.Body.Close()
		if resp.StatusCode >= http.StatusBadRequest {
			logger.Printf("description page returned status code %d.", resp.StatusCode)
			return ""
		}

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
		logger.Println("erro parsing URL %v - %v", u, err)
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
		date := validDate.FindString(title)
		if date == "" {
			logger.Println("Date not found - " + title)
		} else {
			mediaID := validMediaID.FindStringSubmatch(body[end:])
			if mediaID == nil || len(mediaID) != 2 {
				logger.Println("mediaID not found - " + title)
			} else {
				dateLen := len(date)
				if d, err := time.Parse("02/01/2006", date); err == nil {
					date = d.Format(time.RFC822)
				}
				itens = append(itens, item{Date: date, Title: title[dateLen:], Src: fmt.Sprintf("http://video.m.mais.uol.com.br/%v.mp3", mediaID[1]), Author: author})
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
		logger.Printf("error loading %v: %v\n", columnist, err)
		return err
	}

	r.Title = loadTitle(body)
	r.Image = loadImage(body)
	r.URL = loadURL(body)
	r.Host = loadHost(r.URL)
	r.Desc = loadDesc(columnist, body)
	if "" == r.Desc {
		r.Desc = r.Title
	}

	if t := getTracks(body); t != nil {
		r.Items = loadItemsFromTracks(&t, r.Title)
		return nil
	}

	t := t1
	r.Date = time.Now().Format(time.RFC822)
	begin, end := getIndexes(t, body)
	if begin == -1 || end == -1 {
		t = t2
		begin, end = getIndexes(t, body)
	}
	r.Items = loadItems(t, string(body[begin:end]), r.Title)
	return nil
}

func loadItemsFromTracks(t *[]track, author string) []item {
	logger.Println("tracks to items")
	itens := make([]item, len(*t), len(*t))
	validDate := regexp.MustCompile(`[0-9]+\/[0-9]+\/[0-9]+`)
	buffer := bytes.NewBuffer([]byte{})
	for i, v := range *t {
		date := validDate.FindString(v.Title)
		dateLen := len(date)
		if d, err := time.Parse("02/01/2006", date); err == nil {
			date = d.Format(time.RFC822)
		}
		if err := xml.EscapeText(buffer, []byte(v.Title[dateLen:])); err == nil {
			itens[i] = item{Title: buffer.String(), Date: date, Src: v.Src, Author: author}
		} else {
			itens[i] = item{Title: v.Title[dateLen:], Date: date, Src: v.Src, Author: author}
		}
		buffer.Reset()
	}
	return itens
}

func getPageBody(columnist string) ([]byte, error) {
	logger.Println("requesting " + columnist + " page")
	resp, err := http.Get("http://" + contentHost + columnist)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, fmt.Errorf("feed returned status code %d", resp.StatusCode)
	}

	return ioutil.ReadAll(resp.Body)
}

func getTracks(body []byte) []track {
	logger.Println("loading tracks")
	matches := scriptBody.FindSubmatch(body)
	if len(matches) != 2 {
		logger.Printf("returned: \n%v\n%v", len(matches), string(body))
		return nil
	}
	var t = struct {
		Tracks []track
	}{}

	err := json.Unmarshal(matches[1], &t)
	if err != nil {
		logger.Printf("error unquoting json body %v : \n %v", string(matches[1]), err)
		return nil
	}
	return t.Tracks

	/*
		str, err := strconv.Unquote(matches[1])
		if err != nil {
			logger.Printf("error unquoting json body %v : \n %v", matches[1], err)
			return ""
		}

		return str
	*/
}

func buildReadme() {
	logger.Println("building readme")
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
