package models

import (
    "io"
    "io/ioutil"
    "encoding/json"
    "bytes"
    "fmt"
)

// A wrapper around io.Reader that allows marshalling the data in the io.Reader
// to JSON as a string. When MarshalReader.MarshalJSON is called, MarshalReader
// reads and permenently stores the original reader's data as a []byte, then provides
// access to it with a new bytes.Reader.
// The raw data array is coerced to string, then serialized via json.Marshal.
//
type MarshalReader struct {
    reader io.Reader
    data []byte
    MarshalAsString bool

}

// Extract the raw data, replacing the original io.Reader with a bytes.Reader pointed
// at internal storage
func (mr *MarshalReader) Data() ([]byte, error) {
    if mr.data == nil {
        var err error
        mr.data, err = ioutil.ReadAll(mr.reader)
        if err != nil { return nil, err }
        mr.reader = bytes.NewReader(mr.data)
    }

    return mr.data, nil
}

// Marshal the original data in the io.Reader to JSON
func (mr *MarshalReader) MarshalJSON() ([]byte, error) {
    data, err := mr.Data()
    if err != nil { return nil, err }

    // Serialize as string regular text 
    if mr.MarshalAsString {
        return json.Marshal(string(data))
    }

    // serialize byte data as Base64 json string
    return json.Marshal(data)
}

// implement io.Reader
func (mr *MarshalReader) Read(p []byte) (int, error) {
    return mr.reader.Read(p)
}

// Create a new MarshalReader from a regular reader
func NewMarshalReader(r io.Reader) *MarshalReader {
    return &MarshalReader{
        reader: r,
    }
}




// subtly duplicate a reader if you're worried
// use the `use_instead` reader for your dangerous operation
// if things go wrong, you have the `backup` to return to.
func backupReader(in_danger io.Reader) (*MarshalReader, *MarshalReader) {
    data, err := ioutil.ReadAll(in_danger)
    if err != nil {
        fmt.Printf("Error in backupReader creation: %v\n", err)
    }
    use_instead := NewMarshalReader(bytes.NewReader(data))
    backup := NewMarshalReader(bytes.NewReader(data))

    return use_instead, backup
}

var BackupReader = backupReader
