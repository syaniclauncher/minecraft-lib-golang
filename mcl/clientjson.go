package mcl

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// readRawClientJSON reads the client.json of a version as a generic JSON
// document. The generic representation is needed to implement the
// inheritsFrom merging, which operates on the raw document.
func readRawClientJSON(minecraftDirectory, version string) (map[string]any, error) {
	path := filepath.Join(minecraftDirectory, "versions", version, version+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return raw, nil
}

// readClientJSONFile reads the client.json of a version from disk. The
// inheritsFrom chain is not resolved.
func readClientJSONFile(minecraftDirectory, version string) (*ClientJSON, error) {
	raw, err := readRawClientJSON(minecraftDirectory, version)
	if err != nil {
		return nil, err
	}
	return decodeClientJSON(raw)
}

// loadClientJSON reads the client.json of an installed version and resolves
// its inheritsFrom chain.
func loadClientJSON(minecraftDirectory, version string) (*ClientJSON, error) {
	raw, err := readRawClientJSON(minecraftDirectory, version)
	if err != nil {
		return nil, err
	}
	merged, err := resolveInheritance(minecraftDirectory, raw)
	if err != nil {
		return nil, err
	}
	return decodeClientJSON(merged)
}

// resolveInheritance merges a version with the version it inherits from.
//
// The parent is read as it is stored on disk, exactly like the Python
// reference implementation does. In practice the parent is always a vanilla
// version, which never inherits from anything itself.
func resolveInheritance(minecraftDirectory string, raw map[string]any) (map[string]any, error) {
	parentID, ok := raw["inheritsFrom"].(string)
	if !ok || parentID == "" {
		return raw, nil
	}
	parent, err := readRawClientJSON(minecraftDirectory, parentID)
	if err != nil {
		return nil, err
	}
	return mergeInherited(parent, raw), nil
}

// mergeInherited merges a child version document into its parent document.
//
// Lists are concatenated with the child first, nested lists are merged the
// same way and every other value of the child replaces the value of the
// parent. Libraries are special: a library of the parent is only inherited
// when the child does not already provide the same artifact.
func mergeInherited(parent, child map[string]any) map[string]any {
	result := parent

	// Libraries are merged separately: a library of the parent is skipped
	// when the child already contains the same artifact.
	if childLibraries, ok := child["libraries"].([]any); ok {
		owned := map[string]bool{}
		for _, lib := range childLibraries {
			if libMap, ok := lib.(map[string]any); ok {
				if name, ok := libMap["name"].(string); ok {
					owned[libraryNameWithoutVersion(name)] = true
				}
			}
		}
		merged := append([]any{}, childLibraries...)
		if parentLibraries, ok := parent["libraries"].([]any); ok {
			for _, lib := range parentLibraries {
				if libMap, ok := lib.(map[string]any); ok {
					name, _ := libMap["name"].(string)
					if owned[libraryNameWithoutVersion(name)] {
						continue
					}
				}
				merged = append(merged, lib)
			}
		}
		result["libraries"] = merged
	}

	for key, value := range child {
		if key == "libraries" {
			continue
		}
		if childList, ok := value.([]any); ok {
			if parentList, ok := result[key].([]any); ok {
				result[key] = append(append([]any{}, childList...), parentList...)
				continue
			}
		}
		if childMap, ok := value.(map[string]any); ok {
			if parentMap, ok := result[key].(map[string]any); ok {
				for subKey, subValue := range childMap {
					childSubList, ok := subValue.([]any)
					if !ok {
						// Only lists are merged; every other
						// value of the parent is kept.
						continue
					}
					if parentSubList, ok := parentMap[subKey].([]any); ok {
						parentMap[subKey] = append(append([]any{}, parentSubList...), childSubList...)
					}
				}
				continue
			}
		}
		result[key] = value
	}
	return result
}

// libraryNameWithoutVersion strips the version from a Maven coordinate, so
// "org.ow2.asm:asm:9.7.1" becomes "org.ow2.asm:asm".
func libraryNameWithoutVersion(name string) string {
	parts := strings.Split(name, ":")
	if len(parts) < 2 {
		return name
	}
	return strings.Join(parts[:len(parts)-1], ":")
}

// decodeClientJSON converts a raw version document into a ClientJSON.
func decodeClientJSON(raw map[string]any) (*ClientJSON, error) {
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var c ClientJSON
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parse version metadata: %w", err)
	}
	return &c, nil
}

// VersionMetadata returns the metadata of a Minecraft version. It reads the
// version from the local installation and falls back to Mojang's servers when
// the version is not installed.
func VersionMetadata(ctx context.Context, version, minecraftDirectory string) (*ClientJSON, error) {
	if _, err := os.Stat(filepath.Join(minecraftDirectory, "versions", version, version+".json")); err == nil {
		return loadClientJSON(minecraftDirectory, version)
	}

	m, err := Manifest(ctx)
	if err != nil {
		return nil, err
	}
	for _, v := range m.Versions {
		if v.ID != version {
			continue
		}
		var c ClientJSON
		if err := cachedJSON(ctx, v.URL, &c); err != nil {
			return nil, fmt.Errorf("fetch version %s: %w", version, err)
		}
		return &c, nil
	}
	return nil, fmt.Errorf("%w: %s", ErrVersionNotFound, version)
}
