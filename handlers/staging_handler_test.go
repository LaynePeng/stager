package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"

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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
)

var _ = Describe("StagingHandler", func() {
	var (
		fakeCcClient           *fakes.FakeCcClient
		fakeBackend            *fake_backend.FakeBackend
		logOutput              *gbytes.Buffer
		logger                 lager.Logger
		stagingRequestJson     []byte
		stopStagingRequestJson []byte
		fakeDiegoClient        *fake_receptor.FakeClient
		fakeMetricSender       *fake_metric_sender.FakeMetricSender

		handler          handlers.StagingHandler
		responseRecorder *httptest.ResponseRecorder
	)

	BeforeEach(func() {
		logOutput = gbytes.NewBuffer()
		logger = lager.NewLogger("fakelogger")
		logger.RegisterSink(lager.NewWriterSink(logOutput, lager.INFO))

		stagingRequest := cc_messages.StagingRequestFromCC{
			AppId:  "myapp",
			TaskId: "mytask",
		}

		var err error
		stagingRequestJson, err = json.Marshal(stagingRequest)
		Ω(err).ShouldNot(HaveOccurred())

		stopStagingRequest := cc_messages.StopStagingRequestFromCC{
			AppId:  "myapp",
			TaskId: "mytask",
		}
		stopStagingRequestJson, err = json.Marshal(stopStagingRequest)
		Ω(err).ShouldNot(HaveOccurred())

		fakeMetricSender = fake_metric_sender.NewFakeMetricSender()
		metrics.Initialize(fakeMetricSender)
		fakeCcClient = &fakes.FakeCcClient{}
		fakeBackend = &fake_backend.FakeBackend{}
		fakeBackend.StagingRequestsNatsSubjectReturns("stage-subscription-subject")
		fakeBackend.StagingRequestsReceivedCounterReturns(metric.Counter("FakeStageMetricName"))
		fakeBackend.StopStagingRequestsNatsSubjectReturns("stop-subscription-subject")
		fakeBackend.StopStagingRequestsReceivedCounterReturns(metric.Counter("FakeStopMetricName"))
		fakeDiegoClient = &fake_receptor.FakeClient{}

		handler = handlers.NewStagingHandler(logger, fakeDiegoClient, []backend.Backend{fakeBackend})
		responseRecorder = httptest.NewRecorder()
	})

	Context("when a staging request is received", func() {
		JustBeforeEach(func() {
			req, err := http.NewRequest("POST", stager.StageRoute, strings.NewReader(string(stagingRequestJson)))
			Ω(err).ShouldNot(HaveOccurred())

			handler.Stage(responseRecorder, req)
		})

		It("increments the counter to track arriving staging messages", func() {
			Ω(fakeMetricSender.GetCounter("FakeStageMetricName")).Should(Equal(uint64(1)))
		})

		It("builds a staging recipe", func() {
			Ω(fakeBackend.BuildRecipeCallCount()).To(Equal(1))
			Ω(fakeBackend.BuildRecipeArgsForCall(0)).To(Equal(stagingRequestJson))
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
		})
	})
})
