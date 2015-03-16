package handlers

import (
	"net/http"

	"github.com/cloudfoundry-incubator/receptor"
	"github.com/cloudfoundry-incubator/stager/backend"
	"github.com/pivotal-golang/lager"
)

type StagingHandler interface {
	Stage(resp http.ResponseWriter, req *http.Request)
	StopStaging(resp http.ResponseWriter, req *http.Request)
}

type stagingHandler struct {
	diegoClient receptor.Client
	logger      lager.Logger
	backends    []backend.Backend
}

func NewStagingHandler(logger lager.Logger, diegoClient receptor.Client, backends []backend.Backend) StagingHandler {
	logger = logger.Session("staging-handler") //, lager.Data{"TaskDomain": backend.TaskDomain()})

	return &stagingHandler{
		diegoClient: diegoClient,
		logger:      logger,
		backends:    backends,
	}
}

func (handler *stagingHandler) Stage(resp http.ResponseWriter, req *http.Request) {
}

func (handler *stagingHandler) StopStaging(resp http.ResponseWriter, req *http.Request) {
}
