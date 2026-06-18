package domain

type Whitelist struct {
	ID     int64
	MSISDN string
	Status bool
	Delete bool
}

func NewWhitelist() Whitelist {
	return Whitelist{
		Status: true,
		Delete: false,
	}
}
