package wikipath

import "fmt"

func Search(source, target string) []string {
    done := make(chan struct{})
    towardSource := make(map[string]string)

    fmt.Println("Starting search ...");

    for hop := range(Explore(done, source)) {
        // only insert if not already present
        if _, ok := towardSource[hop.toArticle]; !ok {
            towardSource[hop.toArticle] = hop.fromArticle
        }

        if hop.toArticle == target {
            return solutionPath(towardSource, source, target)
        }
    }

    return nil
}


func solutionPath(towardSource map[string]string, source string, target string) []string {
    if source == target {
        return []string{target}
    } else {
        previous := towardSource[target]
        return append(solutionPath(towardSource, source, previous), target)
    }
}
