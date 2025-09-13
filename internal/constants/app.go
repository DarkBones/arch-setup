package constants

import "fmt"

type appStrings struct {
	Name  string
	Title string
}

const name = "Bootstrap Any System"

var App = &appStrings{
	Name:  name,
	Title: fmt.Sprintf("BAS - %s", name),
}
