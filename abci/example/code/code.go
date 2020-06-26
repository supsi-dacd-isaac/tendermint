package code

// Return codes for the examples
const (
	CodeTypeOK            uint32 = 0
	CodeTypeEncodingError uint32 = 1
	CodeTypeBadNonce      uint32 = 2
	CodeTypeUnauthorized  uint32 = 3
	CodeTypeBadRequest    uint32 = 5
	CodeNotPositiveAmount uint32 = 7
	CodeExceedingAmount   uint32 = 8
)
