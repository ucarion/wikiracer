package main

import (
    "encoding/json"
    "fmt"
    "github.com/ucarion/wikiracer/wikipath"
    "gopkg.in/alecthomas/kingpin.v2"
    "log"
    "net/http"
    "os"
)

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
    switch kingpin.MustParse(app.Parse(os.Args[1:])) {
        case findCmd.FullCommand():
            result, err := wikipath.Search(*sourceArg, *targetArg)

            if err != nil {
                fmt.Println("Error:", err)
                os.Exit(1)
            }

            if *formatArg == "human" {
                fmt.Println(linkPathToString(result))
            } else {
                fmt.Println(linkPathToJson(result))
            }
        case serveCmd.FullCommand():
            http.HandleFunc("/find", httpHandler)
            log.Fatal(http.ListenAndServe(":8080", nil))
    }
}

func httpHandler(w http.ResponseWriter, r *http.Request) {
    params := r.URL.Query()
    source, target := params["source"], params["target"]

    if source == nil || len(source) != 1 || target == nil || len(target) != 1 {
        status := http.StatusBadRequest
        http.Error(w, http.StatusText(status), status)
        return
    }

    result, err := wikipath.Search(source[0], target[0])

    if err != nil {
        errorMessage := fmt.Sprintf("Error: %s", err)
        errorMessageJson, err := json.Marshal(errorMessage)
        if err != nil {
            panic(err)
        }

        fmt.Fprintf(w, string(errorMessageJson))
        return
    }

    fmt.Fprintf(w, linkPathToJson(result))
}

func linkPathToJson(path []string) string {
    pathJson, err := json.Marshal(path)
    if err != nil {
        panic(err)
    }

    return string(pathJson)
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
