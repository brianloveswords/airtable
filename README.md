# airtable
Go package for interacting with the Airtable API.

## License

[Mozilla Public License 2.0](https://www.mozilla.org/en-US/MPL/2.0/)

## Install

``` shell
$ go get github.com/brianloveswords/airtable
```

## API Documentation
See [airtable package documentation on godoc.org](https://godoc.org/github.com/brianloveswords/airtable)

## Example Usage
``` go
package main

import (
    "fmt"
    "strings"
    "time"

    "github.com/brianloveswords/airtable"
)

type PublicDomainBookRecord struct {
    airtable.Record // provides ID, CreatedTime
    Fields          struct {
        Title       string `json:"Book Title"`
        Author      string
        Publication time.Time `json:"Publication Date"`
        FullText    string
        Rating      int
        Tags        airtable.MultiSelect
    }
}

// String shows the book record like "<title> by <author> [<rating>]"
func (r *PublicDomainBookRecord) String() string {
    f := r.Fields
    return fmt.Sprintf("%s by %s %s", f.Title, f.Author, r.Rating())
}

// Rating outputs a rating like [***··]
func (r *PublicDomainBookRecord) Rating() string {
    var (
        max    = 5
        rating = r.Fields.Rating
        stars  = strings.Repeat("*", rating)
        dots   = strings.Repeat("·", max-rating)
    )
    return fmt.Sprintf("[%s%s]", stars, dots)
}

func Example() {
    // Create the Airtable client with your APIKey and BaseID for the
    // base you want to interact with.
    client := airtable.Client{
        APIKey: "keyXXXXXXXXXXXXXX",
        BaseID: "appwNa5g4gHCVZQPm",
    }

    books := client.Table("Public Domain Books")

    bestBooks := []PublicDomainBookRecord{}
    books.List(&bestBooks, &airtable.Options{
        // The whole response would be huge because of FullText so we
        // should just get the title and author. NOTE: even though the
        // field is called "Book Title" in the JSON, we should use field
        // by the name we defined it in our struct.
        Fields: []string{"Title", "Author", "Rating"},

        // Only get books with a rating that's 4 or higher.
        Filter: `{Rating} >= 4`,

        // Let's sort from highest to lowest rating, then by author
        Sort: airtable.Sort{
            {"Rating", airtable.SortDesc},
            {"Author", airtable.SortAsc},
        },
    })

    fmt.Println("Best Public Domain Books:")
    for _, bookRecord := range bestBooks {
        fmt.Println(bookRecord.String())
    }

    // Let's prune our library of books we aren't super into.
    badBooks := []PublicDomainBookRecord{}
    books.List(&badBooks, &airtable.Options{
        Fields: []string{"Title", "Author", "Rating"},
        Filter: `{Rating} < 3`,
    })
    for _, badBook := range badBooks {
        fmt.Println("deleting", badBook)
        books.Delete(&badBook)
    }
}
```

## Contributing

TBD
