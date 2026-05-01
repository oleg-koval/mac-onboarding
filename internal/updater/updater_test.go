package updater

import "testing"

func TestShouldSkip(t *testing.T) {
	t.Setenv(envAutoUpdateDone, "")
	t.Setenv(envAutoUpdate, "")

	if shouldSkip("v0.2.4") {
		t.Fatal("expected released version to allow autoupdate")
	}

	t.Setenv(envAutoUpdateDone, "1")
	if !shouldSkip("v0.2.4") {
		t.Fatal("expected re-exec guard to skip")
	}

	t.Setenv(envAutoUpdateDone, "")
	t.Setenv(envAutoUpdate, "0")
	if !shouldSkip("v0.2.4") {
		t.Fatal("expected opt-out to skip")
	}

	t.Setenv(envAutoUpdate, "")
	if !shouldSkip("dev") {
		t.Fatal("expected dev version to skip")
	}
}

func TestBrewStableExecutable(t *testing.T) {
	tests := []struct {
		path string
		want string
		ok   bool
	}{
		{
			path: "/opt/homebrew/Cellar/mac-onboarding/0.2.4/bin/mac-onboarding",
			want: "/opt/homebrew/bin/mac-onboarding",
			ok:   true,
		},
		{
			path: "/usr/local/Caskroom/mac-onboarding/0.2.4/mac-onboarding",
			want: "/usr/local/bin/mac-onboarding",
			ok:   true,
		},
		{
			path: "/tmp/mac-onboarding",
			want: "",
			ok:   false,
		},
	}

	for _, tt := range tests {
		got, ok := brewStableExecutable(tt.path)
		if got != tt.want || ok != tt.ok {
			t.Fatalf("brewStableExecutable(%q) = (%q, %v), want (%q, %v)", tt.path, got, ok, tt.want, tt.ok)
		}
	}
}

func TestIsOutdated(t *testing.T) {
	if !isOutdated([]byte("mac-onboarding\n")) {
		t.Fatal("expected formula to be detected as outdated")
	}
	if isOutdated([]byte("other-formula\n")) {
		t.Fatal("did not expect unrelated formula to match")
	}
}
