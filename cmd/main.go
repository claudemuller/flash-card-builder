package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

type parsedResult struct {
	Parse struct {
		Title string
		Text  map[string]string
	}
}

func main() {
	log.SetPrefix("[vocab-builder] ")

	fname := flag.String("w", "", "the words file to look up")
	flag.Parse()

	if flag.NFlag() == 0 {
		flag.Usage()
		os.Exit(0)
	}

	// Do a thing here...
	data, err := os.ReadFile(*fname)
	if err != nil {
		log.Fatalf("there was a problem reading %s: %s", *fname, err.Error())
	}

	data = bytes.Trim(data, "\n")
	lines := bytes.Split(data, []byte("\n"))
	str := strings.Builder{}

	for _, line := range lines {
		parts := bytes.Split(line, []byte("|"))
		words := bytes.Split(parts[1], []byte(":"))

		switch pos := string(parts[0]); pos {
		case "n":
			term := string(words[1])
			q := url.PathEscape(term)
			url := "https://sv.wiktionary.org/w/api.php?action=parse&format=json&prop=text|revid|displaytitle&callback=?&page=" + q

			log.Printf("running -> %s\n", url)

			res, err := http.Get(url)
			if err != nil {
				log.Fatalf("error hitting API: %v", err)
			}
			defer res.Body.Close()

			data, err := io.ReadAll(res.Body)
			if err != nil {
				log.Fatalf("error reading response body: %v", err)
			}

			data = data[5 : len(data)-1]

			var parsed parsedResult

			if err = json.Unmarshal(data, &parsed); err != nil {
				log.Fatalf("error unmarshalling data: %v", err)
			}

			sr := strings.NewReader(parsed.Parse.Text["*"])
			doc, err := goquery.NewDocumentFromReader(sr)
			if err != nil {
				log.Fatal(err)
			}

			engWord := string(words[0])

			doc.Find("table.grammar[class*=\"template-sv-subst\"] > tbody").Each(func(i int, tbody *goquery.Selection) {
				if strings.Contains(str.String(), term) {
					return
				}

				tbodies := tbody.Children()
				trs := tbodies.Children()
				prefix := "en"

				trs.Each(func(i int, els *goquery.Selection) {
					if i == 3 && strings.TrimSpace(els.Text()) == "neutrum" {
						prefix = "ett"
					}

					switch i {
					case 9:
						str.Write([]byte("a " + engWord + ":" + prefix + " " + els.Text()))
					case 10:
						str.Write([]byte("the " + engWord + ":" + els.Text()))
					case 11:
						str.Write([]byte(engWord + "s:" + els.Text()))
					case 12:
						str.Write([]byte("the " + engWord + "s:" + els.Text()))
					}
				})
			})
		}
	}

	if err = os.WriteFile("../data/flash-cards.csv", []byte(str.String()), 0644); err != nil {
		log.Fatalf("error opening file: %v", err)
	}
}
