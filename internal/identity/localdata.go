package identity

// SealLocalData uses the same OS protection as identity storage without exposing its key.
// Windows: current-user DPAPI. Non-Windows callers must still enforce private file permissions.
func SealLocalData(data []byte) ([]byte, error)   { return seal(data) }
func UnsealLocalData(data []byte) ([]byte, error) { return unseal(data) }
