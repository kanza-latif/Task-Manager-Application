package rabbitmq

import (
	"fmt"

	"github.com/go-resty/resty/v2"
)

func NewAdmin(cfg Config) error {
	GlobalClient = &Client{
		cfg: cfg,
		admin: Admin{
			host: cfg.Host,
			client: resty.New().
				SetDisableWarn(true).
				SetBasicAuth(cfg.AdminUser, cfg.AdminPassword).
				SetHeader("Content-Type", "application/json"),
		},
	}
	return nil
}

func EnsureVHost(vhost string) error {

	url := fmt.Sprintf(
		"http://%s:15672/api/vhosts/%s",
		GlobalClient.admin.host,
		vhost,
	)

	resp, err := GlobalClient.admin.client.
		SetBasicAuth(GlobalClient.cfg.User, GlobalClient.cfg.Password).
		R().
		SetBody(map[string]any{}).
		Put(url)

	if err != nil {
		return err
	}

	if resp.StatusCode() >= 300 {
		return fmt.Errorf("vhost error: %s", resp.String())
	}

	return nil
}

func EnsureUser(user, pass string) error {

	url := fmt.Sprintf(
		"http://%s:15672/api/users/%s",
		GlobalClient.admin.host,
		user,
	)

	resp, err := GlobalClient.admin.client.R().
		SetBody(map[string]any{
			"password": pass,
			"tags":     "administrator",
		}).
		Put(url)

	if err != nil {
		return err
	}

	if resp.StatusCode() >= 300 {
		return fmt.Errorf("user error: %s", resp.String())
	}

	return nil
}

func EnsurePermissions(vhost, user string) error {

	url := fmt.Sprintf(
		"http://%s:15672/api/permissions/%s/%s",
		GlobalClient.admin.host,
		vhost,
		user,
	)

	resp, err := GlobalClient.admin.client.R().
		SetBody(map[string]any{
			"configure": ".*",
			"write":     ".*",
			"read":      ".*",
		}).
		Put(url)

	if err != nil {
		return err
	}

	if resp.StatusCode() >= 300 {
		return fmt.Errorf("permissions error: %s", resp.String())
	}

	return nil
}

func Bootstrap() error {

	if err := EnsureUser(GlobalClient.cfg.User, GlobalClient.cfg.Password); err != nil {
		return err
	}

	if err := EnsureVHost(GlobalClient.cfg.Vhost); err != nil {
		return err
	}

	if err := EnsurePermissions(GlobalClient.cfg.Vhost, GlobalClient.cfg.User); err != nil {
		return err
	}

	err := New()

	return err
}
