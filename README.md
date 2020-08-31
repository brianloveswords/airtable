我是光年实验室高级招聘经理。
我在github上访问了你的开源项目，你的代码超赞。你最近有没有在看工作机会，我们在招软件开发工程师，拉钩和BOSS等招聘网站也发布了相关岗位，有公司和职位的详细信息。
我们公司在杭州，业务主要做流量增长，是很多大型互联网公司的流量顾问。公司弹性工作制，福利齐全，发展潜力大，良好的办公环境和学习氛围。
公司官网是http://www.gnlab.com,公司地址是杭州市西湖区古墩路紫金广场B座，若你感兴趣，欢迎与我联系，
电话是0571-88839161，手机号：18668131388，微信号：echo 'bGhsaGxoMTEyNAo='|base64 -D ,静待佳音。如有打扰，还请见谅，祝生活愉快工作顺利。

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
