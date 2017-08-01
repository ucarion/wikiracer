package main

import (
    "encoding/json"
    "fmt"
    "os"

    "cgt.name/pkg/go-mwclient"
    "gopkg.in/alecthomas/kingpin.v2"
    "github.com/ucarion/wikiracer/wikipath"
)

// Send any more titles in the query than this, and Wikimedia will refuse the
// request
const MAX_TITLES int = 50

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
            result := wikipath.Search(wikiClient, *sourceArg, *targetArg)
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
