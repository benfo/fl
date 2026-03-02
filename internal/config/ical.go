package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

// ICalFeed is a named iCal subscription URL.
type ICalFeed struct {
	Name string `mapstructure:"name" yaml:"name"`
	URL  string `mapstructure:"url"  yaml:"url"`
}

// ICalFeeds returns all configured iCal feeds.
func ICalFeeds() ([]ICalFeed, error) {
	raw := viper.Get("calendar.ical")
	if raw == nil {
		return nil, nil
	}

	var feeds []ICalFeed
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           &feeds,
		WeaklyTypedInput: true,
	})
	if err != nil {
		return nil, err
	}
	if err := decoder.Decode(raw); err != nil {
		return nil, fmt.Errorf("decoding calendar.ical config: %w", err)
	}
	return feeds, nil
}

// AddICalFeed appends a new feed and persists the config file.
func AddICalFeed(name, rawURL string) error {
	feeds, err := ICalFeeds()
	if err != nil {
		return err
	}

	for _, f := range feeds {
		if strings.EqualFold(f.Name, name) {
			return fmt.Errorf("a feed named %q already exists", name)
		}
	}

	feeds = append(feeds, ICalFeed{Name: name, URL: rawURL})
	return saveICalFeeds(feeds)
}

// RemoveICalFeed removes the feed with the given name and persists the config.
func RemoveICalFeed(name string) error {
	feeds, err := ICalFeeds()
	if err != nil {
		return err
	}

	filtered := feeds[:0]
	found := false
	for _, f := range feeds {
		if strings.EqualFold(f.Name, name) {
			found = true
			continue
		}
		filtered = append(filtered, f)
	}
	if !found {
		return fmt.Errorf("no feed named %q found", name)
	}
	return saveICalFeeds(filtered)
}

func saveICalFeeds(feeds []ICalFeed) error {
	// Convert to []map[string]string so viper serialises it correctly.
	raw := make([]map[string]string, len(feeds))
	for i, f := range feeds {
		raw[i] = map[string]string{"name": f.Name, "url": f.URL}
	}
	viper.Set("calendar.ical", raw)

	dir, err := configDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	return viper.WriteConfigAs(dir + "/config.yaml")
}
