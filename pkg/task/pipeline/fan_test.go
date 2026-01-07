package pipeline

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

// Complex data structures for B and C
type UserData struct {
	ID   int
	Name string
}

type OrderData struct {
	ID    int
	Total float64
}

// Final structure for D
type UserReport struct {
	UserName   string
	OrderTotal float64
}

func TestComplexFanOutFanIn(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Stage A: Initial source (Request IDs)
	source := make(chan int, 5)
	go func() {
		for i := 1; i <= 3; i++ {
			source <- i
		}
		close(source)
	}()

	// --- Fan-Out ---
	// Split stream A into two streams for B and C
	chanB := make(chan int, 5)
	chanC := make(chan int, 5)

	go func() {
		defer close(chanB)
		defer close(chanC)
		for id := range source {
			chanB <- id
			chanC <- id
		}
	}()

	// Stage B: Fetch User Data (Concurrency 2)
	pipelineB := Next(Start(ctx, chanB), 2, 5, func(ctx context.Context, id int) (UserData, error) {
		return UserData{ID: id, Name: fmt.Sprintf("User-%d", id)}, nil
	})

	// Stage C: Fetch Order Data (Concurrency 2)
	pipelineC := Next(Start(ctx, chanC), 2, 5, func(ctx context.Context, id int) (OrderData, error) {
		return OrderData{ID: id, Total: float64(id) * 10.5}, nil
	})

	// --- Complex Business Logic Merging (Fan-In) ---
	// We collect results from B and C, then combine them.
	// In a real scenario, you might use a map to match IDs if they arrive out of order.
	reportCh := make(chan UserReport, 5)
	var wg sync.WaitGroup
	wg.Add(2)

	// In-memory "Business Layer" state to match results by ID
	var mu sync.Mutex
	users := make(map[int]UserData)
	orders := make(map[int]OrderData)

	processReport := func(id int) {
		mu.Lock()
		defer mu.Unlock()
		u, uOk := users[id]
		o, oOk := orders[id]
		if uOk && oOk {
			reportCh <- UserReport{
				UserName:   u.Name,
				OrderTotal: o.Total,
			}
		}
	}

	// Consumer for B
	go func() {
		defer wg.Done()
		for user := range pipelineB.source {
			mu.Lock()
			users[user.ID] = user
			mu.Unlock()
			processReport(user.ID)
		}
	}()

	// Consumer for C
	go func() {
		defer wg.Done()
		for order := range pipelineC.source {
			mu.Lock()
			orders[order.ID] = order
			mu.Unlock()
			processReport(order.ID)
		}
	}()

	go func() {
		wg.Wait()
		close(reportCh)
	}()

	// Stage D: Process the final UserReport
	err := Start(ctx, reportCh).Wait(func(report UserReport) error {
		fmt.Printf("Stage D processing report: User=%s, Total=$%.2f\n", report.UserName, report.OrderTotal)
		return nil
	})

	if err != nil {
		t.Errorf("Pipeline failed: %v", err)
	}
}

