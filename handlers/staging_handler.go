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

	backend.StagingRequestsReceivedCounter().Increment()

	taskRequest, err := backend.BuildRecipe(stagingRequest)
	if err != nil {
		logger.Error("recipe-building-failed", err, lager.Data{"staging-request": stagingRequest})

		resp.WriteHeader(http.StatusInternalServerError)
		response := backend.BuildStagingResponseFromRequestError(stagingRequest, "Recipe building failed: "+err.Error())
		responseJson, _ := json.Marshal(response)
		resp.Write(responseJson)
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

		resp.WriteHeader(http.StatusInternalServerError)
		response := backend.BuildStagingResponseFromRequestError(stagingRequest, "Staging failed: "+err.Error())
		responseJson, _ := json.Marshal(response)
		resp.Write(responseJson)
		return
	}

	resp.WriteHeader(http.StatusAccepted)
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
		logger.Error("unmarshal-request-failed", err)
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	lifecycleBackend, found := handler.backends[stopStagingRequest.Lifecycle]
	if !found {
		logger.Error("backend-not-found", nil, lager.Data{"backend": stopStagingRequest.Lifecycle})
		resp.WriteHeader(http.StatusNotFound)
		return
	}

	if stopStagingRequest.AppId == "" {
		logger.Error("missing-app-id", nil)
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	if stopStagingRequest.TaskId == "" {
		logger.Error("missing-task-id", nil)
		resp.WriteHeader(http.StatusBadRequest)
		return
	}

	resp.WriteHeader(http.StatusAccepted)
	lifecycleBackend.StopStagingRequestsReceivedCounter().Increment()

	taskGuid := backend.StagingTaskGuid(stopStagingRequest.AppId, stopStagingRequest.TaskId)

	logger.Info("cancelling", lager.Data{"task_guid": taskGuid})

	err = handler.diegoClient.CancelTask(taskGuid)
	if err != nil {
		logger.Error("stop-staging-failed", err, lager.Data{"stop-staging-request": requestBody})
	}
}
