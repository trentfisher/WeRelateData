package main

// generate a summary of page counts by namespace

import (
    "encoding/csv"
    "os"
    "fmt"
    "io"
    "regexp"
)

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

func main() {
    indexfile, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error opening file:", err)
		return
	}
	defer indexfile.Close()

    counts := map[string]int {}
    reader := csv.NewReader(indexfile)
    for {
        record, err := reader.Read()
        if err == io.EOF {
            break
        } else if err != nil {
            fmt.Fprintln(os.Stderr, "Error: bad record ",err)
            continue
        }
        namespace, _ := splittitle(record[1])
        counts[namespace]++
        counts["TOTAL"]++
    }

    for i := range counts {
        fmt.Printf("%s,%d\n", i, counts[i]);
    }
}
