package docen

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"log"
	"reflect"
	"strings"
	"testing"
)

type docenMock struct{}

func (docenMock) New() *Docen {
	return New()
}

var docen = docenMock{}

func ExampleNew() {
	err := docen.New().
		SetGoVersion("1.14.9").
		SetPort("3000").
		SetTimezone("Europe/Moscow").
		SetTestMode(true).
		SetAdditionalFolder("my-folder/some-files").
		SetAdditionalFile("another-folder/some-files/file").
		GenerateDockerfile()
	if err != nil {
		log.Fatal(err)
	}
}

func ExampleDocen_SetGoVersion() {
	docen.New().SetGoVersion("1.14.9")
}

func ExampleDocen_SetPort() {
	docen.New().SetPort("3000-4000")
}

func ExampleDocen_SetTimezone() {
	docen.New().SetTimezone("Europe/Paris")
}

func ExampleDocen_SetAdditionalFolder() {
	docen.New().SetAdditionalFolder("my-folder/some-files")
}

func ExampleDocen_SetAdditionalFile() {
	docen.New().SetAdditionalFile("my-folder/some-files/file")
}

func ExampleDocen_SetTestMode() {
	docen.New().SetTestMode(true)
}

func ExampleDocen_GenerateDockerfile() {
	err := docen.New().GenerateDockerfile()
	if err != nil {
		log.Fatal(err)
	}
}

func Test_getVersion(t *testing.T) {
	oldRuntimeVersion := runVer
	defer func() {
		runVer = oldRuntimeVersion
	}()

	tests := []struct {
		name           string
		want           string
		runtimeVersion func() string
	}{
		{
			name:           "default version",
			want:           defaultTagVersion,
			runtimeVersion: func() string { return "without version" },
		},
		{
			name:           "runtime version",
			want:           "1.13-" + defaultTagVersion,
			runtimeVersion: func() string { return "go1.13" },
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runVer = tt.runtimeVersion
			if got := getVersion(); got != tt.want {
				t.Errorf("getVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}

type fakeFolder struct {
	fs.FileInfo
	name string
}

func (f *fakeFolder) Name() string {
	return f.name
}

func (f *fakeFolder) IsDir() bool {
	return true
}

func Test_getAdditionalFolders(t *testing.T) {
	oldReadDir := readDir
	defer func() {
		readDir = oldReadDir
	}()

	tests := []struct {
		name    string
		want    map[string]bool
		readDir func(dirname string) ([]fs.FileInfo, error)
	}{
		{
			name:    "failed read dir",
			want:    map[string]bool{},
			readDir: func(dirname string) ([]fs.FileInfo, error) { return nil, errors.New("fake error") },
		},
		{
			name:    "empty dir",
			want:    map[string]bool{},
			readDir: func(dirname string) ([]fs.FileInfo, error) { return []fs.FileInfo{}, nil },
		},
		{
			name: "with folders",
			want: map[string]bool{
				"assets": true,
				"static": true,
				"config": true,
			},
			readDir: func(dirname string) ([]fs.FileInfo, error) {
				return []fs.FileInfo{
					&fakeFolder{name: "assets"},
					&fakeFolder{name: "static"},
					&fakeFolder{name: "config"},
				}, nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			readDir = tt.readDir
			if got := getAdditionalFolders(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getAdditionalFolders() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isVendorMode(t *testing.T) {
	oldReadDir := readDir
	defer func() {
		readDir = oldReadDir
	}()

	tests := []struct {
		name    string
		want    bool
		readDir func(dirname string) ([]fs.FileInfo, error)
	}{

		{
			name:    "failed read dir",
			want:    false,
			readDir: func(dirname string) ([]fs.FileInfo, error) { return nil, errors.New("fake error") },
		},
		{
			name:    "empty dir",
			want:    false,
			readDir: func(dirname string) ([]fs.FileInfo, error) { return []fs.FileInfo{}, nil },
		},
		{
			name: "without vendor folder",
			want: false,
			readDir: func(dirname string) ([]fs.FileInfo, error) {
				return []fs.FileInfo{
					&fakeFolder{name: "static"},
					&fakeFolder{name: "config"},
				}, nil
			},
		},
		{
			name: "with vendor folder",
			want: true,
			readDir: func(dirname string) ([]fs.FileInfo, error) {
				return []fs.FileInfo{
					&fakeFolder{name: "config"},
					&fakeFolder{name: "vendor"},
				}, nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			readDir = tt.readDir
			if got := isVendorMode(); got != tt.want {
				t.Errorf("isVendorMode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_parsePackageName(t *testing.T) {
	tests := []struct {
		name   string
		reader io.Reader
		want   string
	}{
		{
			name:   "failed read",
			reader: bytes.NewReader(nil),
			want:   defaultAppName,
		},
		{
			name:   "module name with quotation marks",
			reader: strings.NewReader(`module "testmodulename"`),
			want:   "testmodulename",
		},
		{
			name:   "module name without quotation marks",
			reader: strings.NewReader(`module testmodulename`),
			want:   "testmodulename",
		},
		{
			name:   "module name with url",
			reader: strings.NewReader(`module github.com/lobz1g/docen`),
			want:   "docen",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parsePackageName(tt.reader); got != tt.want {
				t.Errorf("parsePackageName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNew(t *testing.T) {
	oldReadDir := readDir
	oldRuntimeVersion := runVer
	defer func() {
		runVer = oldRuntimeVersion
		readDir = oldReadDir
	}()
	runVer = func() string { return "go1.13" }
	readDir = func(dirname string) ([]fs.FileInfo, error) { return []fs.FileInfo{}, nil }

	want := &Docen{
		version:         "1.13-alpine",
		additionFolders: map[string]bool{},
		additionFiles:   map[string]bool{},
	}

	t.Run(t.Name(), func(t *testing.T) {
		if got := New(); !reflect.DeepEqual(got, want) {
			t.Errorf("New() = %v, want %v", got, want)
		}
	})
}

func TestDocen_SetGoVersion(t *testing.T) {
	want := &Docen{
		version: "1.13-alpine",
	}

	d := &Docen{}
	t.Run(t.Name(), func(t *testing.T) {
		if got := d.SetGoVersion("1.13"); !reflect.DeepEqual(got, want) {
			t.Errorf("New() = %v, want %v", got, want)
		}
	})

}

func TestDocen_SetPort(t *testing.T) {
	want := &Docen{
		port: "3000",
	}

	d := &Docen{}
	t.Run(t.Name(), func(t *testing.T) {
		if got := d.SetPort("3000"); !reflect.DeepEqual(got, want) {
			t.Errorf("New() = %v, want %v", got, want)
		}
	})
}

func TestDocen_SetTimezone(t *testing.T) {
	want := &Docen{
		timezone: "Time/Zone",
	}

	d := &Docen{}
	t.Run(t.Name(), func(t *testing.T) {
		if got := d.SetTimezone("Time/Zone"); !reflect.DeepEqual(got, want) {
			t.Errorf("New() = %v, want %v", got, want)
		}
	})
}

func TestDocen_SetAdditionalFolder(t *testing.T) {
	want := &Docen{
		additionFolders: map[string]bool{
			"test":   true,
			"folder": true,
		},
	}

	d := &Docen{
		additionFolders: map[string]bool{},
	}
	t.Run(t.Name(), func(t *testing.T) {
		if got := d.SetAdditionalFolder("test").SetAdditionalFolder("folder"); !reflect.DeepEqual(got, want) {

			t.Errorf("New() = %v, want %v", got, want)
		}
	})

}

func TestDocen_SetAdditionalFile(t *testing.T) {
	want := &Docen{
		additionFolders: map[string]bool{
			"test":   true,
			"folder": true,
		},
		additionFiles: map[string]bool{
			"test/file1":   true,
			"folder/file2": true,
		},
	}

	d := &Docen{
		additionFolders: map[string]bool{},
		additionFiles:   map[string]bool{},
	}
	t.Run(t.Name(), func(t *testing.T) {
		if got := d.SetAdditionalFile("test/file1").SetAdditionalFile("folder/file2"); !reflect.DeepEqual(got, want) {

			t.Errorf("New() = %v, want %v", got, want)
		}
	})

}

func TestDocen_SetTestMode(t *testing.T) {
	want := &Docen{
		isTestMode: true,
	}

	d := &Docen{}
	t.Run(t.Name(), func(t *testing.T) {
		if got := d.SetTestMode(true); !reflect.DeepEqual(got, want) {
			t.Errorf("New() = %v, want %v", got, want)
		}
	})

}
