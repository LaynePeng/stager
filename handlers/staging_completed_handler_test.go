package handlers_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/cloudfoundry-incubator/receptor"
	"github.com/cloudfoundry-incubator/runtime-schema/cc_messages"
	"github.com/cloudfoundry-incubator/runtime-schema/models"
	"github.com/cloudfoundry-incubator/stager"
	"github.com/cloudfoundry-incubator/stager/backend"
	"github.com/cloudfoundry-incubator/stager/backend/fake_backend"
	"github.com/cloudfoundry-incubator/stager/cc_client"
	"github.com/cloudfoundry-incubator/stager/cc_client/fakes"
	"github.com/cloudfoundry-incubator/stager/handlers"
	"github.com/cloudfoundry/dropsonde/metric_sender/fake"
	"github.com/cloudfoundry/dropsonde/metrics"
	"github.com/pivotal-golang/clock/fakeclock"
	"github.com/pivotal-golang/lager"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("StagingCompletedHandler", func() {
	var (
		logger lager.Logger
		appId  string
		taskId string

		fakeCCClient        *fakes.FakeCcClient
		fakeBackend         *fake_backend.FakeBackend
		backendResponse     cc_messages.StagingResponseForCC
		backendError        error
		fakeClock           *fakeclock.FakeClock
		metricSender        *fake.FakeMetricSender
		stagingDurationNano time.Duration

		handler          handlers.CompletionHandler
		responseRecorder *httptest.ResponseRecorder
	)

	BeforeEach(func() {
		logger = lager.NewLogger("fakelogger")
		logger.RegisterSink(lager.NewWriterSink(GinkgoWriter, lager.DEBUG))

		appId = "my_app_id"
		taskId = "do_this"

		stagingDurationNano = 900900
		metricSender = fake.NewFakeMetricSender()
		metrics.Initialize(metricSender)

		fakeCCClient = &fakes.FakeCcClient{}
		fakeBackend = &fake_backend.FakeBackend{}
		backendError = nil

		fakeClock = fakeclock.NewFakeClock(time.Now())

		responseRecorder = httptest.NewRecorder()
		handler = handlers.NewStagingCompletionHandler(logger, fakeCCClient, map[string]backend.Backend{"fake": fakeBackend}, fakeClock)
	})

	JustBeforeEach(func() {
		fakeBackend.BuildStagingResponseReturns(backendResponse, backendError)
	})

	postTask := func(task receptor.TaskResponse) *http.Request {
		taskJSON, err := json.Marshal(task)
		Ω(err).ShouldNot(HaveOccurred())

		request, err := http.NewRequest("POST", stager.StagingCompletedRoute, bytes.NewReader(taskJSON))
		Ω(err).ShouldNot(HaveOccurred())

		return request
	}

	Context("when a staging task completes", func() {
		var taskResponse receptor.TaskResponse
		var annotationJson []byte

		BeforeEach(func() {
			var err error
			annotationJson, err = json.Marshal(cc_messages.StagingTaskAnnotation{
				Lifecycle: "fake",
				AppId:     appId,
				TaskId:    taskId,
			})
			Ω(err).ShouldNot(HaveOccurred())
		})

		JustBeforeEach(func() {
			createdAt := fakeClock.Now().UnixNano()
			fakeClock.Increment(stagingDurationNano)

			taskResponse = receptor.TaskResponse{
				TaskGuid:  "the-task-guid",
				Domain:    "fake-domain",
				CreatedAt: createdAt,
				Result: `{
					"buildpack_key":"buildpack-key",
					"detected_buildpack":"Some Buildpack",
					"execution_metadata":"{\"start_command\":\"./some-start-command\"}",
					"detected_start_command":{"web":"./some-start-command"}
				}`,
				Action: &models.RunAction{
					Path: "ls",
				},
				Annotation: string(annotationJson),
			}

			handler.StagingComplete(responseRecorder, postTask(taskResponse))
		})

		It("passes the task response to the matching response builder", func() {
			Eventually(fakeBackend.BuildStagingResponseCallCount()).Should(Equal(1))
			Ω(fakeBackend.BuildStagingResponseArgsForCall(0)).Should(Equal(taskResponse))
		})

		Describe("staging task annotation", func() {
			Context("when the annotation is missing", func() {
				BeforeEach(func() {
					annotationJson = []byte("")
				})

				It("returns bad request", func() {
					Ω(responseRecorder.Code).Should(Equal(http.StatusBadRequest))
				})

				It("does not post staging complete to the CC", func() {
					Ω(fakeCCClient.StagingCompleteCallCount()).Should(Equal(0))
				})
			})

			Context("when the annotation is invalid JSON", func() {
				BeforeEach(func() {
					annotationJson = []byte(",goo")
				})

				It("returns bad request", func() {
					Ω(responseRecorder.Code).Should(Equal(http.StatusBadRequest))
				})

				It("does not post staging complete to the CC", func() {
					Ω(fakeCCClient.StagingCompleteCallCount()).Should(Equal(0))
				})
			})

			Context("when lifecycle is missing from the annotation", func() {
				BeforeEach(func() {
					annotationJson = []byte(`{
						"task_id": "the-task-id",
						"app_id": "the-app-id"
					}`)
				})

				It("returns not found", func() {
					Ω(responseRecorder.Code).Should(Equal(http.StatusNotFound))
				})

				It("does not post staging complete to the CC", func() {
					Ω(fakeCCClient.StagingCompleteCallCount()).Should(Equal(0))
				})
			})
		})

		Context("when the response builder returns an error", func() {
			BeforeEach(func() {
				backendError = errors.New("build error")
			})

			It("returns a 400", func() {
				Ω(responseRecorder.Code).Should(Equal(http.StatusBadRequest))
			})
		})

		Context("when the response builder does not return an error", func() {
			var backendResponseJson []byte

			BeforeEach(func() {
				backendResponse = cc_messages.StagingResponseForCC{
					AppId:  "the-app-id",
					TaskId: "the-task-id",
				}

				var err error
				backendResponseJson, err = json.Marshal(backendResponse)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("posts the response builder's result to CC", func() {
				Ω(fakeCCClient.StagingCompleteCallCount()).Should(Equal(1))
				payload, _ := fakeCCClient.StagingCompleteArgsForCall(0)
				Ω(payload).Should(Equal(backendResponseJson))
			})

			Context("when the CC request succeeds", func() {
				It("increments the staging success counter", func() {
					Ω(metricSender.GetCounter("StagingRequestsSucceeded")).Should(BeEquivalentTo(1))
				})

				It("emits the time it took to stage succesfully", func() {
					Ω(metricSender.GetValue("StagingRequestSucceededDuration")).Should(Equal(fake.Metric{
						Value: float64(stagingDurationNano),
						Unit:  "nanos",
					}))
				})

				It("returns a 200", func() {
					Ω(responseRecorder.Code).Should(Equal(200))
				})
			})

			Context("when the CC request fails", func() {
				BeforeEach(func() {
					fakeCCClient.StagingCompleteReturns(&cc_client.BadResponseError{504})
				})

				It("responds with the status code that the CC returned", func() {
					Ω(responseRecorder.Code).Should(Equal(504))
				})
			})

			Context("When an error occurs in making the CC request", func() {
				BeforeEach(func() {
					fakeCCClient.StagingCompleteReturns(errors.New("whoops"))
				})

				It("responds with a 503 error", func() {
					Ω(responseRecorder.Code).Should(Equal(503))
				})

				It("does not update the staging counter", func() {
					Ω(metricSender.GetCounter("StagingRequestsSucceeded")).Should(BeEquivalentTo(0))
				})

				It("does not update the staging duration", func() {
					Ω(metricSender.GetValue("StagingRequestSucceededDuration")).Should(Equal(fake.Metric{}))
				})
			})
		})
	})

	Context("when a staging task fails", func() {
		var backendResponseJson []byte

		BeforeEach(func() {
			backendResponse = cc_messages.StagingResponseForCC{
				AppId:  "the-app-id",
				TaskId: "the-task-id",
			}

			var err error
			backendResponseJson, err = json.Marshal(backendResponse)
			Ω(err).ShouldNot(HaveOccurred())
		})

		JustBeforeEach(func() {
			createdAt := fakeClock.Now().UnixNano()
			fakeClock.Increment(stagingDurationNano)

			taskResponse := receptor.TaskResponse{
				Domain:        "fake-domain",
				Failed:        true,
				CreatedAt:     createdAt,
				FailureReason: "because I said so",
				Annotation: `{
					"lifecycle": "fake",
					"task_id": "the-task-id",
					"app_id": "the-app-id"
				}`,
				Action: &models.RunAction{
					Path: "ls",
				},
				Result: `{}`,
			}

			handler.StagingComplete(responseRecorder, postTask(taskResponse))
		})

		It("posts the result to CC as an error", func() {
			Ω(fakeCCClient.StagingCompleteCallCount()).Should(Equal(1))
			payload, _ := fakeCCClient.StagingCompleteArgsForCall(0)
			Ω(payload).Should(Equal(backendResponseJson))
		})

		It("responds with a 200", func() {
			Ω(responseRecorder.Code).Should(Equal(http.StatusOK))
		})

		It("increments the staging failed counter", func() {
			Ω(fakeCCClient.StagingCompleteCallCount()).Should(Equal(1))
			Ω(metricSender.GetCounter("StagingRequestsFailed")).Should(BeEquivalentTo(1))
		})

		It("emits the time it took to stage unsuccesfully", func() {
			Ω(metricSender.GetValue("StagingRequestFailedDuration")).Should(Equal(fake.Metric{
				Value: 900900,
				Unit:  "nanos",
			}))
		})
	})

	Context("when a non-staging task is reported", func() {
		JustBeforeEach(func() {
			taskResponse := receptor.TaskResponse{
				Action: &models.RunAction{
					Path: "ls",
				},
				Domain:        "some-other-crazy-domain",
				Failed:        true,
				FailureReason: "because I said so",
				Annotation:    `{}`,
				Result:        `{}`,
			}

			handler.StagingComplete(responseRecorder, postTask(taskResponse))
		})

		It("responds with a 404", func() {
			Ω(responseRecorder.Code).Should(Equal(404))
		})
	})

	Context("when invalid JSON is posted instead of a task", func() {
		JustBeforeEach(func() {
			request, err := http.NewRequest("POST", stager.StagingCompletedRoute, strings.NewReader("{"))
			Ω(err).ShouldNot(HaveOccurred())

			handler.StagingComplete(responseRecorder, request)
		})

		It("responds with a 400", func() {
			Ω(responseRecorder.Code).Should(Equal(400))
		})
	})
})
