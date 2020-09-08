package metadata

import (
	"archive/tar"
	"archive/zip"
	"compress/bzip2"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	FileExt             = ".metadata.json"
	archiveMetadataFile = "PKG-INFO"
)

var (
	normRegex     = regexp.MustCompile("[-_.]+")
	nameRegex     = regexp.MustCompile("(?m:^Name: (.*)$)")
	versionRegex  = regexp.MustCompile("(?m:^Version: (.*)$)")
	homepageRegex = regexp.MustCompile("(?m:^Home-[pP]age: (.*)$)")
)

var (
	errInvalidMetadata       = errors.New("invalid metadata")
	errInvalidArchiveName    = errors.New("invalid archive name")
	errArchiveMemberNotFound = errors.New("member not found in archive")
	errMetadataExtract       = errors.New("failed to extra metadata")
	errUnknownExtension      = errors.New("unknown extension")
)

type getFunc func(string) (*Metadata, error)
type extractFunc func(string, string) (string, error)

var getters = map[string]getFunc{
	".tar.bz2": getFromTarBz2,
	".tar.gz":  getFromTarGz,
	".whl":     getFromWheel,
	".zip":     getFromZip,
}

type Metadata struct {
	Name     string `json:"name"`
	NormName string `json:"norm_name"`
	Version  string `json:"version"`
	Homepage string `json:"homepage"`
	Trusted  bool   `json:"trusted"`
	Hash     string `json:"sha256"`
}

func (c *Metadata) MarshalJSON(w io.Writer) error {
	return json.NewEncoder(w).Encode(c)
}

func normalize(name string) string {
	return strings.ToLower(normRegex.ReplaceAllLiteralString(name, "-"))
}

func parse(s string) (*Metadata, error) {
	m := nameRegex.FindStringSubmatch(s)
	if len(m) == 0 {
		return nil, fmt.Errorf("%w: missing %q field", errInvalidMetadata, "Name")
	}
	name := strings.TrimSpace(m[1])
	m = versionRegex.FindStringSubmatch(s)
	if len(m) == 0 {
		return nil, fmt.Errorf("%w: missing %q field", errInvalidMetadata, "Version")
	}
	version := strings.TrimSpace(m[1])
	m = homepageRegex.FindStringSubmatch(s)
	if len(m) == 0 {
		return nil, fmt.Errorf("%w: missing %q field", errInvalidMetadata, "Home-page")
	}
	homepage := strings.TrimSpace(m[1])
	meta := &Metadata{
		Name:     name,
		NormName: normalize(name),
		Version:  version,
		Homepage: homepage,
		Trusted:  true,
	}
	return meta, nil
}

func getFromArchive(filePath string, ext string, fn extractFunc, member string) (*Metadata, error) {
	filename := filepath.Base(filePath)
	if !strings.HasSuffix(filename, ext) {
		return nil, errInvalidArchiveName
	}
	prefix := strings.TrimSuffix(filename, ext)
	if member == "" {
		member = archiveMetadataFile
	}
	metadataFile := path.Join(prefix, member)
	meta, err := fn(filePath, metadataFile)
	if err != nil {
		if !errors.Is(err, errArchiveMemberNotFound) {
			return nil, err
		}
		idx := strings.LastIndex(prefix, "-")
		if idx == -1 {
			return nil, errMetadataExtract
		}
		name := prefix[:idx]
		version := prefix[idx+1:]
		meta := &Metadata{
			Name:     name,
			NormName: normalize(name),
			Version:  version,
			Trusted:  true,
		}
		return meta, nil
	}
	return parse(meta)
}

func extractMemberFromZip(path string, member string) (string, error) {
	z, err := zip.OpenReader(path)
	if err != nil {
		return "", err
	}
	defer z.Close()
	for _, m := range z.File {
		if m.FileHeader.Name != member {
			continue
		}
		r, err := m.Open()
		if err != nil {
			return "", err
		}
		defer r.Close()
		data, err := ioutil.ReadAll(r)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	return "", errArchiveMemberNotFound
}

func extractMemberFromTar(filePath string, member string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	ext := filepath.Ext(filePath)
	var reader io.Reader = f
	if ext == ".gz" {
		z, err := gzip.NewReader(reader)
		if err != nil {
			return "", err
		}
		defer z.Close()
		reader = z
	} else if ext == ".bz2" {
		reader = bzip2.NewReader(reader)
	}
	t := tar.NewReader(reader)
	hdr, err := t.Next()
	for err == nil {
		if hdr.Name != member {
			hdr, err = t.Next()
			continue
		}
		data, err := ioutil.ReadAll(t)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	if errors.Is(err, io.EOF) {
		return "", errArchiveMemberNotFound
	}
	return "", err
}

func getFromTarBz2(path string) (*Metadata, error) {
	return getFromArchive(path, ".tar.bz2", extractMemberFromTar, "")
}

func getFromTarGz(path string) (*Metadata, error) {
	return getFromArchive(path, ".tar.gz", extractMemberFromTar, "")
}

func getFromWheel(filePath string) (*Metadata, error) {
	whl, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, err
	}
	defer whl.Close()
	whlName := filepath.Base(filePath)
	components := strings.SplitN(whlName, "-", 3)
	if len(components) != 3 {
		return nil, errInvalidArchiveName
	}
	prefix := strings.Join(components[:2], "-")
	metadataFile := path.Join(prefix+".dist-info", "METADATA")
	rawMeta, err := extractMemberFromZip(filePath, metadataFile)
	if err != nil {
		return nil, err
	}
	meta, err := parse(rawMeta)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(whlName, meta.Name) {
		meta.Trusted = false
		u, err := url.Parse(meta.Homepage)
		if err != nil {
			return nil, err
		}
		if len(u.Path) > 0 && strings.HasPrefix(u.Path, "/") {
			p := strings.TrimPrefix(u.Path, "/")
			if len(p) > 0 {
				idx := strings.LastIndex(p, "/")
				if idx != -1 && strings.HasPrefix(whlName, p[:idx]) {
					meta.Name = p[:idx]
				}
			}
		}
	}
	return meta, nil
}

func getFromZip(path string) (*Metadata, error) {
	return getFromArchive(path, ".zip", extractMemberFromZip, "")
}

func getFromJSON(path string) (*Metadata, error) {
	f, err := os.Open(path + FileExt)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var meta Metadata
	err = json.NewDecoder(f).Decode(&meta)
	if err != nil {
		return nil, err
	}
	if meta.Name == "" {
		return nil, nil
	}
	return &meta, nil
}

func Get(path string) (*Metadata, error) {
	meta, err := getFromJSON(path)
	if err == nil && meta != nil {
		return meta, nil
	}
	var getter getFunc
	for ext, g := range getters {
		if strings.HasSuffix(path, ext) {
			getter = g
			break
		}
	}
	if getter == nil {
		return nil, errUnknownExtension
	}
	meta, err = getter(path)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, err
	}
	meta.Hash = fmt.Sprintf("%x", h.Sum(nil))
	return meta, nil
}
