# Specs

This directory contains specs for spec-driven development.

The up-to-date format is described in `tool/pkg/spec/README.md`
and an example can be found at `tool/pkg/spec/TEMPLATE.md`.

When implementing a new feature, you should first describe
it in a small number of specs, usually just one but
occassionally a handful depending on the complexity.

If it depends on existing specs in this specs/ directory,
be sure to include those dependencies in the yaml
frontmatter metadata at the top of the spec file.