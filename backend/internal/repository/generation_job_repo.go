package repository

import (
	"context"
	"database/sql"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/generationjob"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type generationJobRepository struct {
	client *dbent.Client
}

func NewGenerationJobRepository(client *dbent.Client, _ *sql.DB) service.GenerationJobRepository {
	return &generationJobRepository{client: client}
}

func (r *generationJobRepository) Create(ctx context.Context, job *service.GenerationJob) error {
	service.NormalizeGenerationJob(job)
	created, err := r.client.GenerationJob.Create().
		SetPublicID(job.PublicID).
		SetProvider(job.Provider).
		SetModality(job.Modality).
		SetModel(job.Model).
		SetUpstreamModel(job.UpstreamModel).
		SetUserID(job.UserID).
		SetAPIKeyID(job.APIKeyID).
		SetNillableGroupID(job.GroupID).
		SetAccountID(job.AccountID).
		SetNillableUpstreamGenerationID(job.UpstreamGenerationID).
		SetStatus(generationjob.Status(job.Status)).
		SetNillableUpstreamStatus(job.UpstreamStatus).
		SetRequestHash(job.RequestHash).
		SetRequestPayload(job.RequestPayload).
		SetResultPayload(job.ResultPayload).
		SetNillableErrorCode(job.ErrorCode).
		SetNillableErrorMessage(job.ErrorMessage).
		SetOutputCount(job.OutputCount).
		SetActualUpstreamCostAmount(job.ActualUpstreamCostAmount).
		SetNillableActualUpstreamCostUnit(job.ActualUpstreamCostUnit).
		SetCustomerCost(job.CustomerCost).
		SetBillingStatus(generationjob.BillingStatus(job.BillingStatus)).
		SetNillableBillingReference(job.BillingReference).
		SetPollAttempts(job.PollAttempts).
		SetNillableNextPollAt(job.NextPollAt).
		SetNillableLastPolledAt(job.LastPolledAt).
		SetNillableSubmittedAt(job.SubmittedAt).
		SetNillableStartedAt(job.StartedAt).
		SetNillableCompletedAt(job.CompletedAt).
		SetNillableFailedAt(job.FailedAt).
		Save(ctx)
	if err != nil {
		return err
	}
	*job = *generationJobEntityToService(created)
	return nil
}

func (r *generationJobRepository) GetByPublicID(ctx context.Context, publicID string) (*service.GenerationJob, error) {
	entity, err := r.client.GenerationJob.Query().Where(generationjob.PublicIDEQ(publicID)).Only(ctx)
	return generationJobResult(entity, err)
}

func (r *generationJobRepository) GetByUpstreamGenerationID(ctx context.Context, upstreamGenerationID string) (*service.GenerationJob, error) {
	entity, err := r.client.GenerationJob.Query().Where(generationjob.UpstreamGenerationIDEQ(upstreamGenerationID)).Only(ctx)
	return generationJobResult(entity, err)
}

func (r *generationJobRepository) CompareAndSwapStatus(ctx context.Context, publicID string, expectedStatus service.GenerationJobStatus, job *service.GenerationJob) error {
	if job == nil || !service.CanTransitionGenerationJobStatus(expectedStatus, job.Status) {
		return service.ErrGenerationJobConflict
	}
	service.NormalizeGenerationJob(job)
	update := r.client.GenerationJob.Update().
		Where(generationjob.PublicIDEQ(publicID), generationjob.StatusEQ(generationjob.Status(expectedStatus))).
		SetStatus(generationjob.Status(job.Status)).
		SetNillableUpstreamGenerationID(job.UpstreamGenerationID).
		SetNillableUpstreamStatus(job.UpstreamStatus).
		SetResultPayload(job.ResultPayload).
		SetNillableErrorCode(job.ErrorCode).
		SetNillableErrorMessage(job.ErrorMessage).
		SetOutputCount(job.OutputCount).
		SetNillableActualUpstreamCostUnit(job.ActualUpstreamCostUnit).
		SetBillingStatus(generationjob.BillingStatus(job.BillingStatus)).
		SetNillableBillingReference(job.BillingReference).
		SetPollAttempts(job.PollAttempts).
		SetNillableNextPollAt(job.NextPollAt).
		SetNillableLastPolledAt(job.LastPolledAt).
		SetNillableSubmittedAt(job.SubmittedAt).
		SetNillableStartedAt(job.StartedAt).
		SetNillableCompletedAt(job.CompletedAt).
		SetNillableFailedAt(job.FailedAt)
	if job.ActualUpstreamCostAmount == nil {
		update.ClearActualUpstreamCostAmount()
	} else {
		update.SetActualUpstreamCostAmount(job.ActualUpstreamCostAmount)
	}
	if job.ActualUpstreamCostUnit == nil {
		update.ClearActualUpstreamCostUnit()
	}
	if job.CustomerCost == nil {
		update.ClearCustomerCost()
	} else {
		update.SetCustomerCost(job.CustomerCost)
	}
	if job.UpstreamGenerationID == nil {
		update.ClearUpstreamGenerationID()
	}
	if job.UpstreamStatus == nil {
		update.ClearUpstreamStatus()
	}
	if job.ErrorCode == nil {
		update.ClearErrorCode()
	}
	if job.ErrorMessage == nil {
		update.ClearErrorMessage()
	}
	if job.BillingReference == nil {
		update.ClearBillingReference()
	}
	if job.NextPollAt == nil {
		update.ClearNextPollAt()
	}
	if job.LastPolledAt == nil {
		update.ClearLastPolledAt()
	}
	if job.SubmittedAt == nil {
		update.ClearSubmittedAt()
	}
	if job.StartedAt == nil {
		update.ClearStartedAt()
	}
	if job.CompletedAt == nil {
		update.ClearCompletedAt()
	}
	if job.FailedAt == nil {
		update.ClearFailedAt()
	}
	affected, err := update.Save(ctx)
	if err != nil {
		return err
	}
	if affected > 0 {
		stored, err := r.GetByPublicID(ctx, publicID)
		if err != nil {
			return err
		}
		*job = *stored
		return nil
	}
	exists, err := r.client.GenerationJob.Query().Where(generationjob.PublicIDEQ(publicID)).Exist(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return service.ErrGenerationJobNotFound
	}
	return service.ErrGenerationJobConflict
}

func (r *generationJobRepository) CompareAndSwapPoll(ctx context.Context, publicID string, expectedStatus service.GenerationJobStatus, expectedPollAttempts int, job *service.GenerationJob) error {
	if job == nil ||
		(expectedStatus != service.GenerationJobStatusQueued && expectedStatus != service.GenerationJobStatusRunning) ||
		(job.Status != service.GenerationJobStatusQueued && job.Status != service.GenerationJobStatusRunning && job.Status != service.GenerationJobStatusSucceeded && job.Status != service.GenerationJobStatusFailed) ||
		!service.CanTransitionGenerationJobStatus(expectedStatus, job.Status) {
		return service.ErrGenerationJobConflict
	}
	update := r.client.GenerationJob.Update().
		Where(
			generationjob.PublicIDEQ(publicID),
			generationjob.StatusEQ(generationjob.Status(expectedStatus)),
			generationjob.PollAttemptsEQ(expectedPollAttempts),
		).
		SetStatus(generationjob.Status(job.Status)).
		SetNillableUpstreamStatus(job.UpstreamStatus).
		SetResultPayload(job.ResultPayload).
		SetNillableErrorCode(job.ErrorCode).
		SetNillableErrorMessage(job.ErrorMessage).
		SetOutputCount(job.OutputCount).
		SetPollAttempts(job.PollAttempts).
		SetNillableNextPollAt(job.NextPollAt).
		SetNillableLastPolledAt(job.LastPolledAt).
		SetNillableCompletedAt(job.CompletedAt).
		SetNillableFailedAt(job.FailedAt)
	if job.UpstreamStatus == nil {
		update.ClearUpstreamStatus()
	}
	if job.ErrorCode == nil {
		update.ClearErrorCode()
	}
	if job.ErrorMessage == nil {
		update.ClearErrorMessage()
	}
	if job.NextPollAt == nil {
		update.ClearNextPollAt()
	}
	if job.LastPolledAt == nil {
		update.ClearLastPolledAt()
	}
	if job.CompletedAt == nil {
		update.ClearCompletedAt()
	}
	if job.FailedAt == nil {
		update.ClearFailedAt()
	}
	affected, err := update.Save(ctx)
	if err != nil {
		return err
	}
	if affected > 0 {
		stored, err := r.GetByPublicID(ctx, publicID)
		if err != nil {
			return err
		}
		*job = *stored
		return nil
	}
	exists, err := r.client.GenerationJob.Query().Where(generationjob.PublicIDEQ(publicID)).Exist(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return service.ErrGenerationJobNotFound
	}
	return service.ErrGenerationJobConflict
}

func generationJobResult(entity *dbent.GenerationJob, err error) (*service.GenerationJob, error) {
	if dbent.IsNotFound(err) {
		return nil, service.ErrGenerationJobNotFound
	}
	if err != nil {
		return nil, err
	}
	return generationJobEntityToService(entity), nil
}

func generationJobEntityToService(entity *dbent.GenerationJob) *service.GenerationJob {
	if entity == nil {
		return nil
	}
	return &service.GenerationJob{
		ID:                       entity.ID,
		PublicID:                 entity.PublicID,
		Provider:                 entity.Provider,
		Modality:                 entity.Modality,
		Model:                    entity.Model,
		UpstreamModel:            entity.UpstreamModel,
		UserID:                   entity.UserID,
		APIKeyID:                 entity.APIKeyID,
		GroupID:                  entity.GroupID,
		AccountID:                entity.AccountID,
		UpstreamGenerationID:     entity.UpstreamGenerationID,
		Status:                   service.GenerationJobStatus(entity.Status),
		UpstreamStatus:           entity.UpstreamStatus,
		RequestHash:              entity.RequestHash,
		RequestPayload:           entity.RequestPayload,
		ResultPayload:            entity.ResultPayload,
		ErrorCode:                entity.ErrorCode,
		ErrorMessage:             entity.ErrorMessage,
		OutputCount:              entity.OutputCount,
		ActualUpstreamCostAmount: entity.ActualUpstreamCostAmount,
		ActualUpstreamCostUnit:   entity.ActualUpstreamCostUnit,
		CustomerCost:             entity.CustomerCost,
		BillingStatus:            service.GenerationJobBillingStatus(entity.BillingStatus),
		BillingReference:         entity.BillingReference,
		PollAttempts:             entity.PollAttempts,
		NextPollAt:               entity.NextPollAt,
		LastPolledAt:             entity.LastPolledAt,
		SubmittedAt:              entity.SubmittedAt,
		StartedAt:                entity.StartedAt,
		CompletedAt:              entity.CompletedAt,
		FailedAt:                 entity.FailedAt,
		CreatedAt:                entity.CreatedAt,
		UpdatedAt:                entity.UpdatedAt,
	}
}
