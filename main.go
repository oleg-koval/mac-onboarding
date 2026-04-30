package main

import (
	"github.com/oleg-koval/mac-onboarding/cmd"

	// Register modules via side-effect imports.
	_ "github.com/oleg-koval/mac-onboarding/internal/modules/ai_tools"
	_ "github.com/oleg-koval/mac-onboarding/internal/modules/bootstrap"
	_ "github.com/oleg-koval/mac-onboarding/internal/modules/brew"
	_ "github.com/oleg-koval/mac-onboarding/internal/modules/cursor"
	_ "github.com/oleg-koval/mac-onboarding/internal/modules/git"
	_ "github.com/oleg-koval/mac-onboarding/internal/modules/hotkeys"
	_ "github.com/oleg-koval/mac-onboarding/internal/modules/kitty"
	_ "github.com/oleg-koval/mac-onboarding/internal/modules/prefs"
	_ "github.com/oleg-koval/mac-onboarding/internal/modules/shell"
	_ "github.com/oleg-koval/mac-onboarding/internal/modules/system"
)

var Version = "dev"

func main() {
	cmd.Execute()
}
