package entity

type LUNInfo struct {
	LUN    uint32 `json:"lun"`    // Logical Unit Number.
	WWID   string `json:"wwid"`   // World Wide Name (identifier).
	Size   uint64 `json:"size"`   // Size of the LUN in bytes.
	Errors string `json:"errors"` // Error occurred when LUN was fetched.
}
