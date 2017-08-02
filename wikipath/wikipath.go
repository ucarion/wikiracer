package wikipath

import "log"

func Search(source, target string) []string {
    done := make(chan struct{})
    defer close(done)

    // reverse links from articles toward articles "closer" to source
    towardSource := make(map[string]string)
    // forward links from articles toward articles "closer" to target
    towardTarget := make(map[string]string)

    log.Println("Starting search ...");
    fromSource := Explore(done, source, true)
    fromTarget := Explore(done, target, false)

    for pair := range merge(fromSource, fromTarget) {
        isFromSource := pair.bool
        hop := pair.Hop

        if isFromSource {
            // if not yet known, add info about path back to source
            if _, ok := towardSource[hop.toArticle]; !ok {
                towardSource[hop.toArticle] = hop.fromArticle
            }

            // see if a path from toArticle to target is known
            if _, ok := towardTarget[hop.toArticle]; ok {
                return solutionPath(towardSource, towardTarget, source, target, hop.toArticle)
            }
        } else {
            // if not yet known, add info about path toward target
            if _, ok := towardTarget[hop.fromArticle]; !ok {
                towardTarget[hop.fromArticle] = hop.toArticle
            }

            // see if a path from fromArticle to source is known
            if _, ok := towardSource[hop.fromArticle]; ok {
                return solutionPath(towardSource, towardTarget, source, target, hop.fromArticle)
            }
        }
    }

    return nil
}

// merge both streams, and close the output as once either input closes
func merge(fromSource, fromTarget <-chan Hop) <-chan struct{Hop; bool} {
    out := make(chan struct{Hop; bool})
    stopped := false

    output := func(c <-chan Hop, isFromSource bool) {
        for hop := range c {
            if !stopped {
                out <- struct{Hop; bool}{hop, isFromSource}
            }
        }

        stopped = true
        close(out)
    }

    go output(fromSource, true)
    go output(fromTarget, false)

    return out
}

func solutionPath(
    towardSource, towardTarget map[string]string,
    source, target, middle string,
) []string {
    sourceToMiddle := halfSolutionPath(towardSource, source, middle)
    middleToTarget := halfSolutionPath(towardTarget, target, middle)
    reverse(middleToTarget)

    return append(append(sourceToMiddle, middle), middleToTarget...)
}

// returns a path through `links` from `start` to `objective`, excluding `start`.
func halfSolutionPath(links map[string]string, objective, start string) []string {
    if start == objective {
        return []string{}
    } else {
        next := links[start]
        return append(halfSolutionPath(links, objective, next), next)
    }
}

func reverse(a []string) {
    // https://github.com/golang/go/wiki/SliceTricks#reversing
    for i := len(a) / 2 - 1; i >= 0; i-- {
        opp := len(a) - 1 - i
        a[i], a[opp] = a[opp], a[i]
    }
}
