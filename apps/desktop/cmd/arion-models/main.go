// Command arion-models inspects upstream archives for catalog review and
// imports already-downloaded archives into a model store without network.
package main

import (
	"archive/tar"
	"compress/bzip2"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"arion.local/desktop/infrastructure/provision"
	"arion.local/models"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "inspect":
		flags := flag.NewFlagSet("inspect", flag.ExitOnError)
		archive := flags.String("archive", "", "upstream .tar.bz2")
		_ = flags.Parse(os.Args[2:])
		check(inspect(*archive))
	case "import":
		flags := flag.NewFlagSet("import", flag.ExitOnError)
		archive := flags.String("archive", "", "downloaded .tar.bz2")
		artifact := flags.String("artifact", "", "catalog artifact id")
		root := flags.String("root", "", "model store root")
		_ = flags.Parse(os.Args[2:])
		check(importArchive(*archive, *artifact, *root))
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: arion-models inspect -archive file | import -archive file -artifact id -root dir")
	os.Exit(2)
}

func check(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// inspect prints the archive digest and every regular member's size and
// SHA-256, the values a catalog entry pins.
func inspect(name string) error {
	file, err := os.Open(name)
	if err != nil {
		return err
	}
	defer file.Close()
	archiveHash := sha256.New()
	reader := tar.NewReader(bzip2.NewReader(io.TeeReader(file, archiveHash)))
	members := map[string]models.File{}
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		h := sha256.New()
		n, err := io.Copy(h, reader)
		if err != nil {
			return err
		}
		members[header.Name] = models.File{Member: header.Name, SHA256: hex.EncodeToString(h.Sum(nil)), SizeBytes: n}
	}
	if _, err := io.Copy(archiveHash, file); err != nil {
		return err
	}
	info, _ := file.Stat()
	out, _ := json.MarshalIndent(map[string]any{"sha256": hex.EncodeToString(archiveHash.Sum(nil)), "sizeBytes": info.Size(), "members": members}, "", "  ")
	fmt.Println(string(out))
	return nil
}

func importArchive(archive, id, root string) error {
	catalog, err := models.Load()
	if err != nil {
		return err
	}
	artifact, ok := catalog.Artifact(id)
	if !ok {
		return fmt.Errorf("unknown artifact %s", id)
	}
	if err := os.MkdirAll(filepath.Join(root, ".downloads"), 0o700); err != nil {
		return err
	}
	staged := filepath.Join(root, ".downloads", artifact.Source.SHA256+".tar.bz2")
	if err := os.Link(archive, staged); err != nil && !os.IsExist(err) {
		return err
	}
	store := provision.NewStore(root)
	dir, err := store.Ensure(context.Background(), artifact, func(string, provision.Phase, int64, int64) {})
	if err != nil {
		return err
	}
	fmt.Println(dir)
	return nil
}
