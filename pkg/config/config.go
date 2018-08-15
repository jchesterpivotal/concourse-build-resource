package config

type Source struct {
	ConcourseUrl string `json:"concourse_url"`
	Team         string `json:"team"`
	Pipeline     string `json:"pipeline"`
	Job          string `json:"job"`
}

type Version struct {
	BuildId string `json:"build_id"`
}

type InParams struct{}

type VersionMetadataField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type InRequest struct {
	Source           Source   `json:"source"`
	Version          Version  `json:"version"`
	Params           InParams `json:"params,omitempty"`
	WorkingDirectory string   `json:"working_directory,omitempty"`
}

type InResponse struct {
	Version  Version                `json:"version"`
	Metadata []VersionMetadataField `json:"metadata"`
}

type CheckRequest struct {
	Source  Source  `json:"source"`
	Version Version `json:"version,omitempty"`
}

type CheckResponse []Version
