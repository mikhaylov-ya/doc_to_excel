package main

type Journal struct {
	volume int
	number int
}

type Article struct {
	abstract     string
	keywords     []string
	authors      []string
	affiliations []string
	references   Journal
	doi          string
	pubdate      string
}

type Reference struct {
	authors []string
	year    int
	title   string
	meta    string
	doi     string
}
