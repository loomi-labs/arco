package borg

// FrontendError is the error type that is received from the frontend
type FrontendError struct {
	Message string `json:"message"`
	Stack   string `json:"stack"`
}

// -------- Client types --------

type Archive struct {
	Archive  string `json:"archive"`
	Barchive string `json:"barchive"`
	ID       string `json:"id"`
	Name     string `json:"name"`
	Start    string `json:"start"`
	Time     string `json:"time"`
}

type Encryption struct {
	Mode string `json:"mode"`
}

type Repository struct {
	ID           string `json:"id"`
	LastModified string `json:"last_modified"`
	Location     string `json:"location"`
}

type ListResponse struct {
	Archives   []Archive  `json:"archives"`
	Encryption Encryption `json:"encryption"`
	Repository Repository `json:"repository"`
}
