// Copyright 2025 Francisco Oliveto. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// hngrep uses the Hacker News API from the command line to print stories that
// match a PATTERN.
// https://github.com/HackerNews/API

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"sync"
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

func (i item) String() string {
	return fmt.Sprintf("%s\n%s", i.Title, i.URL)
}

const basePath = "https://hacker-news.firebaseio.com/v0"

var (
	news = flag.Bool("new", true, "new stories")
	top  = flag.Bool("top", false, "top stories")
	best = flag.Bool("best", false, "best stories")
)

// TODO: handle error conditions.
func main() {
	log.SetFlags(0)
	flag.Parse()
	if len(flag.Args()) == 0 {
		fmt.Fprintln(os.Stderr, "Usage: hngrep [options] PATTERN")
		flag.PrintDefaults()
		os.Exit(1)
	}
	pattern := flag.Arg(0)

	url := basePath
	switch {
	case *news:
		url += "/newstories.json"
	case *top:
		url += "/topstories.json"
	case *best:
		url += "/beststories.json"
	}
	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	var itemIDs []int
	_ = json.NewDecoder(resp.Body).Decode(&itemIDs)
	resp.Body.Close()
	c := make(chan item, len(itemIDs))
	var gw sync.WaitGroup
	for _, id := range itemIDs {
		gw.Add(1)
		url := basePath + "/item/" + strconv.Itoa(id) + ".json"
		go search(url, pattern, &gw, c)
	}
	go func() {
		gw.Wait()
		close(c)
	}()
	for item := range c {
		fmt.Println(item)
	}
}

// search queries for the item represented by url, and sends it to channel c if
// it matches the pattern.
func search(url string, pattern string, gw *sync.WaitGroup, c chan<- item) {
	defer gw.Done()
	resp, _ := http.Get(url)
	defer resp.Body.Close()
	var item item
	_ = json.NewDecoder(resp.Body).Decode(&item)
	matched, _ := regexp.MatchString(pattern, item.Title)
	if matched {
		c <- item
	}
}
