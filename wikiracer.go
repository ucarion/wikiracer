package main

import (
    "fmt"
    "os"
    "strings"

    "cgt.name/pkg/go-mwclient"
    "cgt.name/pkg/go-mwclient/params"
    "gopkg.in/alecthomas/kingpin.v2"
)

// Send any more titles in the query than this, and Wikimedia will refuse the
// request
const MAX_TITLES int = 50

type WikiHop struct {
    fromArticle string
    toArticle string
}

var (
    app = kingpin.New("wikiracer", "A tool for quickly finding link paths from one Wikipedia article to another.")

    findCmd = app.Command("find", "Find a single path and exit.")
    sourceArg = findCmd.Arg("source", "Wikipedia article to start from. Can be an article name or URL.").Required().String()
    targetArg = findCmd.Arg("target", "Wikipedia article to look for. Can be an article name or URL.").Required().String()

    serveCmd = app.Command("serve", "Start a RESTful server for finding paths.")
)

func main() {
    wikiClient, err := mwclient.New("https://en.wikipedia.org/w/api.php", "Wikiracer")

    if err != nil {
        panic(err)
    }

    switch kingpin.MustParse(app.Parse(os.Args[1:])) {
        case findCmd.FullCommand():
            fmt.Println(searchLinkPath(wikiClient, *sourceArg, *targetArg))
        case serveCmd.FullCommand():
            fmt.Println("Serve")
    }
}

func searchLinkPath(wikiClient *mwclient.Client, source, target string) string {
    // maps each article to an article that links to it
    reverseHops := make(map[string]string)
    toExplore := []string{source}

    for len(toExplore) > 0 {
        num_to_fetch := min(MAX_TITLES, len(toExplore))
        articles := toExplore[0:num_to_fetch]
        toExplore = toExplore[num_to_fetch:]

        for _, hop := range getLinks(wikiClient, articles) {
            // update `reverseHops` and `toExplore` with new article if it's not
            // already been explored
            if _, ok := reverseHops[hop.toArticle]; !ok {
                reverseHops[hop.toArticle] = hop.fromArticle
                toExplore = append(toExplore, hop.toArticle)
            }

            if hop.toArticle == target {
                return solutionPath(reverseHops, source, target)
            }
        }
    }

    return "No solution found"
}

func getLinks(wikiClient *mwclient.Client, titles []string) []WikiHop {
    queryValues := params.Values{
        "titles": strings.Join(titles, "|"),
        "prop":   "links",
        // Wikimedia has a concept of a "namespace", which distinguishes
        // articles like "Apple", "User:Apple", "Category:Fruits", etc. For our
        // purposes, only namespace 0, corresponding to articles, is relevant.
        "plnamespace": "0",
        "pllimit": "500",
    }

    result := make([]WikiHop, 0)

    query := wikiClient.NewQuery(queryValues)
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

            // if there are multiple titles, not all pages will have links for
            // every title
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

                    result = append(result, WikiHop{title, linkTitle})
                }
            }
        }
    }

    if query.Err() != nil {
        panic(query.Err())
    }

    return result
}

func solutionPath(reverseHops map[string]string, source string, target string) string {
    if source == target {
        return target
    } else {
        previous := reverseHops[target]
        return fmt.Sprintf("%s -> %s", solutionPath(reverseHops, source, previous), target)
    }
}

func min(a, b int) int {
    if a > b {
        return b
    } else {
        return a
    }
}
