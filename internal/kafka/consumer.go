package kafka

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"github.com/leaderboard-redis/internal/config"
	"github.com/leaderboard-redis/internal/domain"
)

// ScoreHandler processes score submissions
type ScoreHandler interface {
	SubmitScore(ctx context.Context, submission domain.ScoreSubmission) error
	SubmitScoreBatch(ctx context.Context, batch domain.BatchScoreSubmission) error
}

// Consumer consumes score messages from Kafka
type Consumer struct {
	config        *config.KafkaConfig
	handler       ScoreHandler
	logger        *slog.Logger
	consumerGroup sarama.ConsumerGroup
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	ready         chan bool
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(cfg *config.KafkaConfig, handler ScoreHandler, logger *slog.Logger) (*Consumer, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = sarama.V3_0_0_0
	saramaConfig.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRoundRobin()}
	saramaConfig.Consumer.Offsets.Initial = sarama.OffsetNewest
	saramaConfig.Consumer.Return.Errors = true

	consumerGroup, err := sarama.NewConsumerGroup(cfg.Brokers, cfg.GroupID, saramaConfig)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Consumer{
		config:        cfg,
		handler:       handler,
		logger:        logger,
		consumerGroup: consumerGroup,
		ctx:           ctx,
		cancel:        cancel,
		ready:         make(chan bool),
	}, nil
}

// Start begins consuming messages from Kafka
func (c *Consumer) Start() error {
	c.logger.Info("starting Kafka consumer",
		"brokers", c.config.Brokers,
		"topic", c.config.Topic,
		"group_id", c.config.GroupID,
	)

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for {
			handler := &consumerGroupHandler{
				consumer: c,
				ready:    c.ready,
			}

			if err := c.consumerGroup.Consume(c.ctx, []string{c.config.Topic}, handler); err != nil {
				if err == sarama.ErrClosedConsumerGroup {
					return
				}
				c.logger.Error("error from consumer", "error", err)
			}

			// Check if context was cancelled
			if c.ctx.Err() != nil {
				return
			}

			c.ready = make(chan bool)
		}
	}()

	// Wait until consumer is ready
	<-c.ready
	c.logger.Info("Kafka consumer ready")

	// Handle errors in separate goroutine
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		for {
			select {
			case <-c.ctx.Done():
				return
			case err, ok := <-c.consumerGroup.Errors():
				if !ok {
					return
				}
				c.logger.Error("consumer group error", "error", err)
			}
		}
	}()

	return nil
}

// Stop gracefully stops the consumer
func (c *Consumer) Stop() error {
	c.logger.Info("stopping Kafka consumer")
	c.cancel()
	c.wg.Wait()
	return c.consumerGroup.Close()
}

// consumerGroupHandler implements sarama.ConsumerGroupHandler
type consumerGroupHandler struct {
	consumer *Consumer
	ready    chan bool
}

// Setup is called at the beginning of a new session
func (h *consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
	close(h.ready)
	return nil
}

// Cleanup is called at the end of a session
func (h *consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim processes messages from a topic partition
func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	cfg := h.consumer.config
	batch := make([]domain.ScoreSubmission, 0, cfg.BatchSize)
	batchTimer := time.NewTimer(cfg.BatchTimeout)
	defer batchTimer.Stop()

	processBatch := func() {
		if len(batch) == 0 {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		batchSubmission := domain.BatchScoreSubmission{Scores: batch}
		if err := h.consumer.handler.SubmitScoreBatch(ctx, batchSubmission); err != nil {
			h.consumer.logger.Error("failed to process batch", "error", err, "batch_size", len(batch))
		} else {
			h.consumer.logger.Debug("processed batch", "batch_size", len(batch))
		}

		batch = batch[:0]
	}

	for {
		select {
		case <-session.Context().Done():
			// Process remaining batch before exit
			processBatch()
			return nil

		case <-batchTimer.C:
			processBatch()
			batchTimer.Reset(cfg.BatchTimeout)

		case message, ok := <-claim.Messages():
			if !ok {
				processBatch()
				return nil
			}

			var submission domain.ScoreSubmission
			if err := json.Unmarshal(message.Value, &submission); err != nil {
				h.consumer.logger.Warn("failed to unmarshal message",
					"error", err,
					"offset", message.Offset,
					"partition", message.Partition,
				)
				session.MarkMessage(message, "")
				continue
			}

			// Validate submission
			if submission.PlayerID == "" || submission.LeaderboardID == "" {
				h.consumer.logger.Warn("invalid score submission",
					"player_id", submission.PlayerID,
					"leaderboard_id", submission.LeaderboardID,
				)
				session.MarkMessage(message, "")
				continue
			}

			batch = append(batch, submission)
			session.MarkMessage(message, "")

			if len(batch) >= cfg.BatchSize {
				processBatch()
				batchTimer.Reset(cfg.BatchTimeout)
			}
		}
	}
}

// KafkaMessage represents the message format for Kafka
type KafkaMessage struct {
	PlayerID      string                 `json:"player_id"`
	LeaderboardID string                 `json:"leaderboard_id"`
	Score         int64                  `json:"score"`
	GameID        string                 `json:"game_id,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}
