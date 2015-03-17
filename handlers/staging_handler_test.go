package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/cloudfoundry-incubator/receptor"
	"github.com/cloudfoundry-incubator/receptor/fake_receptor"
	"github.com/cloudfoundry-incubator/runtime-schema/cc_messages"
	"github.com/cloudfoundry-incubator/runtime-schema/metric"
	"github.com/cloudfoundry-incubator/stager"
	"github.com/cloudfoundry-incubator/stager/backend"
	"github.com/cloudfoundry-incubator/stager/backend/fake_backend"
	"github.com/cloudfoundry-incubator/stager/cc_client/fakes"
	"github.com/cloudfoundry-incubator/stager/handlers"
	fake_metric_sender "github.com/cloudfoundry/dropsonde/metric_sender/fake"
	"github.com/cloudfoundry/dropsonde/metrics"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("StagingHandler", func() {
	const (
		FakeStagingRequestsReceivedCounter     = "FakeStageMetricName"
		FakeStopStagingRequestsReceivedCounter = "FakeStopMetricName"
	)

	var (
		fakeMetricSender *fake_metric_sender.FakeMetricSender

		logger          lager.Logger
		fakeDiegoClient *fake_receptor.FakeClient
		fakeCcClient    *fakes.FakeCcClient
		fakeBackend     *fake_backend.FakeBackend

		handler          handlers.StagingHandler
		responseRecorder *httptest.ResponseRecorder
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")

		fakeMetricSender = fake_metric_sender.NewFakeMetricSender()
		metrics.Initialize(fakeMetricSender)

		fakeCcClient = &fakes.FakeCcClient{}

		fakeBackend = &fake_backend.FakeBackend{}
		fakeBackend.StagingRequestsReceivedCounterReturns(metric.Counter(FakeStagingRequestsReceivedCounter))
		fakeBackend.StopStagingRequestsReceivedCounterReturns(metric.Counter(FakeStopStagingRequestsReceivedCounter))
		fakeDiegoClient = &fake_receptor.FakeClient{}

		handler = handlers.NewStagingHandler(logger, map[string]backend.Backend{"fake-backend": fakeBackend}, fakeCcClient, fakeDiegoClient)
		responseRecorder = httptest.NewRecorder()
	})

	Describe("Stage", func() {
		var (
			stagingRequestJson []byte
		)

		JustBeforeEach(func() {
			req, err := http.NewRequest("POST", stager.StageRoute, bytes.NewReader(stagingRequestJson))
			Ω(err).ShouldNot(HaveOccurred())

			handler.Stage(responseRecorder, req)
		})

		Context("when a staging request is received for a registered backend", func() {
			var stagingRequest cc_messages.StagingRequestFromCC

			BeforeEach(func() {
				stagingRequest = cc_messages.StagingRequestFromCC{
					AppId:     "myapp",
					TaskId:    "mytask",
					Lifecycle: "fake-backend",
				}

				var err error
				stagingRequestJson, err = json.Marshal(stagingRequest)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("increments the counter to track arriving staging messages", func() {
				Ω(fakeMetricSender.GetCounter(FakeStagingRequestsReceivedCounter)).Should(Equal(uint64(1)))
			})

			It("returns an Accepted response", func() {
				Ω(responseRecorder.Code).Should(Equal(http.StatusAccepted))
			})

			It("builds a staging recipe", func() {
				Ω(fakeBackend.BuildRecipeCallCount()).To(Equal(1))
				Ω(fakeBackend.BuildRecipeArgsForCall(0)).To(Equal(stagingRequest))
			})

			Context("when the recipe was built successfully", func() {
				var fakeTaskRequest = receptor.TaskCreateRequest{Annotation: "test annotation"}
				BeforeEach(func() {
					fakeBackend.BuildRecipeReturns(fakeTaskRequest, nil)
				})

				It("does not send a staging complete message", func() {
					Ω(fakeCcClient.StagingCompleteCallCount()).To(Equal(0))
				})

				It("creates a task on Diego", func() {
					Ω(fakeDiegoClient.CreateTaskCallCount()).To(Equal(1))
					Ω(fakeDiegoClient.CreateTaskArgsForCall(0)).To(Equal(fakeTaskRequest))
				})

				Context("when creating the task succeeds", func() {
					It("does not send a staging failure response", func() {
						Ω(fakeCcClient.StagingCompleteCallCount()).To(Equal(0))
					})
				})

				Context("when the task has already been created", func() {
					BeforeEach(func() {
						fakeDiegoClient.CreateTaskReturns(receptor.Error{
							Type:    receptor.TaskGuidAlreadyExists,
							Message: "ok, this task already exists",
						})
					})

					It("does not log a failure", func() {
						Ω(logger).ShouldNot(gbytes.Say("staging-failed"))
					})
				})

				Context("create task fails for any other reason", func() {
					BeforeEach(func() {
						fakeDiegoClient.CreateTaskReturns(errors.New("create task error"))
					})

					It("logs the failure", func() {
						Ω(logger).Should(gbytes.Say("staging-failed"))
					})

					It("sends a staging failure response", func() {
						Ω(fakeCcClient.StagingCompleteCallCount()).To(Equal(1))
					})
				})
			})

			Context("when the recipe failed to be built", func() {
				BeforeEach(func() {
					fakeBackend.BuildRecipeReturns(receptor.TaskCreateRequest{}, errors.New("fake error"))
				})

				It("logs the failure", func() {
					Ω(logger).Should(gbytes.Say("recipe-building-failed"))
				})

				Context("when the response builder succeeds", func() {
					BeforeEach(func() {
						responseForCC := cc_messages.StagingResponseForCC{Error: &cc_messages.StagingError{Message: "some fake error"}}
						fakeBackend.BuildStagingResponseFromRequestErrorReturns(responseForCC)
					})

					It("sends a staging failure response", func() {
						Ω(fakeCcClient.StagingCompleteCallCount()).To(Equal(1))
						response, _ := fakeCcClient.StagingCompleteArgsForCall(0)

						stagingResponse := cc_messages.StagingResponseForCC{}
						json.Unmarshal(response, &stagingResponse)
						Ω(stagingResponse.Error).Should(Equal(&cc_messages.StagingError{Message: "some fake error"}))
					})
				})
			})
		})

		Describe("bad requests", func() {
			Context("when the request fails to unmarshal", func() {
				BeforeEach(func() {
					stagingRequestJson = []byte(`bad-json`)
				})

				It("returns bad request", func() {
					Ω(responseRecorder.Code).Should(Equal(http.StatusBadRequest))
				})

				It("does not send a staging complete message", func() {
					Ω(fakeCcClient.StagingCompleteCallCount()).To(Equal(0))
				})
			})

			Context("when the app id is missing", func() {
				BeforeEach(func() {
					stagingRequest := cc_messages.StagingRequestFromCC{
						TaskId:    "mytask",
						Lifecycle: "fake-backend",
					}

					var err error
					stagingRequestJson, err = json.Marshal(stagingRequest)
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("returns bad request", func() {
					Ω(responseRecorder.Code).Should(Equal(http.StatusBadRequest))
				})

				It("does not send a staging complete message", func() {
					Ω(fakeCcClient.StagingCompleteCallCount()).To(Equal(0))
				})
			})

			Context("when the task id is missing", func() {
				BeforeEach(func() {
					stagingRequest := cc_messages.StagingRequestFromCC{
						AppId:     "myapp",
						Lifecycle: "fake-backend",
					}

					var err error
					stagingRequestJson, err = json.Marshal(stagingRequest)
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("returns bad request", func() {
					Ω(responseRecorder.Code).Should(Equal(http.StatusBadRequest))
				})

				It("does not send a staging complete message", func() {
					Ω(fakeCcClient.StagingCompleteCallCount()).To(Equal(0))
				})
			})

			Context("when a staging request is received for an unknown backend", func() {
				BeforeEach(func() {
					stagingRequest := cc_messages.StagingRequestFromCC{
						AppId:     "myapp",
						TaskId:    "mytask",
						Lifecycle: "unknown-backend",
					}

					var err error
					stagingRequestJson, err = json.Marshal(stagingRequest)
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("returns a Not Found response", func() {
					Ω(responseRecorder.Code).Should(Equal(http.StatusNotFound))
				})
			})

			Context("when a malformed staging request is received", func() {
				BeforeEach(func() {
					stagingRequestJson = []byte(`bogus-request`)
				})

				It("returns a BadRequest error", func() {
					Ω(responseRecorder.Code).Should(Equal(http.StatusBadRequest))
				})
			})
		})
	})

	Describe("StopStaging", func() {
		var stopStagingRequestJson []byte

		JustBeforeEach(func() {
			req, err := http.NewRequest("POST", stager.StopStagingRoute, bytes.NewReader(stopStagingRequestJson))
			Ω(err).ShouldNot(HaveOccurred())

			handler.StopStaging(responseRecorder, req)
		})

		Context("when receiving a stop staging request for a registered backend", func() {
			var stopStagingRequest cc_messages.StopStagingRequestFromCC

			BeforeEach(func() {
				stopStagingRequest = cc_messages.StopStagingRequestFromCC{
					AppId:     "myapp",
					TaskId:    "mytask",
					Lifecycle: "fake-backend",
				}

				var err error
				stopStagingRequestJson, err = json.Marshal(stopStagingRequest)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("increments the counter to track arriving stop staging messages", func() {
				Ω(fakeMetricSender.GetCounter(FakeStopStagingRequestsReceivedCounter)).Should(Equal(uint64(1)))
			})

			It("returns an Accepted response", func() {
				Ω(responseRecorder.Code).Should(Equal(http.StatusAccepted))
			})

			Context("when the task guid was built successfully", func() {
				var expectedTaskId string

				BeforeEach(func() {
					expectedTaskId = fmt.Sprintf("%s-%s", stopStagingRequest.AppId, stopStagingRequest.TaskId)
				})

				It("cancels a task on Diego", func() {
					Ω(fakeDiegoClient.CancelTaskCallCount()).To(Equal(1))
					Ω(fakeDiegoClient.CancelTaskArgsForCall(0)).To(Equal(expectedTaskId))
				})
			})
		})

		Describe("bad requests", func() {
			Context("when the request fails to unmarshal", func() {
				BeforeEach(func() {
					stopStagingRequestJson = []byte(`bad-json`)
				})

				It("returns bad request", func() {
					Ω(responseRecorder.Code).Should(Equal(http.StatusBadRequest))
				})

				It("does not send a staging complete message", func() {
					Ω(fakeCcClient.StagingCompleteCallCount()).To(Equal(0))
				})
			})

			Context("when the app id is missing", func() {
				BeforeEach(func() {
					stopStagingRequest := cc_messages.StopStagingRequestFromCC{
						TaskId:    "mytask",
						Lifecycle: "fake-backend",
					}

					var err error
					stopStagingRequestJson, err = json.Marshal(stopStagingRequest)
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("returns bad request", func() {
					Ω(responseRecorder.Code).Should(Equal(http.StatusBadRequest))
				})

				It("does not send a staging complete message", func() {
					Ω(fakeCcClient.StagingCompleteCallCount()).To(Equal(0))
				})
			})

			Context("when the task id is missing", func() {
				BeforeEach(func() {
					stopStagingRequest := cc_messages.StopStagingRequestFromCC{
						AppId:     "myapp",
						Lifecycle: "fake-backend",
					}

					var err error
					stopStagingRequestJson, err = json.Marshal(stopStagingRequest)
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("returns bad request", func() {
					Ω(responseRecorder.Code).Should(Equal(http.StatusBadRequest))
				})

				It("does not send a staging complete message", func() {
					Ω(fakeCcClient.StagingCompleteCallCount()).To(Equal(0))
				})
			})

			Context("when a stop staging request is received for an unknown backend", func() {
				BeforeEach(func() {
					stagingRequest := cc_messages.StopStagingRequestFromCC{
						AppId:     "myapp",
						TaskId:    "mytask",
						Lifecycle: "unknown-backend",
					}

					var err error
					stopStagingRequestJson, err = json.Marshal(stagingRequest)
					Ω(err).ShouldNot(HaveOccurred())
				})

				It("returns a Not Found response", func() {
					Ω(responseRecorder.Code).Should(Equal(http.StatusNotFound))
				})
			})

			Context("when a malformed stop staging request is received", func() {
				BeforeEach(func() {
					stopStagingRequestJson = []byte(`bogus-request`)
				})

				It("returns a BadRequest error", func() {
					Ω(responseRecorder.Code).Should(Equal(http.StatusBadRequest))
				})
			})
		})
	})
})
