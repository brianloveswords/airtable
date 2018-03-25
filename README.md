``` go
client := airtable.Client(...)
table := client.Table("Main")

recs, err := table.List()

rec, ok := recs[0]
if !ok {
    // bail out
}

now := time.Now()
id := rec.GetId()
rec.When = time.Now()

err = rec.Save()
if err != nil {
    // something went wrong
}

rec, err := table.Get(id)
if err != nil {
    // something went wrong
}

if rec == nil {
    // record could not be found
}

if rec.When != now {
    // should have saved time
}
```
