package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
)

type entry struct {
	isDir  bool
	isLast bool
	path   string
	prefix string
	info   os.FileInfo
}

func readDirEntries(root string, prefix string, withFiles bool) (entries []*entry, err error) {
	infos, err := ioutil.ReadDir(root)

	if err != nil {
		return nil, err
	}

	for _, info := range infos {
		isDir := info.IsDir()

		if !(isDir || withFiles) {
			continue
		}

		entries = append(entries, &entry{
			info:   info,
			isDir:  isDir,
			prefix: prefix,
			path:   path.Join(root, info.Name()),
		})
	}

	if ln := len(entries); ln != 0 {
		entries[ln-1].isLast = true
	}

	return
}

func printEntry(out io.Writer, entry *entry) {
	fmt.Fprint(out, entry.prefix)

	if entry.isLast {
		fmt.Fprint(out, "└───")
	} else {
		fmt.Fprint(out, "├───")
	}

	fmt.Fprint(out, entry.info.Name())

	if entry.isDir {
		// skip
	} else if size := entry.info.Size(); size != 0 {
		fmt.Fprintf(out, " (%db)", size)
	} else {
		fmt.Fprint(out, " (empty)")
	}

	fmt.Fprint(out, "\n")
}

func dirTree(out io.Writer, root string, printFiles bool) error {
	var prefix string
	entries, err := readDirEntries(root, "", printFiles)

	if err != nil {
		return err
	}

	for len(entries) != 0 {
		entry, rest := entries[0], entries[1:]

		if entry.isDir {
			if entry.isLast {
				prefix = "\t"
			} else {
				prefix = "│\t"
			}

			childs, err := readDirEntries(entry.path, entry.prefix+prefix, printFiles)

			if err != nil {
				return err
			}

			entries = append(childs, rest...)
		} else {
			entries = rest
		}

		printEntry(out, entry)
	}

	return nil
}

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}
