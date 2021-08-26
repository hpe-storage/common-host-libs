// Copyright 2019 Hewlett Packard Enterprise Development LP

package plugin

var (
	// PluginConfigDir represents config directory for plugin
	PluginConfigDir = ""
)

// GetOrCreatePluginConfigDirectory get or create plugin config directory
func GetOrCreatePluginConfigDirectory() (string, error) {
	return "", nil
}

// set default filesystem
func setDefaultFilesystem(reqOpts map[string]interface{}) (err error) {
	// dont need to set the default filesystem for linux.
	return nil
}

func setDefaultVolumeDir(reqOpts map[string]interface{}) (err error) {
	// dont need to se the default here for linux
	return nil
}
