package healthcheck

import (
	"database/sql"
)

const (
	JOINING_STATE        = "1"
	DONOR_DESYNCED_STATE = "2"
	JOINED_STATE         = "3"
	SYNCED_STATE         = "4"
)

type Healthchecker struct {
	db     *sql.DB
	config HealthcheckerConfig
}

type HealthcheckerConfig struct {
	AvailableWhenDonor    bool
	AvailableWhenReadOnly bool
}

type HealthResult struct {
	Healthy bool
}

func New(db *sql.DB, config HealthcheckerConfig) *Healthchecker {
	return &Healthchecker{
		db:     db,
		config: config,
	}
}

var was_joined = false
var old_state = "0"

func (h *Healthchecker) Check() (*HealthResult, string) {
	var variable_name string
	var state string
	err := h.db.QueryRow("SHOW STATUS LIKE 'wsrep_local_state'").Scan(&variable_name, &state)

	var res, msg = &HealthResult{Healthy: false}, "not synced"
	switch {
	case err != nil:
		res, msg = &HealthResult{Healthy: false}, err.Error()
	case state != SYNCED_STATE && !was_joined:
		if old_state == JOINED_STATE && state != JOINED_STATE {
			res, msg = &HealthResult{Healthy: false}, "no synced"
			was_joined = true
		} else {
			res, msg = nil, "syncing"
		}
	case state == SYNCED_STATE || (state == DONOR_DESYNCED_STATE && h.config.AvailableWhenDonor):
		was_joined = true
		res, msg = &HealthResult{Healthy: true}, "synced"
		if !h.config.AvailableWhenReadOnly {
			var ro_variable_name string
			var ro_value string
			ro_err := h.db.QueryRow("SHOW GLOBAL VARIABLES LIKE 'read_only'").Scan(&ro_variable_name, &ro_value)
			switch {
			case ro_err != nil:
				res, msg = &HealthResult{Healthy: false}, ro_err.Error()
			case ro_value == "ON":
				res, msg = &HealthResult{Healthy: false}, "read-only"
			}
		}
	}
	
	old_state = state
	return res, msg
}
