package db

import (
	"fmt"

	"taskmanager/internal/repository"
)

func (c *Connection) initRepositories() error {
	if c == nil || c.conn == nil {
		return fmt.Errorf("database not initialized")
	}

	c.Alarm = repository.NewAlarmRepository(c.conn)
	c.CGNAT = repository.NewCGNATRepository(c.conn, c.cfg.SiteName)
	c.Session = repository.NewSessionRepository(c.conn)
	c.User = repository.NewUserRepository(c.conn)
	c.Whitelist = repository.NewWhitelistRepository(c.conn)

	return nil
}
