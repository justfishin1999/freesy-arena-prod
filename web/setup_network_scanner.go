package web

import (
	"encoding/json"
	"net/http"

	"github.com/Team254/cheesy-arena/model"
	"github.com/Team254/cheesy-arena/network"
)

// Renders the Network Scanner page.
func (web *Web) networkScannerSettingsHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := web.parseFiles(
		"templates/network_scanner.html",
		"templates/base.html",
	)
	if err != nil {
		handleWebErr(w, err)
		return
	}

	// Grab the singleton Manager and check if it's currently enabled/running.
	mgr := network.GetManager()

	// Create a data struct that embeds EventSettings plus ScannerRunning.
	data := struct {
		*model.EventSettings
		ScannerRunning bool
	}{
		EventSettings:  web.arena.EventSettings,
		ScannerRunning: mgr.Enabled,
	}

	if err := tmpl.ExecuteTemplate(w, "base", data); err != nil {
		handleWebErr(w, err)
	}
}

// Serves JSON of detected devices.
func (web *Web) networkDevicesHandler(w http.ResponseWriter, r *http.Request) {
	manager := network.GetManager()
	if !manager.Enabled {
		http.Error(w, "Scanner disabled", http.StatusServiceUnavailable)
		return
	}
	json.NewEncoder(w).Encode(manager.GetDevices())
}

// Marks a device (by IP) as known (removes Rogue flag). Expects "ip" in POST form.
func (web *Web) networkToggleDeviceHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	ip := r.FormValue("ip")
	if ip == "" {
		http.Error(w, "Missing ip parameter", http.StatusBadRequest)
		return
	}
	manager := network.GetManager()
	manager.MarkAsKnown(ip)
	w.WriteHeader(http.StatusOK)
}
