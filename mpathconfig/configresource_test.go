package mpathconfig

// Copyright 2017 Nimble Storage, Inc
import (
	"testing"
)

func TestParseDeviceSection(t *testing.T) {
	config, err := ParseConfig("./multipath_test.conf")

	if err != nil {
		t.Error(
			"Parsing multipath.conf failed ", err,
		)
	} else {
		// config found
		section, err := config.GetNimbleSection()
		if err != nil {
			t.Error(
				"Parsing nimble device section failed ", err,
			)
		} else {
			// section found
			if (section.GetProperties())["vendor"] != "\"Nimble\"" {
				t.Error(
					"Parsing options from nimble device section failed ", err,
				)
			}
			if section.GetParent().GetName() != "devices" {
				t.Error(
					"Parent section is not set correctly for nimble device section ", err,
				)
			}
		}
	}
}

func TestParseDefaultsSection(t *testing.T) {
	config, err := ParseConfig("./multipath_test.conf")

	if err != nil {
		t.Error(
			"Parsing multipath.conf failed ", err,
		)
	} else {
		// config found
		section, err := config.GetSection("defaults", "")
		if err != nil {
			t.Error(
				"Getting defaults section failed ", err,
			)
		} else {
			// section found
			if (section.GetProperties())["find_multipaths"] != "no" {
				t.Error(
					"Parsing options from defaults section failed ", err,
				)
			}
		}
	}
}

func TestParseBlacklistSection(t *testing.T) {
	config, err := ParseConfig("./multipath_test.conf")

	if err != nil {
		t.Error(
			"Parsing multipath.conf failed ", err,
		)
	} else {
		// config found
		section, err := config.GetSection("blacklist", "")
		if err != nil {
			t.Error(
				"Getting blacklists section failed ", err,
			)
		} else {
			// section found
			if (section.GetProperties())["devnode"] != "\"^(ram|raw|loop|fd|md|dm-|sr|scd|st)[0-9]*\"" {
				t.Error(
					"Parsing options from blacklist section failed ", err,
				)
			}
			if section.GetChildren().Len() == 0 {
				t.Error(
					"Parsing device section from blacklist section failed ", err,
				)
			}
		}
	}
}

func TestParseBlacklistExceptionsSection(t *testing.T) {
	config, err := ParseConfig("./multipath_test.conf")

	if err != nil {
		t.Error(
			"Parsing multipath.conf failed ", err,
		)
	} else {
		// config found
		section, err := config.GetSection("blacklist_exceptions", "")
		if err != nil {
			t.Error(
				"Getting blacklist_exceptions section failed ", err,
			)
		} else {
			// section found
			if section.GetChildren().Len() == 0 {
				t.Error(
					"Parsing device section from blacklist_exceptions section failed ", err,
				)
			}
		}
	}
}
