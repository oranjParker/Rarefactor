package crawler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/nats-io/nats.go"
	pb "github.com/oranjParker/Rarefactor/generated/protos/v1"
	"github.com/oranjParker/Rarefactor/internal/core"
	"github.com/pashagolub/pgxmock/v3"
)

type mockJetStream struct {
	nats.JetStreamContext
	publishedSubject string
	publishedData    []byte
}

func (m *mockJetStream) Publish(subj string, data []byte, opts ...nats.PubOpt) (*nats.PubAck, error) {
	m.publishedSubject = subj
	m.publishedData = data
	return &nats.PubAck{Sequence: 1, Stream: "CRAWL_JOBS"}, nil
}

func TestCrawl_Success(t *testing.T) {
	mockDB, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mockDB.Close()

	jsMock := &mockJetStream{}
	service := &CrawlerService{
		db:   mockDB,
		nats: jsMock,
	}

	seedURL := "https://rarefactor.io"
	req := &pb.CrawlRequest{
		SeedUrl:   seedURL,
		MaxDepth:  2,
		CrawlMode: "targeted",
	}

	mockDB.ExpectExec("INSERT INTO crawl_jobs").
		WithArgs(pgxmock.AnyArg(), req.SeedUrl, int32(2), "targeted", "test").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	resp, err := service.Crawl(context.Background(), req)
	if err != nil {
		t.Fatalf("Crawl failed: %v", err)
	}

	if resp.JobId == "" || resp.Status != "QUEUED" {
		t.Errorf("Unexpected response: %+v", resp)
	}

	if jsMock.publishedSubject != "crawl.jobs" {
		t.Errorf("Expected subject crawl.jobs, got %s", jsMock.publishedSubject)
	}

	var doc core.Document[string]
	if err := json.Unmarshal(jsMock.publishedData, &doc); err != nil {
		t.Fatalf("Failed to unmarshal NATS payload: %v", err)
	}

	if doc.ID != seedURL {
		t.Errorf("Expected URL %s in payload, got %s", seedURL, doc.ID)
	}

	if err := mockDB.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet DB expectations: %v", err)
	}
}

func TestCrawl_Validation(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	service := &CrawlerService{
		db:   mock,
		nats: nil,
	}

	t.Run("Missing Seed URL", func(t *testing.T) {
		req := &pb.CrawlRequest{SeedUrl: ""}
		_, err := service.Crawl(context.Background(), req)
		if err == nil || err.Error() != "seed_url is required" {
			t.Errorf("Expected validation error for empty seed_url, got: %v", err)
		}
	})
}

func TestCancelJob_Success(t *testing.T) {
	mockDB, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mockDB.Close()

	service := &CrawlerService{
		db:   mockDB,
		nats: nil,
	}

	jobID := "test-job-123"
	req := &pb.CancelJobRequest{JobId: jobID}

	mockDB.ExpectExec("UPDATE crawl_jobs SET status = 'CANCELLED'").
		WithArgs(jobID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	resp, err := service.CancelJob(context.Background(), req)
	if err != nil {
		t.Fatalf("CancelJob failed: %v", err)
	}

	if resp.Status != "CANCELLED_SIGNAL_SENT" {
		t.Errorf("Expected status CANCELLED_SIGNAL_SENT, got %s", resp.Status)
	}

	if err := mockDB.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet DB expectations: %v", err)
	}
}

func TestCancelJob_Failure(t *testing.T) {
	mockDB, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mockDB.Close()

	service := &CrawlerService{
		db:   mockDB,
		nats: nil,
	}

	jobID := "test-job-456"
	req := &pb.CancelJobRequest{JobId: jobID}

	mockDB.ExpectExec("UPDATE crawl_jobs SET status = 'CANCELLED'").
		WithArgs(jobID).
		WillReturnError(fmt.Errorf("simulated db timeout"))

	_, err = service.CancelJob(context.Background(), req)
	if err == nil {
		t.Fatal("Expected error on DB failure, got nil")
	}

	if err := mockDB.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet DB expectations: %v", err)
	}
}

func TestCrawl_DBFailure(t *testing.T) {
	mockDB, _ := pgxmock.NewPool()
	defer mockDB.Close()
	service := &CrawlerService{db: mockDB, nats: &mockJetStream{}}

	mockDB.ExpectExec("INSERT INTO crawl_jobs").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(fmt.Errorf("db error"))

	_, err := service.Crawl(context.Background(), &pb.CrawlRequest{SeedUrl: "http://test.com"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCrawl_NatsFailure(t *testing.T) {
	mockDB, _ := pgxmock.NewPool()
	defer mockDB.Close()
	jsMock := &mockJetStreamFail{}
	service := &CrawlerService{db: mockDB, nats: jsMock}

	mockDB.ExpectExec("INSERT INTO crawl_jobs").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	mockDB.ExpectExec("UPDATE crawl_jobs SET status = 'FAILED'").
		WithArgs(pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	_, err := service.Crawl(context.Background(), &pb.CrawlRequest{SeedUrl: "http://test.com"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to queue job") {
		t.Errorf("expected queue job error, got %v", err)
	}

	if err := mockDB.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet DB expectations: %v", err)
	}
}

func TestCrawl_NatsFailure_DBUpdateFailure(t *testing.T) {
	mockDB, _ := pgxmock.NewPool()
	defer mockDB.Close()
	jsMock := &mockJetStreamFail{}
	service := &CrawlerService{db: mockDB, nats: jsMock}

	mockDB.ExpectExec("INSERT INTO crawl_jobs").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	mockDB.ExpectExec("UPDATE crawl_jobs SET status = 'FAILED'").
		WithArgs(pgxmock.AnyArg()).
		WillReturnError(fmt.Errorf("db update failed"))

	_, err := service.Crawl(context.Background(), &pb.CrawlRequest{SeedUrl: "http://test.com"})
	if err == nil {
		t.Fatal("expected error")
	}

	if err := mockDB.ExpectationsWereMet(); err != nil {
		t.Errorf("Unmet DB expectations: %v", err)
	}
}

type mockJetStreamFail struct {
	nats.JetStreamContext
}

func (m *mockJetStreamFail) Publish(subj string, data []byte, opts ...nats.PubOpt) (*nats.PubAck, error) {
	return nil, fmt.Errorf("nats error")
}
