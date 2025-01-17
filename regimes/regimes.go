package regimes

import (
	// Import all the regime definitions which will automatically
	// add themselves to the tax regime register.
	_ "github.com/invopop/gobl/regimes/co"
	_ "github.com/invopop/gobl/regimes/es"
	_ "github.com/invopop/gobl/regimes/fr"
	_ "github.com/invopop/gobl/regimes/gb"
	_ "github.com/invopop/gobl/regimes/nl"
	_ "github.com/invopop/gobl/regimes/pt"
)
