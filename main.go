package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/abdoroot/com/compression"
)

func main() {
	var inputFile, outputFile string
	inputFile = "135-0.txt"

	flag.StringVar(&outputFile, "outputFile", "output.txt", "output file string")
	flag.Parse()

	file, err := os.Open(inputFile)
	if err != nil {
		fmt.Println("error opining file", inputFile)
		return
	}
	defer file.Close()

	Encoder, err := compression.NewEncoder(file, outputFile)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(Encoder.WriteHeaderAndEncodedData())

	////decode a huffman
	in, err := os.Open("output.txt")
	if err != nil {
		fmt.Println("error opining file", in)
		return
	}
	defer in.Close()

	ou, err := os.OpenFile("decoder.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("error opining file", ou)
		return
	}
	defer ou.Close()

	decoder, err := compression.NewDecoder(in)
	if err != nil {
		fmt.Println(err)
		return
	}
	decoder.Decode(ou)
}
