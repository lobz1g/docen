package docen

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

const (
	defaultAppName    = "app"
	defaultTagVersion = "alpine"
)

var (
	goModFile         = "go.mod"
	vendorFolderName  = "vendor"
	additionalFolders = map[string]bool{
		"static":    true,
		"assets":    true,
		"templates": true,
		"config":    true,
	}

	// readDir used for unit testing
	readDir = ioutil.ReadDir
	// runVer used for unit testing
	runVer = runtime.Version
	// openFile used for unit testing
	openFile = os.Open
)

type (
	docener interface {
		New() *Docen
	}

	additionalInfo map[string]bool

	Docen struct {
		timezone        string
		version         string
		port            string
		additionFolders additionalInfo
		additionFiles   additionalInfo
		isTestMode      bool
	}
)

// New method creates new instance of generator.
// By default, the golang version is taken from runtime.Version
// By default, additional folders are `static`, `templates`, `config` and `assets`.
func New() *Docen {
	d := &Docen{
		version:         getVersion(),
		additionFolders: getAdditionalFolders(),
		additionFiles:   newAdditionalInfo(),
	}
	return d
}

// SetGoVersion method allows you to set a specific version of golang.
func (d *Docen) SetGoVersion(version string) *Docen {
	d.version = fmt.Sprintf("%s-%s", version, defaultTagVersion)
	return d
}

// SetPort method allows you to set an exposed port. It can be as single port as a range of ports.
func (d *Docen) SetPort(port string) *Docen {
	d.port = port
	return d
}

// SetTimezone method allows you to set a specific timezone.
func (d *Docen) SetTimezone(timezone string) *Docen {
	d.timezone = timezone
	return d
}

// SetAdditionalFolder method allows you to set additional folders which will be added to a container.
func (d *Docen) SetAdditionalFolder(path string) *Docen {
	d.additionFolders.set(path)
	return d
}

// SetAdditionalFolder method allows you to set additional files which will be added to a container.
func (d *Docen) SetAdditionalFile(path string) *Docen {
	d.additionFiles.set(path)
	d.SetAdditionalFolder(filepath.Dir(path))
	return d
}

// SetTestMode method allows you to enable testing before starting to build the app.
func (d *Docen) SetTestMode(mode bool) *Docen {
	d.isTestMode = mode
	return d
}

// GenerateDockerfile method creates Dockerfile file.
// If vendor mode is enabled then building will be with `-mod=vendor` tag.
func (d *Docen) GenerateDockerfile() error {
	packageName := getPackageName()

	var data strings.Builder
	data.WriteString(fmt.Sprintf("FROM golang:%s as builder\n", d.version))
	data.WriteString("RUN apk update && apk add --no-cache git ca-certificates tzdata && update-ca-certificates\n")
	data.WriteString("RUN adduser -D -g '' appuser\n")

	data.WriteString(fmt.Sprintf("RUN mkdir -p /%s\n", packageName))
	for v := range d.additionFolders {
		data.WriteString(fmt.Sprintf("RUN mkdir -p /%s/%s\n", packageName, v))
	}
	data.WriteString(fmt.Sprintf("COPY . /%s\n", packageName))
	data.WriteString(fmt.Sprintf("WORKDIR /%s\n", packageName))
	if d.isTestMode {
		data.WriteString("RUN CGO_ENABLED=0 go test ./...\n")
	}

	var vendorTag string
	if isVendorMode() {
		vendorTag = "-mod=vendor"
	}
	data.WriteString(
		fmt.Sprintf(
			"RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build %s -ldflags=\"-w -s\" -o /%s\n",
			vendorTag, packageName,
		),
	)

	data.WriteString("FROM scratch\n")
	data.WriteString("COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo\n")
	data.WriteString("COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/\n")
	data.WriteString("COPY --from=builder /etc/passwd /etc/passwd\n")
	if d.timezone != "" {
		data.WriteString(fmt.Sprintf("ENV TZ=%s\n", d.timezone))
	}
	data.WriteString(fmt.Sprintf("COPY --from=builder /%s /%s\n", packageName, packageName))
	for v := range d.additionFolders {
		data.WriteString(fmt.Sprintf("COPY --from=builder /%s/%s /%s/%s\n", packageName, v, packageName, v))
	}
	for v := range d.additionFiles {
		data.WriteString(fmt.Sprintf("COPY --from=builder /%s/%s /%s/%s\n", packageName, v, packageName, v))
	}

	data.WriteString("USER appuser\n")
	if d.port != "" {
		data.WriteString(fmt.Sprintf("EXPOSE %s\n", d.port))
	}
	data.WriteString(fmt.Sprintf("ENTRYPOINT [\"/%s\"]\n", packageName))

	err := createDockerfile(data.String())

	return err
}

func getVersion() string {
	v := runVer()
	re := regexp.MustCompile("[0-9.]+")
	version := re.FindAllString(v, -1)
	if len(version) == 0 {
		return defaultTagVersion
	}
	return fmt.Sprintf("%s-%s", strings.Join(version, ""), defaultTagVersion)
}

func getPackageName() string {
	file, err := openFile(goModFile)
	if err != nil {
		return defaultAppName
	}
	defer file.Close()

	return parsePackageName(file)
}

func parsePackageName(r io.Reader) string {
	reader := bufio.NewReader(r)
	data, _, err := reader.ReadLine()
	if err != nil {
		return defaultAppName
	}

	module := strings.ReplaceAll(string(data), "module ", "")
	if string(module[0]) == "\"" {
		module = strings.ReplaceAll(module, "\"", "")
	}

	name := strings.Split(module, "/")

	return strings.ReplaceAll(name[len(name)-1], ".", "_")
}

func getAdditionalFolders() additionalInfo {
	folders := newAdditionalInfo()

	files, err := getProjectFiles()
	if err != nil {
		return folders
	}

	for _, f := range files {
		if f.IsDir() && additionalFolders[f.Name()] {
			folders.set(f.Name())
		}
	}

	return folders
}

func isVendorMode() bool {
	files, err := getProjectFiles()
	if err != nil {
		return false
	}

	for _, f := range files {
		if f.IsDir() && f.Name() == vendorFolderName {
			return true
		}
	}

	return false
}

func getProjectFiles() ([]fs.FileInfo, error) {
	files, err := readDir("./")
	if err != nil {
		return nil, err
	}

	return files, nil
}

func createDockerfile(data string) error {
	return os.WriteFile("Dockerfile", []byte(data), 0644)
}

func newAdditionalInfo() additionalInfo {
	return map[string]bool{}
}

func (a additionalInfo) set(e string) {
	a[e] = true
}
