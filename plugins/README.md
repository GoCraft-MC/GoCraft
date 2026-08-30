# GoCraft plugins

Put installed `.gcpkg` bundles in this directory, then start GoCraft.

```text
plugins/
  example.gcpkg
```

When the plugin loads, GoCraft creates a data directory named after its
manifest ID. Default files stored below `config/` in the bundle are copied
there once:

```text
plugins/
  example.gcpkg
  dev.example.plugin/
    config.yml
```

Existing data files are never replaced during startup or an upgrade.
