package wikipath

import (
    "fmt"
    "sync"

    "cgt.name/pkg/go-mwclient"
    "cgt.name/pkg/go-mwclient/params"

    "github.com/ucarion/wikiracer/syncset"
)

type Hop struct {
    fromArticle string
    toArticle string
}

type Finder struct {
    wikiClient *mwclient.Client
    source string
    target string

    explored syncset.Set
    reverseHops map[string]string
    waitGroup sync.WaitGroup
}

func Search(wikiClient *mwclient.Client, source, target string) []string {
    finder := Finder{
        wikiClient,
        source,
        target,
        syncset.New(),
        make(map[string]string),
        sync.WaitGroup{},
    }

    hops := make(chan Hop)
    finder.explore(source, hops)

    // There are two possible cases for returning: a path is found, or there's
    // nothing left to explore. Either way, when these cases arrive, the result
    // will be placed in `result`.
    result := make(chan []string)

    // Try to find paths that reach `target`, and consume `hops`.
    go func() {
        for {
            hop := <-hops
            finder.reverseHops[hop.toArticle] = hop.fromArticle
            if hop.toArticle == target {
                result <- solutionPath(finder.reverseHops, source, target)
            }
        }
    }()

    // Try to wait on the WaitGroup, signalling that there are no articles left
    // to explore.
    go func() {
        finder.waitGroup.Wait()
        result <- nil
    }()

    return <-result
}

func (finder *Finder) explore(title string, out chan<- Hop) {
    // do not explore already-explored articles
    if finder.explored.Contains(title) {
        return
    }

    finder.waitGroup.Add(1)
    finder.explored.Insert(title)

    go func() {
        for hop := range(finder.getLinks(title)) {
            out <- hop
            go finder.explore(hop.toArticle, out)
        }
        finder.waitGroup.Done()
    }()
}

func (finder Finder) getLinks(title string) <-chan Hop {
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

        query := finder.wikiClient.NewQuery(queryValues)
        for query.Next() {
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

                        out <- Hop{title, linkTitle}
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

func min(a, b int) int {
    if a > b {
        return b
    } else {
        return a
    }
}

func solutionPath(reverseHops map[string]string, source string, target string) []string {
    if source == target {
        return []string{target}
    } else {
        previous := reverseHops[target]
        return append(solutionPath(reverseHops, source, previous), target)
    }
}
