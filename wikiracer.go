package main

import (
    "encoding/json"
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
    formatArg = findCmd.
        Flag("format", "Display output in human-friendly way (--format=human) or as JSON (--format=json).").
        Default("human").
        Enum("human", "json")

    serveCmd = app.Command("serve", "Start a RESTful server for finding paths.")
)

func main() {
    wikiClient, err := mwclient.New("https://en.wikipedia.org/w/api.php", "Wikiracer")

    if err != nil {
        panic(err)
    }

    switch kingpin.MustParse(app.Parse(os.Args[1:])) {
        case findCmd.FullCommand():
            result := searchLinkPath(wikiClient, *sourceArg, *targetArg)
            if *formatArg == "human" {
                fmt.Println(linkPathToString(result))
            } else {
                resultJson, err := json.Marshal(result)
                if err != nil {
                    panic(err)
                }

                fmt.Println(string(resultJson))
            }
        case serveCmd.FullCommand():
            fmt.Println("Serve")
    }
}

func searchLinkPath(wikiClient *mwclient.Client, source, target string) []string {
    // TODO normalize before searching

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

    return nil
}

func getLinks(wikiClient *mwclient.Client, titles []string) []WikiHop {
    queryValues := params.Values{
        "titles": strings.Join(titles, "|"),
        "prop":   "links",
        "pllimit": "500",

        // Wikimedia has a concept of a "namespace", which distinguishes
        // articles like "Apple", "User:Apple", "Category:Fruits", etc. For our
        // purposes, only namespace 0, corresponding to articles, is relevant.
        "plnamespace": "0",
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

func linkPathToString(path []string) string {
    if path == nil {
        return "No path found."
    } else {
        result := path[0]
        for _, hop := range path[1:] {
            result = fmt.Sprintf("%s -> %s", result, hop)
        }
        return result
    }
}
