package stager

import "github.com/tedsuo/rata"

const (
	StageRoute            = "Stage"
	StopStagingRoute      = "StopStaging"
	StagingCompletedRoute = "StagingCompleted"
)

var Routes = rata.Routes{
	{Path: "/v1/:staging_task", Method: "POST", Name: StageRoute},
	{Path: "/v1/:staging_task", Method: "DELETE", Name: StopStagingRoute},
	{Path: "/v1/:staging_task/completed", Method: "POST", Name: StagingCompletedRoute},
}
