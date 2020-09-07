package main

import (
	"archive/tar"
	"archive/zip"
	"compress/bzip2"
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime/pprof"
	"strings"
)

const (
	metadataExt         = ".metadata.json"
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

var (
	rootHTMLTemplate = template.Must(template.New("root").Parse(`
<!DOCTYPE html>
<html>
  <head>
    <title>Simple index</title>
  </head>
  <body>
    {{- range . }}
    <a href="{{ .Metadata.NormName }}">{{ .Metadata.Name }}</a>
    {{- end }}
  </body>
</html>
`))
	packageHTMLTemplate = template.Must(template.New("pkg").Parse(`
{{- $firstPkg := index . 0 }}
<!DOCTYPE html>
<html>
  <head>
    <title>Links for {{ $firstPkg.Metadata.Name }}</title>
  </head>
  <body>
    <h1>Links for {{ $firstPkg.Metadata.Name }}</h1>
    {{- range . }}
    <a href="{{ .Filename }}#sha256={{ .Metadata.Hash }}">{{ .Filename }}</a><br/>
    {{- end }}
  </body>
</html>
`))
)

type metadataGetFn func(string) (*metadata, error)

var metadataGetters = map[string]metadataGetFn{
	".tar.bz2": getMetadataFromTarBz2,
	".tar.gz":  getMetadataFromTarGz,
	".whl":     getMetadataFromWheel,
	".zip":     getMetadataFromZip,
}

type extractFn func(string, string) (string, error)

type metadata struct {
	Name     string `json:"name"`
	NormName string `json:"norm_name"`
	Version  string `json:"version"`
	Homepage string `json:"homepage"`
	Trusted  bool   `json:"trusted"`
	Hash     string `json:"sha256"`
}

type pkg struct {
	Path     string
	Filename string
	Metadata *metadata
}

func normalize(name string) string {
	return normRegex.ReplaceAllLiteralString(name, "-")
}

func parsePackageMetadata(s string) (*metadata, error) {
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
	meta := &metadata{
		Name:     name,
		NormName: normalize(name),
		Version:  version,
		Homepage: homepage,
		Trusted:  true,
	}
	return meta, nil
}

func getMetadataFromArchive(filePath string, ext string, fn extractFn, member string) (*metadata, error) {
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
		meta := &metadata{
			Name:     name,
			NormName: normalize(name),
			Version:  version,
			Trusted:  true,
		}
		return meta, nil
	}
	return parsePackageMetadata(meta)
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

func getMetadataFromTarBz2(path string) (*metadata, error) {
	return getMetadataFromArchive(path, ".tar.bz2", extractMemberFromTar, "")
}

func getMetadataFromTarGz(path string) (*metadata, error) {
	return getMetadataFromArchive(path, ".tar.gz", extractMemberFromTar, "")
}

func getMetadataFromWheel(filePath string) (*metadata, error) {
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
	meta, err := parsePackageMetadata(rawMeta)
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

func getMetadataFromZip(path string) (*metadata, error) {
	return getMetadataFromArchive(path, ".zip", extractMemberFromZip, "")
}

func getMetadataFromJSON(path string) (*metadata, error) {
	f, err := os.Open(path + metadataExt)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var meta metadata
	err = json.NewDecoder(f).Decode(&meta)
	if err != nil {
		return nil, err
	}
	if meta.Name == "" {
		return nil, nil
	}
	return &meta, nil
}

func getPackageMetadata(path string) (*metadata, error) {
	meta, err := getMetadataFromJSON(path)
	if err == nil && meta != nil {
		return meta, nil
	}
	var getter metadataGetFn
	for ext, g := range metadataGetters {
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

func getPackage(path string) (*pkg, error) {
	meta, err := getPackageMetadata(path)
	if err != nil {
		return nil, fmt.Errorf("error while processing %q: %w", path, err)
	}
	return &pkg{path, filepath.Base(path), meta}, nil
}

func fixPackageNames(pkgs []*pkg) {
	byTrust := make(map[bool][]*pkg, 2)
	for _, pkg := range pkgs {
		trusted := pkg.Metadata.Trusted
		byTrust[trusted] = append(byTrust[trusted], pkg)
	}
	trustedPkgs := byTrust[true]
	if len(trustedPkgs) > 0 {
		trustedName := trustedPkgs[0].Metadata.Name
		for _, pkg := range byTrust[false] {
			pkg.Metadata.Name = trustedName
		}
	}
}

func groupPackagesByNormName(pkgs []*pkg) map[string][]*pkg {
	byNormName := make(map[string][]*pkg)
	for _, pkg := range pkgs {
		normName := pkg.Metadata.NormName
		byNormName[normName] = append(byNormName[normName], pkg)
	}
	return byNormName
}

func listPackages(dir string, fixNames bool) ([]*pkg, error) {
	pkgs := make([]*pkg, 0)
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && !strings.HasSuffix(path, metadataExt) {
			p, err := getPackage(path)
			if err != nil {
				return err
			}
			pkgs = append(pkgs, p)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if fixNames {
		for _, pkgs := range groupPackagesByNormName(pkgs) {
			fixPackageNames(pkgs)
		}
	}
	return pkgs, nil
}

// TODO: sort respecting locale
func listPackageByNames(downloadDir string) (map[string][]*pkg, error) {
	pkgs, err := listPackages(downloadDir, true)
	if err != nil {
		return nil, err
	}
	byName := make(map[string][]*pkg)
	for _, pkg := range pkgs {
		name := pkg.Metadata.Name
		byName[name] = append(byName[name], pkg)
	}
	return byName, nil
}

func listPackageNames(downloadDir string) ([]string, error) {
	byName, err := listPackageByNames(downloadDir)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(byName))
	for name, _ := range byName {
		names = append(names, name)
	}
	return names, nil
}

func generateRootHTML(w io.Writer, pkgs []*pkg) error {
	return rootHTMLTemplate.ExecuteTemplate(w, "root", pkgs)
}

func generatePackageHTML(w io.Writer, pkgs []*pkg) error {
	return packageHTMLTemplate.ExecuteTemplate(w, "pkg", pkgs)
}

func createMirror(downloadDir string, mirrorDir string, pkgs []*pkg) error {
	downloadDir, err := filepath.Abs(downloadDir)
	if err != nil {
		return err
	}
	mirrorDir, err = filepath.Abs(mirrorDir)
	if err != nil {
		return err
	}
	if pkgs == nil || len(pkgs) == 0 {
		var err error
		pkgs, err = listPackages(downloadDir, false)
		if err != nil {
			return err
		}
	}
	pkgsByNormName := groupPackagesByNormName(pkgs)
	rootPkgs := make([]*pkg, 0, len(pkgsByNormName))
	for normName, pkgs := range pkgsByNormName {
		fmt.Println(normName)
		dir := filepath.Join(mirrorDir, normName)
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
		fixPackageNames(pkgs)
		for _, pkg := range pkgs {
			dest := filepath.Join(dir, pkg.Filename)
			link, err := filepath.Rel(dir, pkg.Path)
			if err != nil {
				return err
			}
			err = os.Symlink(link, dest)
			if err != nil && !errors.Is(err, os.ErrExist) {
				return err
			}
		}
		f, err := os.Create(filepath.Join(dir, "index.html"))
		if err != nil {
			return err
		}
		err = generatePackageHTML(f, pkgs)
		f.Close()
		if err != nil {
			return err
		}
		rootPkgs = append(rootPkgs, pkgs[0])
	}
	f, err := os.Create(filepath.Join(mirrorDir, "index.html"))
	if err != nil {
		return err
	}
	defer f.Close()
	return generateRootHTML(f, rootPkgs)
}

func createMetadataFiles(downloadDir string, overwrite bool) error {
	if overwrite {
		err := filepath.Walk(downloadDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && strings.HasSuffix(path, metadataExt) {
				err := os.Remove(path)
				if err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	pkgs, err := listPackages(downloadDir, true)
	if err != nil {
		return err
	}
	for _, pkg := range pkgs {
		metadataFile := pkg.Path + metadataExt
		if _, err := os.Stat(metadataFile); !errors.Is(err, os.ErrNotExist) {
			continue
		}
		f, err := os.Create(metadataFile)
		if err != nil {
			return err
		}
		err = json.NewEncoder(f).Encode(pkg.Metadata)
		if err != nil {
			return err
		}
		f.Close()
	}
	return nil
}

func main() {
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to `file`")
	overwrite := flag.Bool("overwrite", false, "overwrite metadata files")
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal("could not create CPU profile: ", err)
		}
		defer f.Close() // error handling omitted for example
		if err := pprof.StartCPUProfile(f); err != nil {
			log.Fatal("could not start CPU profile: ", err)
		}
		defer pprof.StopCPUProfile()
	}
	dir := flag.Args()[0]
	err := createMetadataFiles(dir, *overwrite)
	if err != nil {
		fmt.Println(err)
	}
	if pkgs, err := listPackages(dir, true); err != nil {
		fmt.Println(err)
	} else {
		for _, pkg := range pkgs {
			fmt.Printf("%v: %+v\n", pkg.Path, pkg.Metadata)
		}
		fmt.Printf("Number of packages: %v\n", len(pkgs))
		err := generateRootHTML(os.Stdout, pkgs)
		if err != nil {
			fmt.Printf("HTML generation failed: %v\n", err)
		}
		err = generatePackageHTML(os.Stdout, pkgs)
		if err != nil {
			fmt.Printf("HTML generation failed: %v\n", err)
		}
	}
	if pkgs, err := listPackageByNames(dir); err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(pkgs)
	}
	if names, err := listPackageNames(dir); err != nil {
		fmt.Println(err)
	} else {
		fmt.Printf("%v %v\n", len(names), names)
	}
	if err := createMirror(dir, flag.Args()[1], nil); err != nil {
		fmt.Println(err)
	}
}
