package wikipath

import (
    "fmt"
    "sync"
    "time"

    "cgt.name/pkg/go-mwclient"
    "cgt.name/pkg/go-mwclient/params"

    "github.com/deckarep/golang-set"
)

const NUM_SIMULTANEOUS_QUERIES int = 10

type Hop struct {
    // the link appeared on this article
    fromArticle string
    // the link's href pointed to this article
    toArticle string
}

func Explore(done <-chan struct{}, source string) <-chan Hop {
    out := make(chan Hop)
    visited := mapset.NewSet()
    waitGroup := sync.WaitGroup{}
    throttle := time.Tick(time.Second / 10)

    var exploreArticle func(string)
    exploreArticle = func(article string) {
        defer waitGroup.Done()

        if visited.Contains(article) {
            return
        }

        visited.Add(article)
        fmt.Printf("Exploring: %s\n", article)
        for hop := range getLinks(article, throttle) {
            out <- hop

            waitGroup.Add(1)
            go exploreArticle(hop.toArticle)
        }
    }

    waitGroup.Add(1)
    go exploreArticle(source)

    go func() {
        waitGroup.Wait()
        close(out)
    }()

    return out
}

var wikiClient *mwclient.Client

func init() {
    var err error
    wikiClient, err = mwclient.New("https://en.wikipedia.org/w/api.php", "Wikiracer")
    if err != nil {
        panic(err)
    }
}

// Wikipedia gives these articles when following redirects
var BLACKLIST mapset.Set = mapset.NewSetFromSlice([]interface{}{"H:L", "H:S"})

func getLinks(title string, throttle <-chan time.Time) <-chan Hop {
    out := make(chan Hop)

    go func() {
        queryValues := params.Values{
            "titles": title,
            "prop":   "links",
            "pllimit": "500",

            // Wikimedia has a concept of a "namespace", which distinguishes
            // articles like "Apple", "User:Apple", "Category:Fruits", etc. For our
            // purposes, only namespace 0, corresponding to articles, is relevant.
            "plnamespace": "0",
        }

        query := wikiClient.NewQuery(queryValues)

        throttledNext := func() bool {
            <-throttle
            return query.Next()
        }

        for throttledNext() {
            response := query.Resp()

            resultPages, err := response.GetObjectArray("query", "pages")
            if err != nil {
                panic(err)
            }

            for _, resultPage := range resultPages {
                title, err := resultPage.GetString("title")
                if err != nil {
                    panic(err)
                }

                // Each "page" corresponds to one of the titles given to Wikimedia,
                // and every response Wikimedia returns will contain a page for each
                // title.
                //
                // Some of these pages will not have any links in them, because all
                // the links are in other pages in this response.
                if _, ok := resultPage.Map()["links"]; ok {
                    links, err := resultPage.GetObjectArray("links")
                    if err != nil {
                        panic(err)
                    }

                    for _, linkObject := range links {
                        linkTitle, err := linkObject.GetString("title")
                        if err != nil {
                            panic(err)
                        }

                        if !BLACKLIST.Contains(linkTitle) {
                            out <- Hop{title, linkTitle}
                        }
                    }
                }
            }
        }

        if query.Err() != nil {
            panic(query.Err())
        }

        close(out)
    }()

    return out
}
