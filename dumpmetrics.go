package main

// read in the WeRelate dump file with the help of an index file 
// (to detect forward references) and generate a more detailed CSV

import (
    "fmt"
    "os"
    "io"
    "encoding/csv"
    "strconv"
    "encoding/xml"
    "regexp"
    "strings"
    "country"
)

// the index we build up of the pages
type PageIndex struct {
    title string
    start int64
    end   int64
}

// record for data from xml
type Page struct {
	Title string `xml:"title"`
	Text string `xml:"revision>text"`
}
// fetch a range of bytes from the given file
func fetchrange(in *os.File, start, end int64) string {
    fmt.Println("fetching byte range", start, "to", end);
    buf := make([]byte, end-start)
    in.ReadAt(buf, start)
    return string(buf[:]);
}

func splittitle(str string) (namespace string, title string) {
    re := regexp.MustCompile("^([A-Za-z]+|[A-Za-z]+ talk):(.+)")
    if (re.MatchString(str)) {
        m := re.FindAllStringSubmatch(str, -1)
        namespace = m[0][1]
        title = m[0][2]
    } else {
        namespace = ""
        title = str
    }
    return namespace, title
}

type Event struct {
    Type    string `xml:"type,attr"`
    Date    string `xml:"date,attr"`
    Place   string `xml:"place,attr"`
    Sources string `xml:"sources,attr"`
    Text    string `xml:",chardata"`
}
type Source struct {
    Id    string `xml:"id,attr"`
    Title string `xml:"title,attr"`
    Text  string `xml:",chardata"`
}
type Name struct {
    Given   string `xml:"given,attr"`
    Surname string `xml:"surname,attr"`
}
type Link struct {
    Title string  `xml:"title,attr"`
}
type Person struct {
    Name             Name     `xml:"person>name"`
    Gender           string   `xml:"person>gender"`
    Event_fact       []Event  `xml:"person>event_fact"`
    Source_citation  []Source `xml:"person>source_citation"`
    Child_of_family  Link     `xml:"person>child_of_family"`
    Spouse_of_family []Link   `xml:"person>spouse_of_family"`
    Text             string   `xml:",chardata"`
    namespace        string
    title            string
    country          string
}
type Family struct {
    Source_citation  []Source `xml:"family>source_citation"`
    Child            []Link   `xml:"family>child"`
    Husband          Link     `xml:"family>husband"`
    Wife             Link     `xml:"family>wife"`
    Text             string   `xml:",chardata"`
    namespace        string
    title            string
}
func persondata(str string) (factsxml Person) {
    str = strings.Join([]string{"<text>",str,"</text>"}, "")
    err := xml.Unmarshal([]byte(str), &factsxml)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: xml person facts parse %v %s\n", err, str)
		return
	}
    return factsxml
}

func familydata(str string) (factsxml Family) {
    // re-wrap it in text tags so the parser can handle it
    str = strings.Join([]string{"<text>",str,"</text>"}, "")
    //fmt.Fprintf(os.Stderr, "\nFAMILY PAGE TEXT:\n%s\n", str)
    err := xml.Unmarshal([]byte(str), &factsxml)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: xml facts parse: %v: %s\n", err, str)
		return
	}
    return factsxml
}

// what country is this person from?
func getcountry(personxml Person) string {
    cncnt := map[string]int {} // count how many times we see each country
    for i := range personxml.Event_fact {
        p := personxml.Event_fact[i].Place
        p = regexp.MustCompile("[|].+").ReplaceAllString(p, "")
        p = regexp.MustCompile("^.+,").ReplaceAllString(p, "")
        fmt.Printf("Event %d %s => %s\n", i, personxml.Event_fact[i].Place, p)
        cncnt[country.Country2code("USA")]++
    }
    cn := ""
    for i := range cncnt {
        if (len(cn) == 0 || cncnt[i] > cncnt[cn]) { cn = i }
    }
    return cn
}

//------------------------------------------------------------------------
func main() {
    // open the two files
	indexfile, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer indexfile.Close()

	pagefile, err := os.Open(os.Args[2])
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer pagefile.Close()


    index := []PageIndex{}
    page2index := map[string]int {}

    // load in the index file
    reader := csv.NewReader(indexfile)
    for {
        record, err := reader.Read()
        if err == io.EOF {
            break
        } else if err != nil {
            fmt.Println("Error: bad record ",err)
            continue
        }
        page2index[record[0]] = len(index);
        var rec PageIndex
        rec.title = record[0]
        rec.start,_ = strconv.ParseInt(record[1], 10, 64)
        rec.end,_ = strconv.ParseInt(record[2], 10, 64)
        index = append(index, rec)
        //fmt.Println("add record", index[0].title, index[0].start, index[0].end);
    }
    fmt.Println("read",len(index), "records")

    // no go through each record and fill in more details from each page
    for i := range index {
        //fmt.Println("Page", i, "title", index[i].title, index[i].start, index[i].end);
        buf := make([]byte, index[i].end-index[i].start)
        pagefile.ReadAt(buf, index[i].start)
        var p Page
        err := xml.Unmarshal(buf, &p)
        if err != nil {
            fmt.Printf("error parsing xml for %s: %v", index[i].title, err)
            continue
        }

        namespace, title := splittitle(p.Title)
        if (regexp.MustCompile("(?i)^#REDIRECT").MatchString(p.Text)) {
            continue // skip over redirect pages
        }
        switch namespace {
        case "Person":
            //fmt.Println("PAGE",p.Title,"NAMESPACE",namespace,"TITLE",title)
            personxml := persondata(p.Text)
            personxml.namespace = namespace
            personxml.title = title
            //  country
            personxml.country = getcountry(personxml)
            fmt.Printf("%q\n", personxml)
        case "Family":
            //fmt.Println("PAGE",p.Title,"NAMESPACE",namespace,"TITLE",title)
            familyxml := familydata(p.Text)
            familyxml.namespace = namespace
            familyxml.title = title
            fmt.Printf("%q\n", familyxml)
        }
    }

    // TBD traverse trees to find tree sizes
    // TBD quality scores

    // print the expanded csv file
    csvout := csv.NewWriter(os.Stdout)
    for i := range index {
        csvout.Write([]string{
            strconv.Itoa(i),
            index[i].title,
//            namespace,
        })
    }
    csvout.Flush()
}
