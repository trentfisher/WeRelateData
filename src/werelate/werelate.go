package werelate

// utility functions

import "regexp"

func SplitTitle(str string) (namespace string, title string) {
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
