package utils

type ErrorResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

var (
	BadRequestErrResponse          = &ErrorResponse{Status: 400, Message: "Bad Request"}
	UnauthorizedErrResponse        = &ErrorResponse{Status: 401, Message: "Unauthorized"}
	NotFoundErrResponse            = &ErrorResponse{Status: 404, Message: "Not Found"}
	ConflictErrResponse            = &ErrorResponse{Status: 409, Message: "Conflict"}
	InternalServerErrorErrResponse = &ErrorResponse{Status: 500, Message: "Internal Server Error"}
)
