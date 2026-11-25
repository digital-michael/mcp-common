package datumdir

import (
	"os"
	"path/filepath"
	"testing"
)

func env(goos, home string, vars map[string]string) Env {
	return Env{GOOS: goos, Home: home, Getenv: func(k string) string { return vars[k] }}
}

// The same table runs for macOS and Linux: the platform follows the XDG layout on both.
func TestDir_macOSAndLinux(t *testing.T) {
	for _, goos := range []string{"darwin", "linux"} {
		t.Run(goos, func(t *testing.T) {
			home := "/home/u"
			cases := []struct {
				name string
				vars map[string]string
				kind Kind
				want string
			}{
				{"config default", nil, Config, "/home/u/.config/datum"},
				{"data default", nil, Data, "/home/u/.local/share/datum"},
				{"state default", nil, State, "/home/u/.local/state/datum"},
				{"config xdg", map[string]string{"XDG_CONFIG_HOME": "/x/c"}, Config, "/x/c/datum"},
				{"data xdg", map[string]string{"XDG_DATA_HOME": "/x/d"}, Data, "/x/d/datum"},
				{"state xdg", map[string]string{"XDG_STATE_HOME": "/x/s"}, State, "/x/s/datum"},
				{"direct beats xdg", map[string]string{"DATUM_CONFIG_DIR": "/direct", "XDG_CONFIG_HOME": "/x/c"}, Config, "/direct"},
				{"direct data", map[string]string{"DATUM_DATA_DIR": "/d"}, Data, "/d"},
				{"direct state", map[string]string{"DATUM_STATE_DIR": "/s"}, State, "/s"},
				{"another kind's xdg is not used", map[string]string{"XDG_DATA_HOME": "/x/d"}, Config, "/home/u/.config/datum"},
				{"the windows variables mean nothing here", map[string]string{"APPDATA": `C:\a`}, Config, "/home/u/.config/datum"},
			}
			for _, c := range cases {
				got, err := env(goos, home, c.vars).Dir(c.kind)
				if err != nil || got != filepath.FromSlash(c.want) {
					t.Errorf("%s: got %q, %v; want %q", c.name, got, err, c.want)
				}
			}
		})
	}
}

// Windows is implemented but not verified on Windows: this only shows what the rule would give.
func TestDir_windows(t *testing.T) {
	cases := []struct {
		name string
		vars map[string]string
		kind Kind
		want string
	}{
		{"config", map[string]string{"APPDATA": `C:\Users\u\AppData\Roaming`, "LOCALAPPDATA": `C:\Users\u\AppData\Local`}, Config, `C:\Users\u\AppData\Roaming\datum`},
		{"data", map[string]string{"APPDATA": `C:\Users\u\AppData\Roaming`, "LOCALAPPDATA": `C:\Users\u\AppData\Local`}, Data, `C:\Users\u\AppData\Local\datum`},
		{"state", map[string]string{"APPDATA": `C:\Users\u\AppData\Roaming`, "LOCALAPPDATA": `C:\Users\u\AppData\Local`}, State, `C:\Users\u\AppData\Local\datum\state`},
		{"no variables: from home", nil, Config, `C:\Users\u\AppData\Roaming\datum`},
		{"direct beats all", map[string]string{"DATUM_STATE_DIR": `D:\s`}, State, `D:\s`},
		{"xdg is not used", map[string]string{"XDG_CONFIG_HOME": "/x"}, Config, `C:\Users\u\AppData\Roaming\datum`},
	}
	for _, c := range cases {
		got, err := env("windows", `C:\Users\u`, c.vars).Dir(c.kind)
		if err != nil || got != c.want {
			t.Errorf("%s: got %q, %v; want %q", c.name, got, err, c.want)
		}
	}
}

func TestDir_noHome(t *testing.T) {
	if _, err := env("linux", "", nil).Dir(Config); err == nil {
		t.Error("no home and no override must be an error, not a relative path")
	}
	if got, err := env("linux", "", map[string]string{"DATUM_CONFIG_DIR": "/c"}).Dir(Config); err != nil || got != "/c" {
		t.Errorf("an override needs no home: %q, %v", got, err)
	}
	if _, err := env("linux", "/h", nil).Dir("bogus"); err == nil {
		t.Error("an unknown kind must be an error")
	}
}

// On the machine the tests run on, the rule gives the real answer: this is the macOS or the Linux run.
func TestCurrent_thisMachine(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("no home folder here")
	}
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("DATUM_CONFIG_DIR", "")
	got, err := ConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, ".config", "datum"); got != want && Current().GOOS != "windows" {
		t.Errorf("ConfigDir() = %q, want %q", got, want)
	}
}
