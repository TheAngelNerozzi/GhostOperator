package main

import (
	"encoding/binary"
	"io/ioutil"
	"os"
)

func main() {
	pngData, err := ioutil.ReadFile("logo.png")
	if err != nil {
		panic(err)
	}

	// 1. ICO Header (6 bytes)
	// Reserved (0,0), Type (1,0 ico), Number of images (1,0)
	header := []byte{0, 0, 1, 0, 1, 0}

	// 2. ICO Directory Entry (16 bytes)
	// Width (0=256), Height (0=256), Colors (0), Reserved (0), Planes (1,0), BPP (32,0),
	// Size (4 bytes), Offset (4 bytes)
	dir := make([]byte, 16)
	dir[0] = 0 // Width 256
	dir[1] = 0 // Height 256
	dir[2] = 0 // Colors
	dir[3] = 0 // Reserved
	binary.LittleEndian.PutUint16(dir[4:], 1)  // Planes
	binary.LittleEndian.PutUint16(dir[6:], 32) // BPP
	binary.LittleEndian.PutUint32(dir[8:], uint32(len(pngData)))
	binary.LittleEndian.PutUint32(dir[12:], 22) // Header (6) + Dir (16)

	// 3. Write File
	f, _ := os.Create("ghost.ico")
	defer f.Close()
	f.Write(header)
	f.Write(dir)
	f.Write(pngData)
}
