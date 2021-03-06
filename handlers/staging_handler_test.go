package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"

	"github.com/cloudfoundry-incubator/receptor"
	"github.com/cloudfoundry-incubator/receptor/fake_receptor"
	"github.com/cloudfoundry-incubator/runtime-schema/cc_messages"
	"github.com/cloudfoundry-incubator/stager"
	"github.com/cloudfoundry-incubator/stager/backend"
	"github.com/cloudfoundry-incubator/stager/backend/fake_backend"
	"github.com/cloudfoundry-incubator/stager/cc_client/fakes"
	"github.com/cloudfoundry-incubator/stager/handlers"
	fake_metric_sender "github.com/cloudfoundry/dropsonde/metric_sender/fake"
	"github.com/cloudfoundry/dropsonde/metrics"
	"github.com/pivotal-golang/lager"
	"github.com/pivotal-golang/lager/lagertest"
	"github.com/tedsuo/rata"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("StagingHandler", func() {

	var (
		fakeMetricSender *fake_metric_sender.FakeMetricSender

		logger          lager.Logger
		fakeDiegoClient *fake_receptor.FakeClient
		fakeCcClient    *fakes.FakeCcClient
		fakeBackend     *fake_backend.FakeBackend

		responseRecorder *httptest.ResponseRecorder
		rataHandler      http.Handler
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("test")

		fakeMetricSender = fake_metric_sender.NewFakeMetricSender()
		metrics.Initialize(fakeMetricSender)

		fakeCcClient = &fakes.FakeCcClient{}

		fakeBackend = &fake_backend.FakeBackend{}
		fakeDiegoClient = &fake_receptor.FakeClient{}

		responseRecorder = httptest.NewRecorder()
		handler := handlers.NewStagingHandler(logger, map[string]backend.Backend{"fake-backend": fakeBackend}, fakeCcClient, fakeDiegoClient)

		var routes rata.Routes
		for _, r := range stager.Routes {
			if r.Name == stager.StageRoute {
				routes = append(routes, r)
			}
			if r.Name == stager.StopStagingRoute {
				routes = append(routes, r)
			}
		}

		var err error
		rataHandler, err = rata.NewRouter(routes, rata.Handlers{
			stager.StageRoute:       http.HandlerFunc(handler.Stage),
			stager.StopStagingRoute: http.HandlerFunc(handler.StopStaging),
		})
		Ω(err).ShouldNot(HaveOccurred())
	})

	Describe("Stage", func() {
		var (
			stagingRequestJson []byte
		)

		JustBeforeEach(func() {
			req, err := http.NewRequest("PUT", "/v1/staging/a-staging-guid", bytes.NewReader(stagingRequestJson))
			Ω(err).ShouldNot(HaveOccurred())

			rataHandler.ServeHTTP(responseRecorder, req)
		})

		Context("when a staging request is received for a registered backend", func() {
			var stagingRequest cc_messages.StagingRequestFromCC

			BeforeEach(func() {
				stagingRequest = cc_messages.StagingRequestFromCC{
					AppId:     "myapp",
					Lifecycle: "fake-backend",
				}

				var err error
				stagingRequestJson, err = json.Marshal(stagingRequest)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("increments the counter to track arriving staging messages", func() {
				Ω(fakeMetricSender.GetCounter("StagingStartRequestsReceived")).Should(Equal(uint64(1)))
			})

			It("returns an Accepted response", func() {
				Ω(responseRecorder.Code).Should(Equal(http.StatusAccepted))
			})

			It("builds a staging recipe", func() {
				Ω(fakeBackend.BuildRecipeCallCount()).To(Equal(1))

				guid, request := fakeBackend.BuildRecipeArgsForCall(0)
				Ω(guid).Should(Equal("a-staging-guid"))
				Ω(request).Should(Equal(stagingRequest))
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
					var taskCreateError error

					BeforeEach(func() {
						taskCreateError = errors.New("some task create error")
						fakeDiegoClient.CreateTaskReturns(taskCreateError)
					})

					It("logs the failure", func() {
						Ω(logger).Should(gbytes.Say("staging-failed"))
					})

					It("returns an internal service error status code", func() {
						Ω(responseRecorder.Code).Should(Equal(http.StatusInternalServerError))
					})

					It("should not call staging complete", func() {
						Ω(fakeCcClient.StagingCompleteCallCount()).To(Equal(0))
					})

					Context("when the response builder succeeds", func() {
						var responseForCC cc_messages.StagingResponseForCC

						BeforeEach(func() {
							responseForCC = cc_messages.StagingResponseForCC{
								Error: cc_messages.SanitizeErrorMessage("Staging failed: " + taskCreateError.Error()),
							}
						})

						It("returns the cloud controller error response", func() {
							var response cc_messages.StagingResponseForCC
							decoder := json.NewDecoder(responseRecorder.Body)
							err := decoder.Decode(&response)
							Ω(err).ShouldNot(HaveOccurred())

							Ω(response).Should(Equal(responseForCC))
						})
					})
				})
			})

			Context("when the recipe failed to be built", func() {
				var buildRecipeError error

				BeforeEach(func() {
					buildRecipeError = errors.New("some build recipe error")
					fakeBackend.BuildRecipeReturns(receptor.TaskCreateRequest{}, buildRecipeError)
				})

				It("logs the failure", func() {
					Ω(logger).Should(gbytes.Say("recipe-building-failed"))
				})

				It("returns an internal service error status code", func() {
					Ω(responseRecorder.Code).Should(Equal(http.StatusInternalServerError))
				})

				It("should not call staging complete", func() {
					Ω(fakeCcClient.StagingCompleteCallCount()).To(Equal(0))
				})

				Context("when the response builder succeeds", func() {
					var responseForCC cc_messages.StagingResponseForCC

					BeforeEach(func() {
						responseForCC = cc_messages.StagingResponseForCC{
							Error: cc_messages.SanitizeErrorMessage("Recipe building failed: " + buildRecipeError.Error()),
						}
					})

					It("returns the cloud controller error response", func() {
						var response cc_messages.StagingResponseForCC
						decoder := json.NewDecoder(responseRecorder.Body)
						err := decoder.Decode(&response)
						Ω(err).ShouldNot(HaveOccurred())

						Ω(response).Should(Equal(responseForCC))
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

			Context("when a staging request is received for an unknown backend", func() {
				BeforeEach(func() {
					stagingRequest := cc_messages.StagingRequestFromCC{
						AppId:     "myapp",
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
		BeforeEach(func() {
			stagingTask := receptor.TaskResponse{
				TaskGuid:   "a-staging-guid",
				Annotation: `{"lifecycle": "fake-backend"}`,
			}

			fakeDiegoClient.GetTaskReturns(stagingTask, nil)
		})

		JustBeforeEach(func() {
			req, err := http.NewRequest("DELETE", "/v1/staging/a-staging-guid", nil)
			Ω(err).ShouldNot(HaveOccurred())

			rataHandler.ServeHTTP(responseRecorder, req)
		})

		Context("when receiving a stop staging request", func() {
			It("retrieves the current staging task by guid", func() {
				Ω(fakeDiegoClient.GetTaskCallCount()).Should(Equal(1))
				Ω(fakeDiegoClient.GetTaskArgsForCall(0)).Should(Equal("a-staging-guid"))
			})

			Context("when an in-flight staging task is not found", func() {
				BeforeEach(func() {
					fakeDiegoClient.GetTaskReturns(receptor.TaskResponse{}, receptor.Error{Type: receptor.TaskNotFound})
				})

				It("returns StatusNotFound", func() {
					Ω(responseRecorder.Code).Should(Equal(http.StatusNotFound))
				})
			})

			Context("when retrieving the current task fails", func() {
				BeforeEach(func() {
					fakeDiegoClient.GetTaskReturns(receptor.TaskResponse{}, errors.New("boom"))
				})

				It("returns StatusInternalServerError", func() {
					Ω(responseRecorder.Code).Should(Equal(http.StatusInternalServerError))
				})
			})

			Context("when retrieving the current task is sucessful", func() {
				Context("when the task annotation fails to unmarshal", func() {
					BeforeEach(func() {
						stagingTask := receptor.TaskResponse{
							TaskGuid:   "a-staging-guid",
							Annotation: `,"lifecycle}`,
						}

						fakeDiegoClient.GetTaskReturns(stagingTask, nil)
					})

					It("returns StatusInternalServerError", func() {
						Ω(responseRecorder.Code).Should(Equal(http.StatusInternalServerError))
					})
				})

				It("increments the counter to track arriving stop staging messages", func() {
					Ω(fakeMetricSender.GetCounter("StagingStopRequestsReceived")).Should(Equal(uint64(1)))
				})

				It("cancels the Diego task", func() {
					Ω(fakeDiegoClient.CancelTaskCallCount()).To(Equal(1))
					Ω(fakeDiegoClient.CancelTaskArgsForCall(0)).To(Equal("a-staging-guid"))
				})

				It("returns an Accepted response", func() {
					Ω(responseRecorder.Code).Should(Equal(http.StatusAccepted))
				})

			})
		})
	})
})
