package stager

import "github.com/tedsuo/rata"

const (
	StageRoute            = "Stage"
	StopStagingRoute      = "StopStaging"
	StagingCompletedRoute = "StagingCompleted"
)

var Routes = rata.Routes{
	{Path: "/v1/start", Method: "POST", Name: StageRoute},
	{Path: "/v1/stop", Method: "DELETE", Name: StopStagingRoute},
	{Path: "/v1/completed", Method: "POST", Name: StagingCompletedRoute},
}
