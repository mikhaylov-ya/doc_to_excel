package main

type Journal struct {
	volume int
	number int
}

type Reference struct {
	authors []string
	year    int
	title   string
	meta    string
	doi     string
}

type Article struct {
	title        string
	abstract     string
	pages        string
	keywords     string
	authors      string
	affiliations string
	references   []string
	doi          string
}
