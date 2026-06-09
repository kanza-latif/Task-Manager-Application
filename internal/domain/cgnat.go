package domain

// CGNAT represents a row in cgnat_table.
// private_ip and public_ip are stored as VARBINARY(16) in MySQL,
// retrieved here as []byte and converted to string for display.
type CGNAT struct {
	ID        int64
	PrivateIP string
	PublicIP  string
	StartPort int
	EndPort   int
}
