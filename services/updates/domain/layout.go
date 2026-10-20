package domain

// Keys of the release bucket, relative to its configured prefix. Files are
// immutable; channel objects are replaced by the platform on publication.
func FileKey(name string) string { return "files/" + name }

// ManifestKey is the MyGo manifest a channel serves for target.
func ManifestKey(channel Channel, target Target) string {
	return "channels/" + string(channel) + "/update-" + string(target) + ".json"
}

// IndexKey is the channel index.
func IndexKey(channel Channel) string { return "channels/" + string(channel) + "/index.json" }
