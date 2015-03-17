// This file was generated by counterfeiter
package fake_backend

import (
	"sync"

	"github.com/cloudfoundry-incubator/receptor"
	"github.com/cloudfoundry-incubator/runtime-schema/cc_messages"
	"github.com/cloudfoundry-incubator/runtime-schema/metric"
	"github.com/cloudfoundry-incubator/stager/backend"
)

type FakeBackend struct {
	StagingRequestsReceivedCounterStub        func() metric.Counter
	stagingRequestsReceivedCounterMutex       sync.RWMutex
	stagingRequestsReceivedCounterArgsForCall []struct{}
	stagingRequestsReceivedCounterReturns struct {
		result1 metric.Counter
	}
	StopStagingRequestsReceivedCounterStub        func() metric.Counter
	stopStagingRequestsReceivedCounterMutex       sync.RWMutex
	stopStagingRequestsReceivedCounterArgsForCall []struct{}
	stopStagingRequestsReceivedCounterReturns struct {
		result1 metric.Counter
	}
	TaskDomainStub        func() string
	taskDomainMutex       sync.RWMutex
	taskDomainArgsForCall []struct{}
	taskDomainReturns struct {
		result1 string
	}
	BuildRecipeStub        func(request cc_messages.StagingRequestFromCC) (receptor.TaskCreateRequest, error)
	buildRecipeMutex       sync.RWMutex
	buildRecipeArgsForCall []struct {
		request cc_messages.StagingRequestFromCC
	}
	buildRecipeReturns struct {
		result1 receptor.TaskCreateRequest
		result2 error
	}
	BuildStagingResponseStub        func(receptor.TaskResponse) (cc_messages.StagingResponseForCC, error)
	buildStagingResponseMutex       sync.RWMutex
	buildStagingResponseArgsForCall []struct {
		arg1 receptor.TaskResponse
	}
	buildStagingResponseReturns struct {
		result1 cc_messages.StagingResponseForCC
		result2 error
	}
	BuildStagingResponseFromRequestErrorStub        func(request cc_messages.StagingRequestFromCC, errorMessage string) cc_messages.StagingResponseForCC
	buildStagingResponseFromRequestErrorMutex       sync.RWMutex
	buildStagingResponseFromRequestErrorArgsForCall []struct {
		request      cc_messages.StagingRequestFromCC
		errorMessage string
	}
	buildStagingResponseFromRequestErrorReturns struct {
		result1 cc_messages.StagingResponseForCC
	}
	StagingTaskGuidStub        func(request cc_messages.StopStagingRequestFromCC) (string, error)
	stagingTaskGuidMutex       sync.RWMutex
	stagingTaskGuidArgsForCall []struct {
		request cc_messages.StopStagingRequestFromCC
	}
	stagingTaskGuidReturns struct {
		result1 string
		result2 error
	}
}

func (fake *FakeBackend) StagingRequestsReceivedCounter() metric.Counter {
	fake.stagingRequestsReceivedCounterMutex.Lock()
	fake.stagingRequestsReceivedCounterArgsForCall = append(fake.stagingRequestsReceivedCounterArgsForCall, struct{}{})
	fake.stagingRequestsReceivedCounterMutex.Unlock()
	if fake.StagingRequestsReceivedCounterStub != nil {
		return fake.StagingRequestsReceivedCounterStub()
	} else {
		return fake.stagingRequestsReceivedCounterReturns.result1
	}
}

func (fake *FakeBackend) StagingRequestsReceivedCounterCallCount() int {
	fake.stagingRequestsReceivedCounterMutex.RLock()
	defer fake.stagingRequestsReceivedCounterMutex.RUnlock()
	return len(fake.stagingRequestsReceivedCounterArgsForCall)
}

func (fake *FakeBackend) StagingRequestsReceivedCounterReturns(result1 metric.Counter) {
	fake.StagingRequestsReceivedCounterStub = nil
	fake.stagingRequestsReceivedCounterReturns = struct {
		result1 metric.Counter
	}{result1}
}

func (fake *FakeBackend) StopStagingRequestsReceivedCounter() metric.Counter {
	fake.stopStagingRequestsReceivedCounterMutex.Lock()
	fake.stopStagingRequestsReceivedCounterArgsForCall = append(fake.stopStagingRequestsReceivedCounterArgsForCall, struct{}{})
	fake.stopStagingRequestsReceivedCounterMutex.Unlock()
	if fake.StopStagingRequestsReceivedCounterStub != nil {
		return fake.StopStagingRequestsReceivedCounterStub()
	} else {
		return fake.stopStagingRequestsReceivedCounterReturns.result1
	}
}

func (fake *FakeBackend) StopStagingRequestsReceivedCounterCallCount() int {
	fake.stopStagingRequestsReceivedCounterMutex.RLock()
	defer fake.stopStagingRequestsReceivedCounterMutex.RUnlock()
	return len(fake.stopStagingRequestsReceivedCounterArgsForCall)
}

func (fake *FakeBackend) StopStagingRequestsReceivedCounterReturns(result1 metric.Counter) {
	fake.StopStagingRequestsReceivedCounterStub = nil
	fake.stopStagingRequestsReceivedCounterReturns = struct {
		result1 metric.Counter
	}{result1}
}

func (fake *FakeBackend) TaskDomain() string {
	fake.taskDomainMutex.Lock()
	fake.taskDomainArgsForCall = append(fake.taskDomainArgsForCall, struct{}{})
	fake.taskDomainMutex.Unlock()
	if fake.TaskDomainStub != nil {
		return fake.TaskDomainStub()
	} else {
		return fake.taskDomainReturns.result1
	}
}

func (fake *FakeBackend) TaskDomainCallCount() int {
	fake.taskDomainMutex.RLock()
	defer fake.taskDomainMutex.RUnlock()
	return len(fake.taskDomainArgsForCall)
}

func (fake *FakeBackend) TaskDomainReturns(result1 string) {
	fake.TaskDomainStub = nil
	fake.taskDomainReturns = struct {
		result1 string
	}{result1}
}

func (fake *FakeBackend) BuildRecipe(request cc_messages.StagingRequestFromCC) (receptor.TaskCreateRequest, error) {
	fake.buildRecipeMutex.Lock()
	fake.buildRecipeArgsForCall = append(fake.buildRecipeArgsForCall, struct {
		request cc_messages.StagingRequestFromCC
	}{request})
	fake.buildRecipeMutex.Unlock()
	if fake.BuildRecipeStub != nil {
		return fake.BuildRecipeStub(request)
	} else {
		return fake.buildRecipeReturns.result1, fake.buildRecipeReturns.result2
	}
}

func (fake *FakeBackend) BuildRecipeCallCount() int {
	fake.buildRecipeMutex.RLock()
	defer fake.buildRecipeMutex.RUnlock()
	return len(fake.buildRecipeArgsForCall)
}

func (fake *FakeBackend) BuildRecipeArgsForCall(i int) cc_messages.StagingRequestFromCC {
	fake.buildRecipeMutex.RLock()
	defer fake.buildRecipeMutex.RUnlock()
	return fake.buildRecipeArgsForCall[i].request
}

func (fake *FakeBackend) BuildRecipeReturns(result1 receptor.TaskCreateRequest, result2 error) {
	fake.BuildRecipeStub = nil
	fake.buildRecipeReturns = struct {
		result1 receptor.TaskCreateRequest
		result2 error
	}{result1, result2}
}

func (fake *FakeBackend) BuildStagingResponse(arg1 receptor.TaskResponse) (cc_messages.StagingResponseForCC, error) {
	fake.buildStagingResponseMutex.Lock()
	fake.buildStagingResponseArgsForCall = append(fake.buildStagingResponseArgsForCall, struct {
		arg1 receptor.TaskResponse
	}{arg1})
	fake.buildStagingResponseMutex.Unlock()
	if fake.BuildStagingResponseStub != nil {
		return fake.BuildStagingResponseStub(arg1)
	} else {
		return fake.buildStagingResponseReturns.result1, fake.buildStagingResponseReturns.result2
	}
}

func (fake *FakeBackend) BuildStagingResponseCallCount() int {
	fake.buildStagingResponseMutex.RLock()
	defer fake.buildStagingResponseMutex.RUnlock()
	return len(fake.buildStagingResponseArgsForCall)
}

func (fake *FakeBackend) BuildStagingResponseArgsForCall(i int) receptor.TaskResponse {
	fake.buildStagingResponseMutex.RLock()
	defer fake.buildStagingResponseMutex.RUnlock()
	return fake.buildStagingResponseArgsForCall[i].arg1
}

func (fake *FakeBackend) BuildStagingResponseReturns(result1 cc_messages.StagingResponseForCC, result2 error) {
	fake.BuildStagingResponseStub = nil
	fake.buildStagingResponseReturns = struct {
		result1 cc_messages.StagingResponseForCC
		result2 error
	}{result1, result2}
}

func (fake *FakeBackend) BuildStagingResponseFromRequestError(request cc_messages.StagingRequestFromCC, errorMessage string) cc_messages.StagingResponseForCC {
	fake.buildStagingResponseFromRequestErrorMutex.Lock()
	fake.buildStagingResponseFromRequestErrorArgsForCall = append(fake.buildStagingResponseFromRequestErrorArgsForCall, struct {
		request      cc_messages.StagingRequestFromCC
		errorMessage string
	}{request, errorMessage})
	fake.buildStagingResponseFromRequestErrorMutex.Unlock()
	if fake.BuildStagingResponseFromRequestErrorStub != nil {
		return fake.BuildStagingResponseFromRequestErrorStub(request, errorMessage)
	} else {
		return fake.buildStagingResponseFromRequestErrorReturns.result1
	}
}

func (fake *FakeBackend) BuildStagingResponseFromRequestErrorCallCount() int {
	fake.buildStagingResponseFromRequestErrorMutex.RLock()
	defer fake.buildStagingResponseFromRequestErrorMutex.RUnlock()
	return len(fake.buildStagingResponseFromRequestErrorArgsForCall)
}

func (fake *FakeBackend) BuildStagingResponseFromRequestErrorArgsForCall(i int) (cc_messages.StagingRequestFromCC, string) {
	fake.buildStagingResponseFromRequestErrorMutex.RLock()
	defer fake.buildStagingResponseFromRequestErrorMutex.RUnlock()
	return fake.buildStagingResponseFromRequestErrorArgsForCall[i].request, fake.buildStagingResponseFromRequestErrorArgsForCall[i].errorMessage
}

func (fake *FakeBackend) BuildStagingResponseFromRequestErrorReturns(result1 cc_messages.StagingResponseForCC) {
	fake.BuildStagingResponseFromRequestErrorStub = nil
	fake.buildStagingResponseFromRequestErrorReturns = struct {
		result1 cc_messages.StagingResponseForCC
	}{result1}
}

func (fake *FakeBackend) StagingTaskGuid(request cc_messages.StopStagingRequestFromCC) (string, error) {
	fake.stagingTaskGuidMutex.Lock()
	fake.stagingTaskGuidArgsForCall = append(fake.stagingTaskGuidArgsForCall, struct {
		request cc_messages.StopStagingRequestFromCC
	}{request})
	fake.stagingTaskGuidMutex.Unlock()
	if fake.StagingTaskGuidStub != nil {
		return fake.StagingTaskGuidStub(request)
	} else {
		return fake.stagingTaskGuidReturns.result1, fake.stagingTaskGuidReturns.result2
	}
}

func (fake *FakeBackend) StagingTaskGuidCallCount() int {
	fake.stagingTaskGuidMutex.RLock()
	defer fake.stagingTaskGuidMutex.RUnlock()
	return len(fake.stagingTaskGuidArgsForCall)
}

func (fake *FakeBackend) StagingTaskGuidArgsForCall(i int) cc_messages.StopStagingRequestFromCC {
	fake.stagingTaskGuidMutex.RLock()
	defer fake.stagingTaskGuidMutex.RUnlock()
	return fake.stagingTaskGuidArgsForCall[i].request
}

func (fake *FakeBackend) StagingTaskGuidReturns(result1 string, result2 error) {
	fake.StagingTaskGuidStub = nil
	fake.stagingTaskGuidReturns = struct {
		result1 string
		result2 error
	}{result1, result2}
}

var _ backend.Backend = new(FakeBackend)
