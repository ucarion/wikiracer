package wikipath

import (
    "fmt"
    "sync"

    "cgt.name/pkg/go-mwclient"
    "cgt.name/pkg/go-mwclient/params"

    "github.com/ucarion/wikiracer/syncset"
)

const NUM_WORKERS_PER_POOL int = 10

type ExplorerPool struct {
    // workers consume tasks from this pool
    tasks <-chan ExploreTask
}

type ExploreTask struct {
    // article to start exploring from
    article string
    // articles already explored
    explored *syncset.Set
    // output channel
    out chan<- Hop
    // stores number of articles being explored for query
    waitGroup *sync.WaitGroup
}

type Hop struct {
    // the link appeared on this article
    fromArticle string
    // the link's href pointed to this article
    toArticle string
}

// Create pool with workers ready to go
func NewExplorerPool() ExplorerPool {
    pool := ExplorerPool{make(chan ExploreTask)}

    for i := 0; i < NUM_WORKERS_PER_POOL; i++ {
        go func() {
            for {
                pool.processTask(<-pool.tasks)
            }
        }()
    }

    return pool
}

func (pool *ExplorerPool) Explore(article string) <-chan Hop {
    explored := syncset.New()
    out := make(chan Hop)
    waitGroup := sync.WaitGroup{}

    pool.processTask(ExploreTask{article, &explored, out, &waitGroup})

    go func() {
        waitGroup.Wait()
        close(out)
    }()

    return out
}

func (pool *ExplorerPool) processTask(task ExploreTask) {
    fmt.Printf("Process task: %s\n", task.article)
    if task.explored.Contains(task.article) {
        return
    }

    task.explored.Insert(task.article)
    task.waitGroup.Add(1)
    go func() {
        for hop := range getLinks(task.article) {
            task.out <- hop
            nextTask := ExploreTask{hop.toArticle, task.explored, task.out, task.waitGroup}
            pool.processTask(nextTask)
        }

        task.waitGroup.Done()
    }()
}

var wikiClient *mwclient.Client

func init() {
    var err error
    wikiClient, err = mwclient.New("https://en.wikipedia.org/w/api.php", "Wikiracer")
    if err != nil {
        panic(err)
    }
}

func getLinks(title string) <-chan Hop {
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
