package plugin

import "context"

// LoadDirectory discovers, provisions, and loads every bundle in a directory.
func (r *Registry) LoadDirectory(ctx context.Context, directory string) error {
	bundles, err := ScanBundles(directory)
	if err != nil {
		return err
	}
	if err := r.Preflight(ctx, bundles); err != nil {
		return err
	}
	return r.LoadAll(ctx, bundles)
}

// LoadDefaultDirectory loads bundles placed in the server-root plugins folder.
func (r *Registry) LoadDefaultDirectory(ctx context.Context) error {
	return r.LoadDirectory(ctx, DefaultDirectory)
}
