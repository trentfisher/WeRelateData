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
//    "strings"
    "country"
)

// the index we build up of the pages
type PageIndex struct {
    title string
    start int64
    end   int64
    country   string
    tree      int
    score     int
}

// record for data from xml
type Page struct {
	Title string `xml:"title"`
	Text string `xml:"revision>text"`
}
// fetch a range of bytes from the given file
func fetchrange(in *os.File, start, end int64) string {
    //fmt.Println("fetching byte range", start, "to", end);
    buf := make([]byte, end-start)
    in.ReadAt(buf, start)
    return string(buf[:]);
}

func fetchxmlfrag(pagefile *os.File, start int64, end int64) (p Page) {
    buf := make([]byte, end-start)
    pagefile.ReadAt(buf, start)
    err := xml.Unmarshal(buf, &p)
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: parsing xml for: %v", err)
    }
    return p
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
    Name             Name     `xml:"name"`
    Gender           string   `xml:"gender"`
    Event_fact       []Event  `xml:"event_fact"`
    Source_citation  []Source `xml:"source_citation"`
    Child_of_family  Link     `xml:"child_of_family"`
    Spouse_of_family []Link   `xml:"spouse_of_family"`
    Text             string
    namespace        string
    title            string
    country          string
}
type Family struct {
    Source_citation  []Source `xml:"source_citation"`
    Child            []Link   `xml:"child"`
    Husband          Link     `xml:"husband"`
    Wife             Link     `xml:"wife"`
    Text             string
    namespace        string
    title            string
}
func persondata(str string) (factsxml Person) {
    facts, text := splitfacts(str)
    err := xml.Unmarshal([]byte(facts), &factsxml)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: xml person facts parse %v %s\n", err, str)
		return
	}
    factsxml.Text = text
    return factsxml
}

// split the wiki page contents into the xml facts section and the page
// text which doesn't seem to be xml
func splitfacts(str string) (facts, text string) {
    m := regexp.MustCompile("(?is)^(.*<(/family|/person|family/|person/)>)(.*)").
        FindStringSubmatch(str)
    if (len(m) != 4) {
        fmt.Fprintf(os.Stderr, "Error: split facts failed for %s\n", str)
        facts = ""
        text = str
    } else {
        facts = m[1]
        text = m[3]
    }
    // trim out some gunk and white space from the text
    text = regexp.MustCompile("<show_sources_images_notes/>").
        ReplaceAllString(text, "")
    text = regexp.MustCompile("(?s)[ \t\r\n]+$").ReplaceAllString(text, "")
    text = regexp.MustCompile("(?s)^[ \t\r\n]+").ReplaceAllString(text, "")
    return facts, text
}
func familydata(str string) (factsxml Family) {
    facts, text := splitfacts(str)
    err := xml.Unmarshal([]byte(facts), &factsxml)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: xml facts parse: %v: %s\n", err, str)
		return
	}
    factsxml.Text = text
    return factsxml
}

// what country is this person from?
func getcountry(personxml Person) string {
    cncnt := map[string]int {} // count how many times we see each country
    birthcn := ""
    for i := range personxml.Event_fact {
        p := personxml.Event_fact[i].Place
        p = regexp.MustCompile("[|].+").ReplaceAllString(p, "")
        p = regexp.MustCompile("^.+, *").ReplaceAllString(p, "")
        p = regexp.MustCompile(" +$").ReplaceAllString(p, "")
        c := country.Country2code(p)
        fmt.Fprintf(os.Stderr, "Event %d %s => %s => %s\n", i, personxml.Event_fact[i].Place, p, c)
        cncnt[c]++
        if (personxml.Event_fact[i].Type == "Birth" ||
            personxml.Event_fact[i].Type == "Christening") {
            birthcn = c
        }
    }

    // we prefer birth data
    if (len(birthcn) > 0) { return birthcn }

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
		fmt.Fprintln(os.Stderr, "Error opening file:", err)
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
            fmt.Fprintln(os.Stderr, "Error: bad record ",err)
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
    fmt.Fprintln(os.Stderr, "Loaded",len(index), "index records")

    // no go through each record and fill in more details from each page
    for i := range index {
        fmt.Fprintln(os.Stderr, "Page", i, "title", index[i].title, index[i].start, index[i].end);
        p := fetchxmlfrag(pagefile, index[i].start, index[i].end)

        namespace, title := splittitle(p.Title)
/*        if (regexp.MustCompile("(?i)^#REDIRECT").MatchString(p.Text)) {
            continue // skip over redirect pages
        }
*/
        switch namespace {
        case "Person":
            //fmt.Println("PAGE",p.Title,"NAMESPACE",namespace,"TITLE",title)
            personxml := persondata(p.Text)
            personxml.namespace = namespace
            personxml.title = title
            //  country
            index[i].country = getcountry(personxml)
            //fmt.Printf("PERSON %q\n", personxml)
        case "Family":
            //fmt.Println("PAGE",p.Title,"NAMESPACE",namespace,"TITLE",title)
            familyxml := familydata(p.Text)
            familyxml.namespace = namespace
            familyxml.title = title
            //fmt.Printf("FAMILY %q\n", familyxml)
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
            index[i].country,
            strconv.Itoa(index[i].tree),
            strconv.Itoa(index[i].score),
        })
    }
    csvout.Flush()
}
