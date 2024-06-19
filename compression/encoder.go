package compression

import (
	"fmt"
	"io"
	"os"
	"sort"
)

/* Encoder use Huffman alogrism */

type Frequencies []CharFrequency

func (f Frequencies) Sort() {
	if len(f) > 0 {
		sort.Slice(f, func(i, j int) bool {
			return f[i].Frequency < f[j].Frequency
		})
	}
}

type CharFrequencyType int

var CharFreq, AlogFreq CharFrequencyType = 1, 2

type CharFrequency struct {
	Frequency int
	Type      CharFrequencyType
	PrefCode  []int
	Char      string
	N         *Node
}

type Encoder struct {
	FileContent, OutputFile string
	Frequencies             Frequencies
}

type Node struct {
	Value       int
	Char        string
	Left, Right *Node
}

func NewEncoder(inputFile io.Reader, outputFile string) (*Encoder, error) {
	//Get the file content
	FileContent, err := io.ReadAll(inputFile)
	if err != nil {
		return nil, err
	}

	e := &Encoder{
		Frequencies: make(Frequencies, 0),
		FileContent: string(FileContent),
		OutputFile:  outputFile,
	}

	//count each char Frequency
	e.countFrequency()
	if err := e.GeneratePrefCode(); err != nil {
		return nil, err
	}

	return e, nil
}

func (e *Encoder) countFrequency() {
	if len(e.FileContent) == 0 {
		return
	}

	frequencyMap := make(map[rune]int)

	for _, char := range e.FileContent {
		frequencyMap[char]++
	}

	// Convert the map to the Frequencies slice
	for char, count := range frequencyMap {
		e.Frequencies = append(e.Frequencies, CharFrequency{
			Char:      string(char),
			Frequency: count,
			Type:      CharFreq,
		})
	}
}

func (e *Encoder) BuildTree() *Node {
	var (
		n, leftn, rightn       *Node
		firstIdex, secondIndex = 0, 1
	)

	//sort the source frequencies befory copy
	e.Frequencies.Sort()
	// Create a new slice to work with for the tree construction
	Frequencies := make(Frequencies, len(e.Frequencies))
	if n := copy(Frequencies, e.Frequencies); n != len(Frequencies) {
		//copy fail
		fmt.Println("fail to copy Frequencies slice")
		return nil
	}

	//ensure the frequency slice is sorted
	Frequencies.Sort()

	for len(Frequencies) > 1 {
		//left node
		if Frequencies[firstIdex].Type == CharFreq {
			leftn = &Node{Char: Frequencies[firstIdex].Char, Value: Frequencies[firstIdex].Frequency}
		} else {
			leftn = Frequencies[firstIdex].N
		}

		//right node
		if Frequencies[secondIndex].Type == CharFreq {
			rightn = &Node{Char: Frequencies[secondIndex].Char, Value: Frequencies[secondIndex].Frequency}
		} else {
			rightn = Frequencies[secondIndex].N
		}

		value := Frequencies[firstIdex].Frequency + Frequencies[secondIndex].Frequency
		n = &Node{
			Value: value,
			Left:  leftn,
			Right: rightn,
		}

		//remove first and secound from the slice
		Frequencies = Frequencies[2:]
		//add the new node value
		Frequencies = append(Frequencies, CharFrequency{
			Frequency: value,
			Type:      AlogFreq,
			N:         n,
		})
		//fmt.Println(Frequencies)
		//resort Frequencies slice
		Frequencies.Sort()
	}

	if len(Frequencies) == 1 {
		return Frequencies[0].N
	}

	return nil
}

func (e *Encoder) GeneratePrefCode() error {
	node := e.BuildTree()
	if node == nil {
		return fmt.Errorf("error building the tree/heap")
	}

	for i, f := range e.Frequencies {
		//generate the char code
		code, _ := TreeLoop(node, f.Char)
		e.Frequencies[i].PrefCode = code
		//break
	}
	return nil
}

func (e *Encoder) EncodeData() ([]byte, error) {
	// Create a map to store each character's Huffman prefix code
	prefixCodeMap := make(map[string][]int)
	for _, f := range e.Frequencies {
		prefixCodeMap[f.Char] = f.PrefCode
	}

	var encodedData []byte // This will store the final encoded data as bytes
	var currentByte byte   // This byte will collect up to 8 bits of Huffman code
	var bitCount uint8     // Counter to track how many bits are in currentByte

	for _, char := range e.FileContent {
		if code, found := prefixCodeMap[string(char)]; found {
			// For each character, get its Huffman code and process each bit
			for _, bit := range code {
				if bitCount == 8 {
					// When currentByte is full (8 bits), append it to encodedData
					encodedData = append(encodedData, currentByte)
					currentByte = 0 // Reset currentByte for the next set of bits
					bitCount = 0    // Reset bitCount
				}
				// Shift currentByte left by 1 to make space for the new bit and add the bit
				currentByte = (currentByte << 1) | byte(bit)
				bitCount++
			}
		}
	}

	// Append any remaining bits in currentByte to encodedData
	if bitCount > 0 {
		// Pad with zeros to fill the remaining bits to form a full byte
		for bitCount < 8 {
			currentByte <<= 1 // Shift left to add zeroes to the least significant bits
			bitCount++
		}
		encodedData = append(encodedData, currentByte)
	}

	return encodedData, nil
}

func b(n []int) []byte {
	bt := []byte{}
	for _, v := range n {
		bt = append(bt, byte(v))
	}
	return bt
}

func (e *Encoder) WriteHeaderAndEncodedData() error {
	if len(e.OutputFile) == 0 {
		return fmt.Errorf("output file param is missing")
	}

	f, err := os.OpenFile(e.OutputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	f.Write([]byte("Header:\n"))
	for _, fr := range e.Frequencies {
		newlineb := []byte("\n")
		strtowrite := []byte(fmt.Sprintf("%v:", fr.Char))
		prefcodeb := b(fr.PrefCode)
		strtowrite = append(strtowrite, prefcodeb...)
		strtowrite = append(strtowrite, newlineb...)
		f.Write(strtowrite)
	}

	f.Write([]byte("\nEncoded Data:\n"))
	// Encode the data
	encodedData, err := e.EncodeData()
	if err != nil {
		return err
	}

	_, err = f.Write(encodedData)
	return err
}

func TreeLoop(node *Node, char string) ([]int, error) {
	if node == nil {
		return nil, fmt.Errorf("tree is nil")
	}

	var prefcode []int
	var checkLeaf func(n *Node, nPrevCode []int) bool

	checkLeaf = func(n *Node, nPrevCode []int) bool {
		if n == nil {
			return false
		}
		if n.Char == char {
			// Found the character, save the code
			prefcode = nPrevCode
			return true
		}

		// Check left subtree with '0' appended to the current code
		if checkLeaf(n.Left, append(nPrevCode, 0)) {
			return true
		}

		// Check right subtree with '1' appended to the current code
		if checkLeaf(n.Right, append(nPrevCode, 1)) {
			return true
		}

		return false
	}

	if checkLeaf(node, prefcode) {
		return prefcode, nil
	}
	return nil, fmt.Errorf("character not found in the tree")
}
