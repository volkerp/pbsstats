package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

const (
	FidxHeaderSize   = 4096
	MagicSize        = 8
	UUIDSize         = 16
	CsumSize         = 32
	FidxReservedSize = 4016
	DidxReservedSize = 4032
	FidxDigestSize   = 32
)

var FidxMagic = [MagicSize]byte{47, 127, 65, 237, 145, 253, 15, 205}
var DidxMagic = [MagicSize]byte{28, 145, 78, 165, 25, 186, 179, 205}

type FidxHeader struct {
	Magic     [MagicSize]byte
	UUID      [UUIDSize]byte
	Ctime     int64
	IndexCsum [CsumSize]byte
	Size      uint64
	ChunkSize uint64
	Reserved  [FidxReservedSize]byte
}

type Digest [FidxDigestSize]byte

type Fidx struct {
	Header  FidxHeader
	Digests []Digest
}

type DidxHeader struct {
	Magic     [MagicSize]byte
	UUID      [UUIDSize]byte
	Ctime     int64
	IndexCsum [CsumSize]byte
	Reserved  [DidxReservedSize]byte
}

type DidxEntry struct {
	Offset uint64
	Digest Digest
}

type Didx struct {
	Header  DidxHeader
	Digests []DidxEntry
}

func readFidxHeader(r io.Reader) (*FidxHeader, error) {
	var hdr FidxHeader
	err := binary.Read(r, binary.LittleEndian, &hdr)
	if err != nil {
		return nil, err
	}
	if hdr.Magic != FidxMagic {
		return nil, fmt.Errorf("invalid magic: %v", hdr.Magic)
	}
	return &hdr, nil
}

func readDidxHeader(r io.Reader) (*DidxHeader, error) {
	var hdr DidxHeader
	err := binary.Read(r, binary.LittleEndian, &hdr)
	if err != nil {
		return nil, err
	}
	if hdr.Magic != DidxMagic {
		return nil, fmt.Errorf("invalid magic: %v", hdr.Magic)
	}
	return &hdr, nil
}

func readFidxDigests(r io.Reader) ([]Digest, error) {
	var digests []Digest
	for {
		digest := make([]byte, FidxDigestSize)
		n, err := io.ReadFull(r, digest)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if n != FidxDigestSize {
			break
		}
		digests = append(digests, Digest(digest))
	}
	return digests, nil
}

func readDidxEntries(r io.Reader) ([]DidxEntry, error) {
	entries := make([]DidxEntry, 0)
	for {
		var entry DidxEntry
		err := binary.Read(r, binary.LittleEndian, &entry)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func readFidxFile(filename string) (*Fidx, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	hdr, err := readFidxHeader(f)
	if err != nil {
		return nil, err
	}

	digests, err := readFidxDigests(f)
	if err != nil {
		return nil, err
	}

	return &Fidx{*hdr, digests}, nil
}

func readDidxFile(filename string) (*Didx, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	hdr, err := readDidxHeader(f)
	if err != nil {
		return nil, err
	}

	entries, err := readDidxEntries(f)
	if err != nil {
		return nil, err
	}
	return &Didx{*hdr, entries}, nil
}

// func main() {
// 	if len(os.Args) < 2 {
// 		fmt.Println("Usage: fidx_reader <file.fidx>")
// 		os.Exit(1)
// 	}

// 	filename := os.Args[1]
// 	fidx, err := readFidxFile(filename)
// 	if err != nil {
// 		fmt.Printf("Error reading fidx file: %v\n", err)
// 		os.Exit(1)
// 	}

// 	fmt.Printf("UUID: %x\n", fidx.Header.UUID)
// 	fmt.Printf("Ctime: %s\n", time.Unix(fidx.Header.Ctime, 0).Format(time.RFC3339))
// 	fmt.Printf("Size: %d\n", fidx.Header.Size)
// 	fmt.Printf("ChunkSize: %d\n", fidx.Header.ChunkSize)
// 	fmt.Printf("Num Chunks: %d\n", len(fidx.Digests))
// 	if len(fidx.Digests) > 0 {
// 		fmt.Printf("First digest: %s\n", hex.EncodeToString(fidx.Digests[0][:]))
// 	}
// 	fmt.Printf("Last digest: %s\n", hex.EncodeToString(fidx.Digests[len(fidx.Digests)-1][:]))

// 	// // Optional: verify index checksum
// 	// // Seek to start of digests
// 	// f.Seek(FidxHeaderSize, io.SeekStart)
// 	// digestData := make([]byte, 0)
// 	// for {
// 	// 	buf := make([]byte, 4096)
// 	// 	n, err := f.Read(buf)
// 	// 	if n > 0 {
// 	// 		digestData = append(digestData, buf[:n]...)
// 	// 	}
// 	// 	if err == io.EOF {
// 	// 		break
// 	// 	}
// 	// 	if err != nil {
// 	// 		panic(err)
// 	// 	}
// 	// }
// 	// sum := sha256.Sum256(digestData)
// 	// if sum != hdr.IndexCsum {
// 	// 	fmt.Println("WARNING: index checksum mismatch")
// 	// } else {
// 	// 	fmt.Println("Index checksum OK")
// 	// }
// }
