package mcl_test

import (
	"testing"

	"github.com/syaniclauncher/minecraft-lib-golang/mcl"
)

func TestGetModLoader(t *testing.T) {
	for _, id := range mcl.ListModLoaders() {
		loader, err := mcl.GetModLoader(id)
		if err != nil {
			t.Fatalf("GetModLoader(%q): %v", id, err)
		}
		if loader.ID() != id {
			t.Errorf("the loader %q reports the id %q", id, loader.ID())
		}
		if loader.Name() == "" {
			t.Errorf("the loader %q has no name", id)
		}
	}

	if _, err := mcl.GetModLoader("does-not-exist"); err == nil {
		t.Error("GetModLoader should fail for an unknown id")
	}
}

func TestModLoaderInstalledVersions(t *testing.T) {
	tests := []struct {
		loader        mcl.ModLoader
		minecraft     string
		loaderVersion string
		want          string
	}{
		{mcl.Fabric, "1.20.1", "0.15.0", "fabric-loader-0.15.0-1.20.1"},
		{mcl.Quilt, "1.20.1", "0.20.0", "quilt-loader-0.20.0-1.20.1"},
		{mcl.Forge, "1.20.1", "47.1.0", "1.20.1-forge-47.1.0"},
		{mcl.NeoForge, "1.20.1", "47.1.0", "neoforge-47.1.0"},
	}

	for _, test := range tests {
		got := test.loader.InstalledVersion(test.minecraft, test.loaderVersion)
		if got != test.want {
			t.Errorf("%s is installed under %q, want %q", test.loader.ID(), got, test.want)
		}
	}
}
