# Wikiracer

`wikiracer` is a tool that quickly finds a way to get between any two Wikipedia
articles, following only links in the articles. The emphasis is on *speed of
finding paths*, *not* on producing the shortest path.

## Examples

These examples redirect logging (which appears on stderr), so that only the
result is shown. The logs have information about articles being explored, as
well as how long the query took to complete.

### From the command-line

```bash
# The provided example query. My result is much more hilarious than yours was.
$ wikiracer find "Mike Tyson" "Segment" --format=json 2>/dev/null
["Mike Tyson","Adultery","Romania","Human rights in Romania","Segregation","Segment (disambiguation)","Segment"]
```

```bash
# Find a path from my current hometown to my old hometown in middle-of-nowhere,
# Northern France
$ time wikiracer find "La Jolla" "Lebucquière" 2>/dev/null
La Jolla -> Balboa Park (San Diego) -> Woodrow Wilson -> Arras -> Lebucquière
1.42s user 0.12s system 58% cpu 2.648 total
```

### Using the server

In one terminal, run:

```bash
$ wikiracer serve
```

And in another terminal, run:

```bash
$ curl "localhost:8080/find?source=Apple&target=Zimbabwe"
["Apple","China","Zimbabwe"]
```

## Testing it yourself with Docker

This project includes a Dockerfile, so you can test it yourself by running:

```bash
$ sudo docker build -t ucarion/wikiracer .
$ sudo docker run -it ucarion/wikiracer
```

This will install and run integration tests on `wikiracer`. If you want to test
it yourself, append `bash` to that `run` command above to override the default
command, which is `./integration_test.sh`.

## How it works, and how I got here

### High-level overview

The code is split into four parts, each corresponding to a file:

* `explorer.go`: Interacts with the Wikimedia API. It exposes two functions: one
  to normalize article titles, and another which "explores" the link graph and
produces an infinite stream of links discovered.

* `wikipath.go`: Handles the logic of finding a path, exposed as
  `wikipath.Search`. It fires off two exploratory streams (a "forward" stream
following links from the source, and a "backward" stream following backlinks
from the target), inserting the results into a pair of dictionaries, each
documenting how to reach the source or target. When these dictionaries share a
key, it constructs a solution from these two dicts.

* `wikiracer.go`: The main package. Handles command-line commands, starting an
  HTTP server, and calling `wikipath.Search` with the user's query.

* `integration_test.sh`: A Bash script which makes sure things like dead ends,
  trivial solutions, a user inputting a Wikipedia URL, and the happy path all
work correctly.

> Aside: Incidentally, the only consistently-found path I could find (race
> conditions over the network affect what results are found) is from "Kevin
> Bacon" to "Paul Erdos", via "Erdos Number". Since you can measure *actors* by
> how many costar-hops they are from Mr.  Bacon, and *scientists* by how many
> coauthor-hops they are from Mr.  Erdos, it's quite apropos.

**Searching from the front and back simultaneously is an important
optimization.** To see this, imagine Wikipedia link-paths are about 6 steps
long, and every article links to 100 others.

* Exploring just from the front means exploring every article 6 hops from the
  source, of which there are `100^6`.

* Exploring from both sides means exploring every article 3 hops from either the
  source or target; if the path is of length 6, one of these articles is sure to
be 3 hops from both. There are `2 * 100^3` such articles. This is half a million
times smaller than the previous number.

Wikimedia provides a backlink ("What links here?") API, and it divides the
exponential coefficient of the search's runtime by two. Taking advantage of this
yields a dramatic improvement.

### Strategies tried

**This was my first time writing anything in Go.** It took me awhile to figure out
how to do useful things with goroutines and channels. My first solution only did
a forward search, exploring links until one of them linked to the target. It
didn't use concurrency or channels at all.

I then spent a few hours trying a bunch of different ways to parallelize the
search. A big problem here was that I was sending too many queries to Wikipedia
at once, and thus crashing as a result (I *believe* I was filling up my OS's
queue of outgoing connections. Wikipedia was replying with HTTP/2
[GOAWAY][http2-goaway] messages, so I couldn't initiate connections with them).
I spent far too long trying to code up coordinating worker pools which could
handle two simultaneous requests from two API calls; such an approach could
work, but it was too much for a Go n00b like myself. That night, I instead read
about and played with different concurrency patterns in Go, and I started to
grok how Go works.

[http2-goaway]: https://http2.github.io/http2-spec/#GOAWAY

Everything got better once I just used throttling. Go makes this easy, since
it's just a question of reading from a channel that emits something every tenth
of a second before firing off another API call. *Keep it simple, stupid*.

In total, I spent a day and half on this project.

### Possible improvements

I didn't write any unit tests for this. As mentioned before, I do have
integration tests, but the logic for finding a path given a channel for "forward
search" and for "backward search" could be tested directly. I could also check
that my "extract a title from a Wikipedia article" regex-based approach works
correctly in all cases. This would also be an opportunity to familiarize myself
with how unit testing works within Go-land.

I don't think I'm doing cancellation right. As mentioned in [this
blogpost][golang-blog] from the Golang devs about concurrency patterns in Go,
it's common to use a `<-chan struct{}` called `done`, which you close when the
reader is done consuming from the producer, to quench the producer. I use this
pattern, but I have the suspicion I'm not checking it everywhere/as often as I
should be, because the producers are slow to all shut themselves down. I'd
*really* like to be sure I'm not leaking goroutines, since that would eventually
crash the server.

Error handling could be improved. I have lots of places in the code where I
panic in case of unexpected errors (such as Wikimedia's response not being
shaped as expected); in a production environment, it would likely be better to
log the error and abort the query, responding with an HTTP 500. Currently, only
errors related bad user input are bubbled back up to the user. I didn't elect to
refactor error-handling because I'm not very familiar with error-handling
patterns in Go. Right now, it seems like Go's handling of errors is scarcely
better than C's, which suggests to me that I don't understand Go's way of doing
things yet.

[golang-blog]: https://blog.golang.org/pipelines

## Usage

```
$ wikiracer
usage: wikiracer [<flags>] <command> [<args> ...]

A tool for quickly finding link paths from one Wikipedia article to another.

Flags:
  --help  Show context-sensitive help (also try --help-long and --help-man).

Commands:
  help [<command>...]
    Show help.

  find [<flags>] <source> <target>
    Find a single path and exit.

  serve
    Start a RESTful server for finding paths.
```

```
$ wikiracer help find
usage: wikiracer find [<flags>] <source> <target>

Find a single path and exit.

Flags:
  --help          Show context-sensitive help (also try --help-long and --help-man).
  --format=human  Display output in human-friendly way (--format=human) or as JSON (--format=json).

Args:
  <source>  Wikipedia article to start from. Can be an article name or URL.
  <target>  Wikipedia article to look for. Can be an article name or URL.
```
