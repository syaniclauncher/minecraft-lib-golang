package mcl

import (
	"regexp"
	"runtime"
	"strconv"
)

// currentOSName returns the name the version metadata uses for the current
// operating system.
func currentOSName() string {
	switch runtime.GOOS {
	case "windows":
		return "windows"
	case "darwin":
		return "osx"
	default:
		return "linux"
	}
}

// ruleAllows reports whether a single rule permits the current platform and
// the given launch options.
//
// A rule with the action "allow" applies when all of its constraints match, a
// rule with the action "disallow" applies when they do not.
func ruleAllows(rule Rule, opts Options) bool {
	// The verdict when a constraint does not match: an "allow" rule does
	// not apply, a "disallow" rule does.
	fallback := rule.Action != "allow"

	if rule.OS != nil {
		switch rule.OS.Name {
		case "windows":
			if runtime.GOOS != "windows" {
				return fallback
			}
		case "osx":
			if runtime.GOOS != "darwin" {
				return fallback
			}
		case "linux":
			if runtime.GOOS != "linux" {
				return fallback
			}
		}
		if rule.OS.Arch == "x86" && strconv.IntSize != 32 {
			return fallback
		}
		if rule.OS.Version != "" && !matchOSVersion(rule.OS.Version) {
			return fallback
		}
	}

	for feature := range rule.Features {
		switch feature {
		case "has_custom_resolution":
			if !opts.CustomResolution {
				return fallback
			}
		case "is_demo_user":
			if !opts.Demo {
				return fallback
			}
		case "has_quick_plays_support":
			if opts.QuickPlayPath == "" {
				return fallback
			}
		case "is_quick_play_singleplayer":
			if opts.QuickPlaySingleplayer == "" {
				return fallback
			}
		case "is_quick_play_multiplayer":
			if opts.QuickPlayMultiplayer == "" {
				return fallback
			}
		case "is_quick_play_realms":
			if opts.QuickPlayRealms == "" {
				return fallback
			}
		}
	}

	return !fallback
}

// ruleListAllows reports whether every rule of the list permits the current
// platform and the given launch options.
func ruleListAllows(rules []Rule, opts Options) bool {
	for _, rule := range rules {
		if !ruleAllows(rule, opts) {
			return false
		}
	}
	return true
}

// matchOSVersion reports whether the regular expression matches the start of
// the operating system version, like Python's re.match does.
func matchOSVersion(pattern string) bool {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return false
	}
	loc := re.FindStringIndex(osVersion())
	return loc != nil && loc[0] == 0
}
