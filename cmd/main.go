package main

import (
	"log"

	"fbnoi.com/template"
)

func main() {

	s := template.NewSourceFile("var/template/test.html")
	stream, err := template.Tokenize(s)
	if err != nil {
		log.Fatalln(err)
	}
	doc, err := template.Assemble(stream)
	log.Println(doc)
	// for !stream.IsEOF() {
	// 	token, err := stream.Next()
	// 	if err != nil {
	// 		log.Fatalln(err)
	// 	}
	// 	log.Println(token)
	// }
}
