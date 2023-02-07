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
			ord := strings.Join(getNoun(words, str), "\n")
			str.Write([]byte(ord))
		case "v":
			ord := strings.Join(getVerb(words, str), ";")
			str.Write([]byte(ord))
		default:
			log.Fatalf("incorrect part of speech specified")
		}
	}

	if err = os.WriteFile("./data/flash-cards.csv", []byte(str.String()), 0644); err != nil {
		log.Fatalf("error opening file: %v", err)
	}
}

func getVerb(words [][]byte, str strings.Builder) []string {
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

	var ord []string
	engWord := string(words[0])

	doc.Find("table.grammar[class*=\"template-sv-verb\"] > tbody").Each(func(i int, tbody *goquery.Selection) {
		if contains(ord, strings.TrimSpace(term)) {
			return
		}

		tbodies := tbody.Children()
		trs := tbodies.Children()

		var twoColumns bool
		if strings.Contains(strings.TrimSpace(trs.Text()), "Passiv") {
			twoColumns = true
		}

		trs.Find("span").Each(func(i int, el *goquery.Selection) {
			word := strings.TrimSpace(el.Text())

			if !twoColumns {
				switch i {
				case 0:
					ord = append(ord, "att "+word+" (to "+engWord+")")
				case 1:
					ord = append(ord, word+" ("+engWord+"ing)")
				case 2:
					ord = append(ord, word+" ("+engWord+"ed)")
				case 3:
					ord = append(ord, "har "+word+" (have "+engWord+"ed)\n")
				case 4:
					ord = append([]string{engWord + ":" + word}, ord...)
				}
			} else {
				switch i {
				case 0:
					ord = append(ord, "att "+word+" (to "+engWord+")")
				case 2:
					ord = append(ord, word+" ("+engWord+"ing)")
				case 4:
					ord = append(ord, word+" ("+engWord+"ed)")
				case 6:
					ord = append(ord, "har "+word+" (have "+engWord+"ed)\n")
				case 8:
					ord = append([]string{engWord + ":" + word}, ord...)
				}
			}
		})
	})

	return ord
}

func getNoun(words [][]byte, str strings.Builder) []string {
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

	var ord []string
	engWord := string(words[0])

	doc.Find("table.grammar[class*=\"template-sv-subst\"] > tbody").Each(func(i int, tbody *goquery.Selection) {
		if contains(ord, strings.TrimSpace(term)) {
			return
		}

		tbodies := tbody.Children()
		trs := tbodies.Children()
		prefix := "en"

		trs.Each(func(i int, els *goquery.Selection) {
			word := strings.TrimSpace(els.Text())

			if i == 3 && word == "neutrum" {
				prefix = "ett"
			}

			switch i {
			case 9:
				ord = append(ord, "a "+engWord+":"+prefix+" "+word)
			case 10:
				ord = append(ord, "the "+engWord+":"+word)
			case 11:
				ord = append(ord, engWord+"s:"+word)
			case 12:
				ord = append(ord, "the "+engWord+"s:"+word+"\n")
			}
		})
	})

	return ord
}

func contains(haystack []string, needle string) bool {
	for _, str := range haystack {
		if strings.Contains(str, needle) {
			return true
		}
	}
	return false
}
