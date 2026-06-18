package db

import (
	"log"

	"taskmanager/internal/domain"
)

// InitBootstrap loads CGNAT + Whitelist ONCE and returns them.
func InitBootstrap() ([]*domain.CGNAT, []*domain.Whitelist, error) {

	if GlobalConn.cfg.Verbosity > 1 {
		log.Printf("Loading CGNAT and Whitelisting information from database.")
	}

	if GlobalConn == nil {
		return nil, nil, nil
	}

	cgnatList, err := GlobalConn.CGNAT.List()
	if err != nil {
		return nil, nil, err
	}

	whitelistList, err := GlobalConn.Whitelist.List()
	if err != nil {
		return nil, nil, err
	}

	return cgnatList, whitelistList, nil
}
