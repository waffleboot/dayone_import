package main

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

const dir = "import_journal"
const dst = dir + "/Journal.json"
const src = "Journal_dayone"

type Entry struct {
	Starred             bool    `json:"starred"`
	EditingTime         int     `json:"editingTime"`
	CreationDeviceModel string  `json:"creationDeviceModel"`
	CreationDate        string  `json:"creationDate"`
	Uuid                string  `json:"uuid"`
	CreationOSName      string  `json:"creationOSName"`
	CreationDevice      string  `json:"creationDevice"`
	ModifiedDate        string  `json:"modifiedDate"`
	IsPinned            bool    `json:"isPinned"`
	IsAllDay            bool    `json:"isAllDay"`
	TimeZone            string  `json:"timeZone"`
	CreationDeviceType  string  `json:"creationDeviceType"`
	Duration            int     `json:"duration"`
	Text                string  `json:"text"`
	CreationOSVersion   string  `json:"creationOSVersion"`
	Photos              []Photo `json:"photos"`
}

type Photo struct {
	FileSize          int64  `json:"fileSize"`
	OrderInEntry      int    `json:"orderInEntry"`
	CreationDevice    string `json:"creationDevice"`
	Duration          int    `json:"duration"`
	Favorite          bool   `json:"favorite"`
	Type              string `json:"type"`
	Identifier        string `json:"identifier"`
	Date              string `json:"date"`
	ExposureBiasValue int    `json:"exposureBiasValue"`
	Height            int    `json:"height"`
	Width             int    `json:"width"`
	Md5               string `json:"md5"`
	IsSketch          bool   `json:"isSketch"`
}

type Data struct {
	Metadata struct {
		Version string `json:"version"`
	} `json:"metadata"`
	Entries []Entry `json:"entries"`
}

func main() {
	err := run()
	if err != nil {
		fmt.Println(err)
	}
}

func run() (err error) {
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		errClose := f.Close()
		switch {
		case errClose != nil && err != nil:
			err = errors.Join(err, errClose)
		case errClose != nil:
			err = errClose
		}
	}()

	d, err := build()
	if err != nil {
		return err
	}

	enc := json.NewEncoder(f)
	enc.SetIndent(" ", " ")

	return enc.Encode(d)
}

func build() (Data, error) {
	entries, err := collect()
	if err != nil {
		return Data{}, err
	}

	var d Data
	d.Metadata.Version = "1.0"
	d.Entries = entries
	return d, nil
}

func collect() ([]Entry, error) {
	var files []string
	err := filepath.Walk(src+"/entries", func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	entries := make([]Entry, 0, len(files))
	for _, file := range files {
		// if file != src+"/entries/0FE83A0ADF0740E4B62DD7843D25BC96.doentry" {
		// 	continue
		// }
		entry, err := read(file)
		if err != nil {
			fmt.Println(file, err)
			continue
			// return nil, err
		}
		entries = append(entries, entry)
		// break
	}

	return entries, nil
}

func read(file string) (_ Entry, err error) {
	f, err := os.Open(file)
	if err != nil {
		return Entry{}, err
	}
	defer func() {
		errClose := f.Close()
		switch {
		case errClose != nil && err != nil:
			err = errors.Join(err, errClose)
		case errClose != nil:
			err = errClose
		}
	}()

	dec := xml.NewDecoder(f)

	var (
		wasKey bool
		wasVal bool
		key    string
		val    string
	)

	var entry Entry

	for {
		t, err := dec.Token()
		if errors.Is(io.EOF, err) {
			break
		}
		if err != nil {
			return Entry{}, err
		}
		switch e := t.(type) {
		case xml.StartElement:
			switch e.Name.Local {
			case "key":
				wasKey = true
			case "date", "string":
				wasVal = true
			case "false":
				entry.Starred = false
			case "true":
				entry.Starred = true
			}
		case xml.CharData:
			if wasKey {
				key, wasKey = string(e), false
			}
			if wasVal {
				val, wasVal = string(e), false
				switch key {
				case "Creation Date":
					// ts, err := time.Parse(time.RFC3339, val)
					// if err != nil {
					// 	return Entry{}, err
					// }
					// entry.ModifiedDate = ts
					entry.CreationDate = val
					entry.ModifiedDate = val
				case "Entry Text":
					entry.Text = val
				case "UUID":
					entry.Uuid = val
				}
			}
		}
	}

	entry.CreationDeviceModel = "Mac14,7"
	entry.CreationOSName = "macOS"
	entry.CreationDevice = "Андрей’s MacBook Pro"
	entry.TimeZone = "Europe\\/Moscow"
	entry.CreationDeviceType = "MacBook Pro"
	entry.CreationOSVersion = "13.5.1"

	fi, err := os.Stat(src + "/photos/" + entry.Uuid + ".jpg")
	if errors.Is(err, os.ErrNotExist) {
		return entry, nil
	}
	if err != nil {
		return Entry{}, err
	}

	entry.Photos = []Photo{{
		FileSize:       fi.Size(),
		CreationDevice: "Андрей’s MacBook Pro",
		Type:           "jpeg",
		Identifier:     entry.Uuid,
		Date:           entry.CreationDate,
		Height:         350,
		Width:          480,
		Md5:            entry.Uuid,
	}}

	return entry, nil
}
