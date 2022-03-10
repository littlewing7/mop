// Copyright (c) 2013-2019 by Michael Dvorkin and contributors. All Rights Reserved.
// Use of this source code is governed by a MIT-style license that can
// be found in the LICENSE file.

package mop

import (
	"encoding/json"
	"io/ioutil"
	"sort"
	"strings"

	"github.com/Knetic/govaluate"
)

const defaultGainColor = "green"
const defaultLossColor = "red"
const defaultTagColor = "yellow"
const defaultHeaderColor = "lightgray"
const defaultTimeColor = "lightgray"
const defaultColor = "lightgray"

// Profile manages Mop program settings as defined by user (ex. list of
// stock tickers). The settings are serialized using JSON and saved in
// the ~/.moprc file.
type Profile struct {
	Tickers       []string // List of stock tickers to display.
	MarketRefresh int      // Time interval to refresh market data.
	QuotesRefresh int      // Time interval to refresh stock quotes.
	SortColumn    int      // Column number by which we sort stock quotes.
	Ascending     bool     // True when sort order is ascending.
	Grouped       bool     // True when stocks are grouped by advancing/declining.
	Filter        string   // Filter in human form
	Colors        struct { // User defined colors
		Gain    string
		Loss    string
		Tag     string
		Header  string
		Time    string
		Default string
	}
	filterExpression *govaluate.EvaluableExpression // The filter as a govaluate expression
	selectedColumn   int                            // Stores selected column number when the column editor is active.
	filename         string                         // Path to the file in which the configuration is stored
}

func IsSupportedColor(colorName string) bool {
	switch colorName {
	case
		"black",
		"red",
		"green",
		"yellow",
		"blue",
		"magenta",
		"cyan",
		"white",
		"darkgray",
		"lightred",
		"lightgreen",
		"lightyellow",
		"lightblue",
		"lightmagenta",
		"lightcyan",
		"lightgray":
		return true
	}
	return false
}

// Creates the profile and attempts to load the settings from ~/.moprc file.
// If the file is not there it gets created with default values.
func NewProfile(filename string) *Profile {
	profile := &Profile{filename: filename}
	data, err := ioutil.ReadFile(filename)
	if err != nil { // Set default values:
		profile.MarketRefresh = 12 // Market data gets fetched every 12s (5 times per minute).
		profile.QuotesRefresh = 5  // Stock quotes get updated every 5s (12 times per minute).
		profile.Grouped = false    // Stock quotes are *not* grouped by advancing/declining.
		profile.Tickers = []string{`AAPL`, `C`, `GOOG`, `IBM`, `KO`, `ORCL`, `V`}
		profile.SortColumn = 0   // Stock quotes are sorted by ticker name.
		profile.Ascending = true // A to Z.
		profile.Filter = ""
		profile.Colors.Gain = defaultGainColor
		profile.Colors.Loss = defaultLossColor
		profile.Colors.Tag = defaultTagColor
		profile.Colors.Header = defaultHeaderColor
		profile.Colors.Time = defaultTimeColor
		profile.Colors.Default = defaultColor
		profile.Save()
	} else {
		json.Unmarshal(data, profile)

		InitColor(&profile.Colors.Gain, defaultGainColor)
		InitColor(&profile.Colors.Loss, defaultLossColor)
		InitColor(&profile.Colors.Tag, defaultTagColor)
		InitColor(&profile.Colors.Header, defaultHeaderColor)
		InitColor(&profile.Colors.Time, defaultTimeColor)
		InitColor(&profile.Colors.Default, defaultColor)

		profile.SetFilter(profile.Filter)
	}
	profile.selectedColumn = -1

	return profile
}

func InitColor(color *string, defaultValue string) {
	*color = strings.ToLower(*color)
	if !IsSupportedColor(*color) {
		*color = defaultValue
	}
}

// Save serializes settings using JSON and saves them in ~/.moprc file.
func (profile *Profile) Save() error {
	data, err := json.MarshalIndent(profile, "", "    ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(profile.filename, data, 0644)
}

// AddTickers updates the list of existing tickers to add the new ones making
// sure there are no duplicates.
func (profile *Profile) AddTickers(tickers []string) (added int, err error) {
	added, err = 0, nil
	existing := make(map[string]bool)

	// Build a hash of existing tickers so we could look it up quickly.
	for _, ticker := range profile.Tickers {
		existing[ticker] = true
	}

	// Iterate over the list of new tickers excluding the ones that
	// already exist.
	for _, ticker := range tickers {
		if _, found := existing[ticker]; !found {
			profile.Tickers = append(profile.Tickers, ticker)
			added++
		}
	}

	if added > 0 {
		sort.Strings(profile.Tickers)
		err = profile.Save()
	}

	return
}

// RemoveTickers removes requested stock tickers from the list we track.
func (profile *Profile) RemoveTickers(tickers []string) (removed int, err error) {
	removed, err = 0, nil
	for _, ticker := range tickers {
		for i, existing := range profile.Tickers {
			if ticker == existing {
				// Requested ticker is there: remove i-th slice item.
				profile.Tickers = append(profile.Tickers[:i], profile.Tickers[i+1:]...)
				removed++
			}
		}
	}

	if removed > 0 {
		err = profile.Save()
	}

	return
}

// Reorder gets called by the column editor to either reverse sorting order
// for the current column, or to pick another sort column.
func (profile *Profile) Reorder() error {
	if profile.selectedColumn == profile.SortColumn {
		profile.Ascending = !profile.Ascending // Reverse sort order.
	} else {
		profile.SortColumn = profile.selectedColumn // Pick new sort column.
	}
	return profile.Save()
}

// Regroup flips the flag that controls whether the stock quotes are grouped
// by advancing/declining issues.
func (profile *Profile) Regroup() error {
	profile.Grouped = !profile.Grouped
	return profile.Save()
}

// SetFilter creates a govaluate.EvaluableExpression.
func (profile *Profile) SetFilter(filter string) {
	if len(filter) > 0 {
		var err error
		profile.filterExpression, err = govaluate.NewEvaluableExpression(filter)

		if err != nil {
			panic(err)
		}

	} else if len(filter) == 0 && profile.filterExpression != nil {
		profile.filterExpression = nil
	}

	profile.Filter = filter
}
