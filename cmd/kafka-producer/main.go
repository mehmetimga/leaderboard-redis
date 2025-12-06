package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/IBM/sarama"
)

// ScoreSubmission represents a score submission message
type ScoreSubmission struct {
	PlayerID      string                 `json:"player_id"`
	LeaderboardID string                 `json:"leaderboard_id"`
	Score         int64                  `json:"score"`
	GameID        string                 `json:"game_id,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

var playerPrefixes = []string{
	"Phoenix", "Shadow", "Thunder", "Storm", "Blaze", "Ninja", "Dragon", "Wolf", "Hawk", "Viper",
	"Ghost", "Titan", "Frost", "Cyber", "Nova", "Raven", "Omega", "Alpha", "Delta", "Sigma",
	"Ace", "Bolt", "Crash", "Dash", "Edge", "Flash", "Glitch", "Haze", "Ion", "Jade",
	"Knight", "Luna", "Mystic", "Neon", "Orion", "Pulse", "Quantum", "Rebel", "Spark", "Turbo",
}

func getPlayerName(idx int) string {
	prefixIdx := idx % len(playerPrefixes)
	suffix := idx/len(playerPrefixes) + 1
	return fmt.Sprintf("%s%d", playerPrefixes[prefixIdx], suffix)
}

func main() {
	// Command line flags
	brokers := flag.String("brokers", "localhost:9094", "Kafka brokers (comma-separated)")
	topic := flag.String("topic", "leaderboard-scores", "Kafka topic")
	leaderboardID := flag.String("leaderboard", "game1", "Leaderboard ID")
	totalPlayers := flag.Int("players", 1000, "Total number of players to create")
	updatesPerSecond := flag.Int("rate", 100, "Updates per second")
	batchSize := flag.Int("batch", 10, "Batch size for initial population")
	duration := flag.Duration("duration", 0, "Duration to run (0 = forever)")
	initialOnly := flag.Bool("initial-only", false, "Only create initial players, no continuous updates")
	flag.Parse()

	brokerList := strings.Split(*brokers, ",")

	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println("  🚀 Kafka Leaderboard Producer")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("  Brokers:          %s\n", *brokers)
	fmt.Printf("  Topic:            %s\n", *topic)
	fmt.Printf("  Leaderboard:      %s\n", *leaderboardID)
	fmt.Printf("  Total Players:    %d\n", *totalPlayers)
	fmt.Printf("  Updates/sec:      %d\n", *updatesPerSecond)
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()

	// Configure Sarama producer
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForLocal
	config.Producer.Compression = sarama.CompressionSnappy
	config.Producer.Flush.Frequency = 100 * time.Millisecond
	config.Producer.Flush.Messages = 100
	config.Producer.Return.Successes = true
	config.Producer.Return.Errors = true

	// Create producer
	producer, err := sarama.NewAsyncProducer(brokerList, config)
	if err != nil {
		log.Fatalf("Failed to create producer: %v", err)
	}
	defer producer.Close()

	// Handle producer errors and successes
	var successCount, errorCount int64
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for range producer.Successes() {
			atomic.AddInt64(&successCount, 1)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for err := range producer.Errors() {
			atomic.AddInt64(&errorCount, 1)
			log.Printf("Producer error: %v", err)
		}
	}()

	// Handle shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	done := make(chan struct{})

	// Send message helper
	sendMessage := func(submission ScoreSubmission) {
		data, err := json.Marshal(submission)
		if err != nil {
			log.Printf("Failed to marshal message: %v", err)
			return
		}

		msg := &sarama.ProducerMessage{
			Topic: *topic,
			Key:   sarama.StringEncoder(submission.PlayerID),
			Value: sarama.ByteEncoder(data),
		}

		select {
		case producer.Input() <- msg:
		case <-done:
			return
		}
	}

	// Create initial players in batches
	fmt.Printf("Creating %d initial players...\n", *totalPlayers)
	for i := 0; i < *totalPlayers; i += *batchSize {
		end := i + *batchSize
		if end > *totalPlayers {
			end = *totalPlayers
		}

		for j := i; j < end; j++ {
			submission := ScoreSubmission{
				PlayerID:      getPlayerName(j),
				LeaderboardID: *leaderboardID,
				Score:         int64(rand.Intn(5000) + 1000),
			}
			sendMessage(submission)
		}

		progress := float64(end) / float64(*totalPlayers) * 100
		fmt.Printf("\r  Progress: %d/%d players (%.1f%%)", end, *totalPlayers, progress)
	}
	fmt.Printf("\n✓ Created %d players\n\n", *totalPlayers)

	if *initialOnly {
		fmt.Println("Initial-only mode: Exiting after creating players")
		close(done)
		producer.AsyncClose()
		wg.Wait()
		fmt.Printf("\n✓ Completed. Sent: %d, Errors: %d\n", atomic.LoadInt64(&successCount), atomic.LoadInt64(&errorCount))
		return
	}

	// Start continuous updates
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("Starting continuous updates (%d/sec)\n", *updatesPerSecond)
	fmt.Println("Top players have 70% chance to be updated (to create movement)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Println()
	fmt.Println("Press Ctrl+C to stop")
	fmt.Println()

	interval := time.Second / time.Duration(*updatesPerSecond)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	statsTicker := time.NewTicker(5 * time.Second)
	defer statsTicker.Stop()

	var endTime time.Time
	if *duration > 0 {
		endTime = time.Now().Add(*duration)
	}

	var updateCount int64

	for {
		select {
		case <-sigChan:
			fmt.Println("\n\nShutting down...")
			close(done)
			producer.AsyncClose()
			wg.Wait()
			fmt.Printf("\n✓ Completed. Sent: %d, Errors: %d\n", atomic.LoadInt64(&successCount), atomic.LoadInt64(&errorCount))
			return

		case <-ticker.C:
			if *duration > 0 && time.Now().After(endTime) {
				fmt.Println("\n\nDuration reached, shutting down...")
				close(done)
				producer.AsyncClose()
				wg.Wait()
				fmt.Printf("\n✓ Completed. Sent: %d, Errors: %d\n", atomic.LoadInt64(&successCount), atomic.LoadInt64(&errorCount))
				return
			}

			// 70% chance to pick from top 20 players
			var playerIdx int
			if rand.Intn(100) < 70 {
				playerIdx = rand.Intn(20)
			} else {
				playerIdx = rand.Intn(*totalPlayers-20) + 20
			}

			// Score based on player position
			var score int64
			if playerIdx < 10 {
				score = int64(rand.Intn(800) + 400)
			} else if playerIdx < 50 {
				score = int64(rand.Intn(600) + 300)
			} else {
				score = int64(rand.Intn(400) + 200)
			}

			submission := ScoreSubmission{
				PlayerID:      getPlayerName(playerIdx),
				LeaderboardID: *leaderboardID,
				Score:         score,
			}
			sendMessage(submission)
			atomic.AddInt64(&updateCount, 1)

		case <-statsTicker.C:
			updates := atomic.LoadInt64(&updateCount)
			success := atomic.LoadInt64(&successCount)
			errors := atomic.LoadInt64(&errorCount)
			fmt.Printf("[%s] Updates: %d | Sent: %d | Errors: %d\n",
				time.Now().Format("15:04:05"),
				updates,
				success,
				errors,
			)
		}
	}
}
