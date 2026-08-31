package goplugin

import "GoCraft/core/plugin"

// Instance is one native plugin process.
type Instance struct {
	manifest plugin.Manifest
}

func (i *Instance) Manifest() plugin.Manifest { return i.manifest }
