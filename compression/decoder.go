package compression

import (
	"fmt"
	"io"
	"strings"
)

type Decoder struct {
	FileContent []byte            //encoded
	Prefix      map[string]string // ["1111110000"] = "a"
}

func NewDecoder(inputFile io.Reader) (*Decoder, error) {
	FileContent, err := io.ReadAll(inputFile)
	if err != nil {
		return nil, err
	}

	e := &Decoder{
		Prefix:      make(map[string]string, 0),
		FileContent: FileContent,
	}

	e.FileContent = FileContent

	return e, nil
}

func (d *Decoder) Decode(outputFile io.Writer) error {
	if err := d.DecodeHeader(); err != nil {
		return err
	}

	if err := d.DecodeData(outputFile); err != nil {
		return err
	}
	//decode the body
	return nil
}

func (d *Decoder) DecodeHeader() error {
	fcs := string(d.FileContent)
	headerIndex := strings.Index(fcs, "Encoded Data:")
	encodedHeader := strings.Trim(fcs[:headerIndex], "Header:\n")
	encodedHeaderCharLine := strings.Split(encodedHeader, "\n")

	for _, line := range encodedHeaderCharLine {
		s := strings.Trim(line, " ")
		sl := strings.Split(s, ":")
		if len(sl) == 2 {
			char := sl[0]
			bcode := []byte(sl[1]) //code in bytes

			var code string
			for _, num := range bcode {
				//every byte represent an int
				//todo: make every byte represent num 0,1 (in encoder)
				codeint := int(num)
				if codeint == 0 {
					code += "0"
				} else if codeint == 1 {
					code += "1"
				}
			}
			d.Prefix[code] = char
		}
	}
	return nil
}

func readBit(data []byte, bitIndex int) int {
	byteIndex := bitIndex / 8
	bitOffset := 7 - (bitIndex % 8) // Start from the most significant bit
	return (int(data[byteIndex]) >> bitOffset) & 1
}

func (d *Decoder) DecodeData(outputFile io.Writer) error {
	fcs := string(d.FileContent)
	dataIndex := strings.Index(fcs, "\nEncoded Data:\n") + len("\nEncoded Data:\n")
	encodedData := []byte(d.FileContent[dataIndex:5000]) //bytes

	var currentPrefix strings.Builder
	var decodedData strings.Builder
	//1 byte = 8bits = 01101010
	bitIndex := 0
	totalBits := len(encodedData) * 8
	for bitIndex < totalBits {
		// Read the next bit
		bit := readBit(encodedData, bitIndex)
		currentPrefix.WriteString(fmt.Sprintf("%d", bit))
		// Check if the current prefix matches any known prefix code
		if char, exists := d.Prefix[currentPrefix.String()]; exists {
			decodedData.WriteString(char)
			currentPrefix.Reset() // Reset the prefix for the next character
		}
		bitIndex++
	}

	// Write the decoded data to the output file
	_, err := outputFile.Write([]byte(decodedData.String()))
	return err
}
