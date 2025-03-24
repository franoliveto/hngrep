// Copyright 2025 Francisco Oliveto. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// news uses the Hacker News API from the command line to print stories that
// match a PATTERN.
// https://github.com/HackerNews/API

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
)

type item struct {
	ID          int
	Deleted     bool
	Type        string // the type of item. One of "job", "story", "comment", "poll", or "pollopt".
	By          string // the username of the item's author.
	Time        int64  // creation date of the item, in Unix Time.
	Text        string // the comment, story or pool text. HTML.
	Dead        bool   // true if the item is dead.
	Parent      int    // the comment's parent: either another comment or the relevant story.
	Poll        int    // the pollopt's associated poll.
	Kids        []int  // the ids of the item's comments, in ranked display order.
	URL         string // the URL of the story
	Score       int
	Title       string // the title of the story, poll or job. HTML.
	Parts       []int
	Descendants int // in the case of stories or polls, the total comment count.
}

const basePath = "https://hacker-news.firebaseio.com/v0"

var (
	news = flag.Bool("new", true, "new stories")
	top  = flag.Bool("top", false, "top stories")
	best = flag.Bool("best", false, "best stories")
)

// TODO: handle errors.
func main() {
	log.SetFlags(0)
	flag.Parse()
	if len(flag.Args()) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: news [options] PATTERN")
		flag.PrintDefaults()
		os.Exit(1)
	}
	pattern := flag.Arg(0)

	var which string
	switch {
	case *news:
		which = "new"
	case *top:
		which = "top"
	case *best:
		which = "best"
	}
	stories, err := getStories(which)
	if err != nil {
		log.Fatal(err)
	}
	c := make(chan item, len(stories))
	for _, id := range stories {
		url := basePath + "/item/" + strconv.Itoa(id) + ".json"
		go fetch(url, c)
	}
	search := func(pattern string) (*searchResult, error) {
		var items []item
		for range stories {
			item := <-c
			matched, _ := regexp.MatchString(pattern, item.Title)
			if matched {
				items = append(items, item)
			}
		}
		return &searchResult{Total: len(items), Items: items}, nil
	}
	result, err := search(pattern)
	if err != nil {
		log.Fatal(err)
	}
	if err := print(result); err != nil {
		log.Fatal(err)
	}
}

type searchResult struct {
	Total int
	Items []item
}

func print(r *searchResult) error {
	const templ = `
<h1>{{.Total}} Hacker News stories</h1>
<table style='border-spacing: 5px'>
<tr style='text-align: left'>
	<th>#</th>
	<th>points</th>
	<th>comments</th>
	<th>author</th>
	<th>title</th>
</tr>
{{range .Items}}
<tr>
	<td>{{.ID}}</td>
	<td>{{.Score}}</td>
	<td>{{.Descendants}}</td>
	<td>{{.By}}</td>
	<td><a href='{{.URL}}'>{{.Title}}</a></td>
</tr>
{{end}}
</table>
`
	t := template.Must(template.New("").Parse(templ))
	if err := t.Execute(os.Stdout, r); err != nil {
		return err
	}
	return nil
}

func getStories(which string) ([]int, error) {
	url := "https://hacker-news.firebaseio.com/v0/" + which + "stories.json"
	var stories []int
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	_ = json.NewDecoder(resp.Body).Decode(&stories)
	resp.Body.Close()
	return stories, nil
}

func fetch(url string, c chan<- item) {
	resp, _ := http.Get(url)
	defer resp.Body.Close()
	var item item
	_ = json.NewDecoder(resp.Body).Decode(&item)
	c <- item
}
