// Package archive manages the export tar.gz and provides helpers for modules
// to add and extract files without knowing the archive format.
package archive

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type archiveEntry struct {
	name string
	data []byte
}

// AddFile appends a single file from srcPath into the archive at archivePath,
// stored under the internal name entryName.
// The archive is created if it does not exist; existing entries are preserved
// (append mode via re-create — tar.gz does not support true append, so we
// rebuild the archive on each AddFile call; for large exports use AddDir).
func AddFile(archivePath, srcPath, entryName string) error {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", srcPath, err)
	}
	return addEntry(archivePath, entryName, data)
}

// AddDir recursively appends all files under srcDir into the archive,
// stored under prefix/<relative-path>.
// Collects all files first, then adds in batch to avoid O(n²) rebuilds.
func AddDir(archivePath, srcDir, prefix string) error {
	var newEntries []archiveEntry

	if err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Skip directories and symlinks; only add regular files
		if !info.Mode().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			// Skip unreadable files (broken symlinks, permission errors)
			return nil
		}
		newEntries = append(newEntries, archiveEntry{filepath.Join(prefix, rel), data})
		return nil
	}); err != nil {
		return err
	}

	// Batch-add all entries in single archive rebuild
	return addEntries(archivePath, newEntries)
}

// ExtractFile pulls a single entry out of the archive and writes it to a temp file.
// Caller is responsible for deleting the temp file.
func ExtractFile(archivePath, entryName string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", fmt.Errorf("open archive: %w", err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return "", fmt.Errorf("gzip: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("tar: %w", err)
		}
		if hdr.Name == entryName {
			tmp, err := os.CreateTemp("", "mac-onboarding-*-"+filepath.Base(entryName))
			if err != nil {
				return "", err
			}
			if _, err := io.Copy(tmp, tr); err != nil {
				tmp.Close()
				os.Remove(tmp.Name())
				return "", err
			}
			tmp.Close()
			return tmp.Name(), nil
		}
	}
	return "", fmt.Errorf("entry %q not found in archive", entryName)
}

// ExtractDir pulls all entries under prefix/ from the archive into dstDir.
func ExtractDir(archivePath, prefix, dstDir string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("gzip: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar: %w", err)
		}
		if !strings.HasPrefix(hdr.Name, prefix+"/") {
			continue
		}
		rel := strings.TrimPrefix(hdr.Name, prefix+"/")
		dst := filepath.Join(dstDir, rel)
		if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
			return err
		}
		out, err := os.Create(dst)
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			return err
		}
		out.Close()
	}
	return nil
}

// ListEntries returns all entry names in the archive.
func ListEntries(archivePath string) ([]string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gr.Close()

	var names []string
	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		names = append(names, hdr.Name)
	}
	return names, nil
}

// addEntries adds multiple entries in a single pass (batch mode for efficiency).
func addEntries(archivePath string, entries []archiveEntry) error {
	var existing []archiveEntry

	// Read existing entries once
	if f, err := os.Open(archivePath); err == nil {
		gr, err := gzip.NewReader(f)
		if err == nil {
			tr := tar.NewReader(gr)
			for {
				hdr, err := tr.Next()
				if err == io.EOF {
					break
				}
				if err != nil {
					break
				}
				// Check if this name is being replaced
				isReplaced := false
				for _, ne := range entries {
					if ne.name == hdr.Name {
						isReplaced = true
						break
					}
				}
				if isReplaced {
					continue
				}
				b, _ := io.ReadAll(tr)
				existing = append(existing, archiveEntry{hdr.Name, b})
			}
			gr.Close()
		}
		f.Close()
	}

	// Write new archive once with all entries
	out, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("create archive: %w", err)
	}
	defer out.Close()

	gw := gzip.NewWriter(out)
	tw := tar.NewWriter(gw)

	writeEntry := func(name string, d []byte) error {
		hdr := &tar.Header{
			Name: name,
			Mode: 0600,
			Size: int64(len(d)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		_, err := tw.Write(d)
		return err
	}

	for _, e := range existing {
		if err := writeEntry(e.name, e.data); err != nil {
			return err
		}
	}
	for _, e := range entries {
		if err := writeEntry(e.name, e.data); err != nil {
			return err
		}
	}

	if err := tw.Close(); err != nil {
		return err
	}
	return gw.Close()
}

// addEntry is the internal writer. It rebuilds the archive with the new entry appended.
// For a fresh archive it creates the file. Existing entries with the same name are replaced.
func addEntry(archivePath, entryName string, data []byte) error {
	// Read existing entries (if archive exists).
	type entry struct {
		name string
		data []byte
	}
	var existing []entry

	if f, err := os.Open(archivePath); err == nil {
		gr, err := gzip.NewReader(f)
		if err == nil {
			tr := tar.NewReader(gr)
			for {
				hdr, err := tr.Next()
				if err == io.EOF {
					break
				}
				if err != nil {
					break
				}
				if hdr.Name == entryName {
					continue // will be replaced
				}
				b, _ := io.ReadAll(tr)
				existing = append(existing, entry{hdr.Name, b})
			}
			gr.Close()
		}
		f.Close()
	}

	// Write new archive.
	out, err := os.Create(archivePath)
	if err != nil {
		return fmt.Errorf("create archive: %w", err)
	}
	defer out.Close()

	gw := gzip.NewWriter(out)
	tw := tar.NewWriter(gw)

	writeEntry := func(name string, d []byte) error {
		hdr := &tar.Header{
			Name: name,
			Mode: 0600,
			Size: int64(len(d)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		_, err := tw.Write(d)
		return err
	}

	for _, e := range existing {
		if err := writeEntry(e.name, e.data); err != nil {
			return err
		}
	}
	if err := writeEntry(entryName, data); err != nil {
		return err
	}

	if err := tw.Close(); err != nil {
		return err
	}
	return gw.Close()
}
