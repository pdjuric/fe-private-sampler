package common

const (
	StatusNotFound = "not found"
	StatusCreated  = "created"
	StatusReady    = "ready"
	StatusInvalid  = "invalid"
	StatusError    = "error"
)

const (
	BodyJSON        = "application/json"
	BodyOctetStream = "application/octet-stream"
)

type ResponseType string

const (
	StringResponse ResponseType = "string"
	JSONResponse   ResponseType = "json"
	ErrorResponse  ResponseType = "error"
	DataResponse   ResponseType = "application/octet-stream"
	NoResponse     ResponseType = "no response"
)
