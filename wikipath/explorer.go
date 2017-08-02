package wikipath

import (
    "cgt.name/pkg/go-mwclient"
    "cgt.name/pkg/go-mwclient/params"
    "github.com/antonholmquist/jason"
    "github.com/deckarep/golang-set"
    "log"
    "sync"
    "time"
)

const NUM_SIMULTANEOUS_QUERIES int = 10

type Hop struct {
    // the link appeared on this article
    fromArticle string
    // the link's href pointed to this article
    toArticle string
}

func Explore(done <-chan struct{}, source string, forward bool) <-chan Hop {
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
        log.Println("Exploring article:", article)
        for hop := range getLinks(done, article, forward, throttle) {
            select {
            case out <- hop:
            case <-done:
                return
            }

            waitGroup.Add(1)
            if forward {
                go exploreArticle(hop.toArticle)
            } else {
                go exploreArticle(hop.fromArticle)
            }
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

func getLinks(done <-chan struct{}, title string, forward bool, throttle <-chan time.Time) <-chan Hop {
    out := make(chan Hop)

    go func() {
        query := wikiClient.NewQuery(queryParams(title, forward))

        throttledNext := func() bool {
            select {
            case <-done:
                return false
            default:
                <-throttle
                return query.Next()
            }
        }

        for throttledNext() {
            response := query.Resp()

            var resultKey string
            if forward {
                resultKey = "pages"
            } else {
                resultKey = "backlinks"
            }

            resultPages, err := response.GetObjectArray("query", resultKey)
            if err != nil {
                panic(err)
            }

            for _, resultPage := range resultPages {
                processResultPage(out, forward, title, resultPage)
            }
        }

        if query.Err() != nil {
            panic(query.Err())
        }

        close(out)
    }()

    return out
}

func processResultPage(out chan<- Hop, forward bool, queryTitle string, resultPage *jason.Object) {
    title, err := resultPage.GetString("title")
    if err != nil {
        panic(err)
    }

    if !forward {
        out <- Hop{title, queryTitle}
        return
    }

    // Each "page" corresponds to one of the titles given to Wikimedia, and
    // every response Wikimedia returns will contain a page for each title.
    //
    // Some of these pages will not have any links in them, because all the
    // links are in other pages in this response.
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

func queryParams(title string, forward bool) params.Values {
    if forward {
        return params.Values{
            "titles": title,
            "prop":   "links",
            "pllimit": "500",

            // Wikimedia has a concept of a "namespace", which distinguishes
            // articles like "Apple", "User:Apple", "Category:Fruits", etc. For our
            // purposes, only namespace 0, corresponding to articles, is relevant.
            "plnamespace": "0",
        }
    } else {
        return params.Values {
            "list": "backlinks",
            "bltitle": title,
            "bllimit": "500",

            // See comment above for explanation.
            "blnamespace": "0",
        }
    }
}
