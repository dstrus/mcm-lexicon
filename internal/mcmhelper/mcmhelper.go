// Package mcmhelper reads MCM Helper's menus: the declarative config.json
// at MCM/Config/<id>/config.json that defines a menu's pages and controls,
// the settings.ini beside it with the mod's defaults, and the player's own
// values in MCM/Settings/<id>.ini.
package mcmhelper

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

// Config is a menu as config.json defines it.
type Config struct {
	ModName     string
	DisplayName string
	Pages       []Page // a menu without pages has one, with Name ""
}

type Page struct {
	Name     string // pageDisplayName, raw; "" for the default page
	Controls []Control
}

// Control is one entry of a page's content.
type Control struct {
	ID         string // "key:Section" for a setting; "" for headers and text
	Key        string // from ID
	Section    string // from ID
	Type       string // toggle, slider, keymap, color, input, enum, stepper, text, header, empty, hiddenToggle…
	Text       string // the label, raw
	Help       string
	SourceType string // ModSettingBool, ModSettingInt, ModSettingFloat, ModSettingString, GlobalValue, PropertyValue…
	Min        *float64
	Max        *float64
	Step       *float64
	Format     string   // formatString
	Options    []string // an enum's or stepper's choices, raw
	Default    any      // valueOptions.defaultValue, as JSON gives it; nil when absent
	Value      any      // a text control's value, when the config fixes it
	Action     string   // action.type: CallFunction, CallGlobalFunction…
}

// ModSetting reports whether the control's value lives in the mod's ini
// files, where a default and the player's own value can be read.
func (c Control) ModSetting() bool { return strings.HasPrefix(c.SourceType, "ModSetting") }

type rawConfig struct {
	ModName     string       `json:"modName"`
	DisplayName string       `json:"displayName"`
	Content     []rawControl `json:"content"`
	Pages       []struct {
		Name    string       `json:"pageDisplayName"`
		Content []rawControl `json:"content"`
	} `json:"pages"`
}

type rawControl struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	Text         string `json:"text"`
	Help         string `json:"help"`
	ValueOptions struct {
		SourceType   string   `json:"sourceType"`
		Value        any      `json:"value"` // a text control's value, when fixed
		Min          *float64 `json:"min"`
		Max          *float64 `json:"max"`
		Step         *float64 `json:"step"`
		FormatString string   `json:"formatString"`
		Options      []string `json:"options"`
		DefaultValue any      `json:"defaultValue"`
	} `json:"valueOptions"`
	Action struct {
		Type string `json:"type"`
	} `json:"action"`
}

// Parse reads a config.json.
func Parse(b []byte) (Config, error) {
	b = bytes.TrimPrefix(b, []byte{0xEF, 0xBB, 0xBF})
	var raw rawConfig
	if err := json.Unmarshal(b, &raw); err != nil {
		return Config{}, fmt.Errorf("config.json isn't valid JSON: %w", err)
	}
	if raw.DisplayName == "" && raw.ModName == "" {
		return Config{}, fmt.Errorf("config.json has neither a displayName nor a modName")
	}
	c := Config{ModName: raw.ModName, DisplayName: raw.DisplayName}
	if c.DisplayName == "" {
		c.DisplayName = c.ModName
	}
	if len(raw.Content) > 0 || len(raw.Pages) == 0 {
		c.Pages = append(c.Pages, Page{Controls: controls(raw.Content)})
	}
	for _, p := range raw.Pages {
		c.Pages = append(c.Pages, Page{Name: p.Name, Controls: controls(p.Content)})
	}
	return c, nil
}

func controls(raw []rawControl) []Control {
	out := make([]Control, 0, len(raw))
	for _, r := range raw {
		c := Control{
			ID:         r.ID,
			Type:       r.Type,
			Text:       r.Text,
			Help:       r.Help,
			SourceType: r.ValueOptions.SourceType,
			Min:        r.ValueOptions.Min,
			Max:        r.ValueOptions.Max,
			Step:       r.ValueOptions.Step,
			Format:     r.ValueOptions.FormatString,
			Options:    r.ValueOptions.Options,
			Default:    r.ValueOptions.DefaultValue,
			Value:      r.ValueOptions.Value,
			Action:     r.Action.Type,
		}
		if key, section, ok := strings.Cut(r.ID, ":"); ok {
			c.Key, c.Section = key, section
		}
		out = append(out, c)
	}
	return out
}

// INI is an ini file's values, by section then key, both compared without
// regard to case.
type INI map[string]map[string]string

// ParseINI reads an ini file as MCM Helper writes them: [Section] headers,
// key = value lines, and ; or # comments.
func ParseINI(b []byte) INI {
	out := INI{}
	section := ""
	sc := bufio.NewScanner(bytes.NewReader(bytes.TrimPrefix(b, []byte{0xEF, 0xBB, 0xBF})))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		switch {
		case line == "", strings.HasPrefix(line, ";"), strings.HasPrefix(line, "#"):
		case strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]"):
			section = strings.ToLower(strings.TrimSpace(line[1 : len(line)-1]))
		default:
			key, value, ok := strings.Cut(line, "=")
			if !ok {
				continue
			}
			if out[section] == nil {
				out[section] = map[string]string{}
			}
			out[section][strings.ToLower(strings.TrimSpace(key))] = strings.TrimSpace(value)
		}
	}
	return out
}

// Get is the value of key in section, if the file sets it.
func (ini INI) Get(section, key string) (string, bool) {
	v, ok := ini[strings.ToLower(section)][strings.ToLower(key)]
	return v, ok
}
