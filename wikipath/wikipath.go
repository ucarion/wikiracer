package wikipath

// import "fmt"
// import "encoding/json"

func Search(pool *ExplorerPool, source, target string) []string {
    towardSource := make(map[string]string)

    for hop := range(pool.Explore(source)) {
        towardSource[hop.toArticle] = hop.fromArticle

        if hop.toArticle == target {
            return solutionPath(towardSource, source, target)
        }
    }

    return nil
}


// func PrettyPrint(v interface{}) {
//       b, _ := json.MarshalIndent(v, "", "  ")
//       println(string(b))
// }

func solutionPath(towardSource map[string]string, source string, target string) []string {
    if source == target {
        return []string{target}
    } else {
        previous := towardSource[target]
        return append(solutionPath(towardSource, source, previous), target)
    }
}
