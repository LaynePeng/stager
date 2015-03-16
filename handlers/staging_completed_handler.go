package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/cloudfoundry-incubator/receptor"
	"github.com/cloudfoundry-incubator/runtime-schema/metric"
	"github.com/cloudfoundry-incubator/stager/backend"
	"github.com/cloudfoundry-incubator/stager/cc_client"
	"github.com/pivotal-golang/clock"
	"github.com/pivotal-golang/lager"
)

const (
	// Metrics
	stagingSuccessCounter  = metric.Counter("StagingRequestsSucceeded")
	stagingSuccessDuration = metric.Duration("StagingRequestSucceededDuration")
	stagingFailureCounter  = metric.Counter("StagingRequestsFailed")
	stagingFailureDuration = metric.Duration("StagingRequestFailedDuration")
)

type CompletionHandler interface {
	StagingComplete(resp http.ResponseWriter, req *http.Request)
}

type completionHandler struct {
	ccClient cc_client.CcClient
	backends []backend.Backend
	logger   lager.Logger
	clock    clock.Clock
}

func NewStagingCompletionHandler(logger lager.Logger, ccClient cc_client.CcClient, backends []backend.Backend, clock clock.Clock) CompletionHandler {
	return &completionHandler{
		ccClient: ccClient,
		backends: backends,
		logger:   logger.Session("completion-handler"),
		clock:    clock,
	}
}

func (handler *completionHandler) StagingComplete(res http.ResponseWriter, req *http.Request) {
	var task receptor.TaskResponse
	err := json.NewDecoder(req.Body).Decode(&task)
	if err != nil {
		handler.logger.Error("parsing-incoming-task-failed", err)
		res.WriteHeader(http.StatusBadRequest)
		return
	}

	logger := handler.logger.Session("task-complete-callback-received", lager.Data{
		"guid": task.TaskGuid,
	})

	responseJson, err := handler.stagingResponse(task)
	if err != nil {
		res.WriteHeader(http.StatusBadRequest)
		logger.Error("get-staging-response-failed", err)
		return
	}

	if responseJson == nil {
		res.WriteHeader(http.StatusNotFound)
		res.Write([]byte("Unknown task domain"))
		return
	}

	logger.Info("posting-staging-complete", lager.Data{
		"payload": responseJson,
	})

	err = handler.ccClient.StagingComplete(responseJson, logger)
	if err != nil {
		logger.Error("cc-staging-complete-failed", err)
		if responseErr, ok := err.(*cc_client.BadResponseError); ok {
			res.WriteHeader(responseErr.StatusCode)
		} else {
			res.WriteHeader(http.StatusServiceUnavailable)
		}
		return
	}

	handler.reportMetrics(task)

	logger.Info("posted-staging-complete")
	res.WriteHeader(http.StatusOK)
}

func (handler *completionHandler) reportMetrics(task receptor.TaskResponse) {
	duration := handler.clock.Now().Sub(time.Unix(0, task.CreatedAt))
	if task.Failed {
		stagingFailureCounter.Increment()
		stagingFailureDuration.Send(duration)
	} else {
		stagingSuccessDuration.Send(duration)
		stagingSuccessCounter.Increment()
	}
}

func (handler *completionHandler) stagingResponse(task receptor.TaskResponse) ([]byte, error) {
	for _, backend := range handler.backends {
		if backend.TaskDomain() == task.Domain {
			return backend.BuildStagingResponse(task)
		}
	}

	return nil, nil
}
