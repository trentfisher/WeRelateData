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
    XMLName          xml.Name
    Name             Name     `xml:"name"`
    Gender           string   `xml:"gender"`
    Source_citation  []Source `xml:"source_citation"`
    Child_of_family  Link     `xml:"child_of_family"`
    Spouse_of_family []Link   `xml:"spouse_of_family"`
    Text             string
}
type Family struct {
    XMLName         xml.Name
    Source_citation []Source
    Child           []Link `xml:"child_of_family"`
    Husband         Link `xml:"child_of_family"`
    Wife            Link `xml:"child_of_family"`
    Text            string
}

func persondata(str string) (factsxml Person) {
    m := strings.SplitAfter(str, "</person>")
    err := xml.Unmarshal([]byte(m[0]), &factsxml)
	if err != nil {
		fmt.Printf("error: xml facts parse %v", err)
		return
	}
    factsxml.Text = m[1]
    return factsxml
}

func familydata(str string) (factsxml Person) {
    m := strings.SplitAfter(str, "</family>")
    err := xml.Unmarshal([]byte(m[0]), &factsxml)
	if err != nil {
		fmt.Printf("error: xml family parse %v", err)
		return
	}
    factsxml.Text = m[1]
    return factsxml
}

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
        switch namespace {
        case "Person":
            fmt.Println("PAGE",p.Title,"NAMESPACE",namespace,"TITLE",title)
            personxml := persondata(p.Text)
            fmt.Printf("%q\n", personxml)
        case "Family":
            fmt.Println("PAGE",p.Title,"NAMESPACE",namespace,"TITLE",title)
            familyxml := familydata(p.Text)
            fmt.Printf("%q\n", familyxml)
        }
    }

    // TBD traverse trees to find tree sizes
    // TBD quality scores
    // TBD country

    // print the expanded csv file
    csvout := csv.NewWriter(os.Stdout)
    csvout.Write([]string{
        p.Title
        namespace
    })
    csvout.Flush()
}
