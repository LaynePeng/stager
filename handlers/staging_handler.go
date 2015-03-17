package handlers

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/cloudfoundry-incubator/receptor"
	"github.com/cloudfoundry-incubator/runtime-schema/cc_messages"
	"github.com/cloudfoundry-incubator/stager/backend"
	"github.com/cloudfoundry-incubator/stager/cc_client"
	"github.com/pivotal-golang/lager"
)

type StagingHandler interface {
	Stage(resp http.ResponseWriter, req *http.Request)
	StopStaging(resp http.ResponseWriter, req *http.Request)
}

type stagingHandler struct {
	logger      lager.Logger
	backends    map[string]backend.Backend
	ccClient    cc_client.CcClient
	diegoClient receptor.Client
}

func NewStagingHandler(
	logger lager.Logger,
	backends map[string]backend.Backend,
	ccClient cc_client.CcClient,
	diegoClient receptor.Client,
) StagingHandler {
	logger = logger.Session("staging-handler")

	return &stagingHandler{
		logger:      logger,
		backends:    backends,
		ccClient:    ccClient,
		diegoClient: diegoClient,
	}
}

func (handler *stagingHandler) Stage(resp http.ResponseWriter, req *http.Request) {
	logger := handler.logger.Session("staging-request")

	requestBody, err := ioutil.ReadAll(req.Body)
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	var stagingRequest cc_messages.StagingRequestFromCC
	err = json.Unmarshal(requestBody, &stagingRequest)
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	backend, ok := handler.backends[stagingRequest.Lifecycle]
	if !ok {
		resp.WriteHeader(http.StatusNotFound)
		return
	}

	if stagingRequest.AppId == "" {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	if stagingRequest.TaskId == "" {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	resp.WriteHeader(http.StatusAccepted)
	backend.StagingRequestsReceivedCounter().Increment()

	taskRequest, err := backend.BuildRecipe(stagingRequest)
	if err != nil {
		logger.Error("recipe-building-failed", err, lager.Data{"staging-request": stagingRequest})
		handler.sendStagingCompleteError(logger, backend, "Recipe building failed: ", err, stagingRequest)
		return
	}

	logger.Info("desiring-task", lager.Data{
		"task_guid":    taskRequest.TaskGuid,
		"callback_url": taskRequest.CompletionCallbackURL,
	})
	err = handler.diegoClient.CreateTask(taskRequest)
	if receptorErr, ok := err.(receptor.Error); ok {
		if receptorErr.Type == receptor.TaskGuidAlreadyExists {
			err = nil
		}
	}

	if err != nil {
		logger.Error("staging-failed", err, lager.Data{"staging-request": stagingRequest})
		handler.sendStagingCompleteError(logger, backend, "Staging failed: ", err, stagingRequest)
	}
}

func (handler *stagingHandler) StopStaging(resp http.ResponseWriter, req *http.Request) {
	logger := handler.logger.Session("stop-staging-request")

	requestBody, err := ioutil.ReadAll(req.Body)
	if err != nil {
		resp.WriteHeader(http.StatusInternalServerError)
		return
	}

	var stopStagingRequest cc_messages.StopStagingRequestFromCC
	err = json.Unmarshal(requestBody, &stopStagingRequest)
	if err != nil {
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	backend, ok := handler.backends[stopStagingRequest.Lifecycle]
	if !ok {
		resp.WriteHeader(http.StatusNotFound)
		return
	}

	resp.WriteHeader(http.StatusAccepted)
	backend.StopStagingRequestsReceivedCounter().Increment()

	taskGuid, err := backend.StagingTaskGuid(stopStagingRequest)
	if err != nil {
		logger.Error("staging-task-guid-failed", err, lager.Data{"stop-staging-request": requestBody})
		return
	}

	logger.Info("cancelling", lager.Data{"task_guid": taskGuid})

	err = handler.diegoClient.CancelTask(taskGuid)
	if err != nil {
		logger.Error("stop-staging-failed", err, lager.Data{"stop-staging-request": requestBody})
	}
}

func (handler *stagingHandler) sendStagingCompleteError(logger lager.Logger, backend backend.Backend, messagePrefix string, err error, stagingRequest cc_messages.StagingRequestFromCC) {
	response := backend.BuildStagingResponseFromRequestError(stagingRequest, messagePrefix+err.Error())
	responseJson, err := json.Marshal(response)
	if err != nil {
		logger.Error("marshalling-build-staging-response-failed", err)
		return
	}

	handler.ccClient.StagingComplete(responseJson, logger)
}
